package scanner

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/fatih/color"
	"golang.org/x/net/proxy"
)

const (
	xrayBufferSize        = 32768
	xrayDownloadURL       = "https://speed.cloudflare.com/__down?bytes=52428800"
	xrayDownloadTimeout   = 15 * time.Second
	xrayTestNum           = 10
	xrayPort              = 443
	xraySocksReadyTimeout = 8 * time.Second
	xrayPingTimes         = 1
	xrayPingTimeout       = 10 * time.Second
	xrayURLConfigPath     = "./config/xray_config.txt"
	xrayJSONConfigPath    = "./config/xray_config.json"
	xrayKillSleep         = 200 * time.Millisecond
)

type xraySocksInfo struct {
	Address string
	Port    int
	User    string
	Pass    string
}

type xrayFailReason struct {
	ip     string
	reason string
}

var xrayDiagMu sync.Mutex
var xrayDiagStartupTimeout int
var xrayDiagHTTPError int
var xrayDiagWrongStatus int
var xrayDiagConfigError int
var xrayDiagFirstStderrSet bool
var xrayDiagFirstStderr string
var xrayFirstFailures []xrayFailReason

var allowedStreamFields = map[string]bool{
	"network": true, "security": true, "tlsSettings": true,
	"realitySettings": true, "wsSettings": true, "grpcSettings": true,
	"tcpSettings": true, "httpSettings": true, "quicSettings": true,
	"dsSettings": true, "httpupgradeSettings": true, "splithttpSettings": true,
	"sockopt": true,
}

var urlPlaceholders = []string{
	"your-uuid", "your-server", "your-domain",
	"example.com", "ip_placeholder", "your-password", "your-id",
}

var jsonPlaceholders = []string{
	"your-uuid-here", "ip_placeholder", "your-domain.com",
}

func resetXrayDiag() {
	xrayDiagMu.Lock()
	xrayDiagStartupTimeout = 0
	xrayDiagHTTPError = 0
	xrayDiagWrongStatus = 0
	xrayDiagConfigError = 0
	xrayDiagFirstStderr = ""
	xrayDiagFirstStderrSet = false
	xrayFirstFailures = nil
	xrayDiagMu.Unlock()
}

func recordDiag(field string, ipStr string, detail string) {
	xrayDiagMu.Lock()
	switch field {
	case "startupTimeout":
		xrayDiagStartupTimeout++
	case "httpError":
		xrayDiagHTTPError++
	case "wrongStatus":
		xrayDiagWrongStatus++
	case "configError":
		xrayDiagConfigError++
	}
	if detail != "" && !xrayDiagFirstStderrSet {
		xrayDiagFirstStderr = detail
		xrayDiagFirstStderrSet = true
	}
	if ipStr != "" && len(xrayFirstFailures) < 3 {
		reason := field
		if detail != "" {
			reason = detail
		}
		xrayFirstFailures = append(xrayFirstFailures, xrayFailReason{ip: ipStr, reason: reason})
	}
	xrayDiagMu.Unlock()
}

func GetXrayDiagInfo() string {
	xrayDiagMu.Lock()
	defer xrayDiagMu.Unlock()
	total := xrayDiagStartupTimeout + xrayDiagHTTPError + xrayDiagWrongStatus + xrayDiagConfigError
	if total == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n========== Xray Diagnostic Report ==========\n")
	if xrayDiagStartupTimeout > 0 {
		sb.WriteString(fmt.Sprintf("  Core startup timeouts  : %d\n", xrayDiagStartupTimeout))
		sb.WriteString("    Core port did not bind in time. Device may be under load.\n")
	}
	if xrayDiagHTTPError > 0 {
		sb.WriteString(fmt.Sprintf("  Tunnel connection errors: %d\n", xrayDiagHTTPError))
		sb.WriteString("    Core started but tunnel could not reach the remote server.\n")
		sb.WriteString("    This is normal if those IPs are filtered on your network.\n")
	}
	if xrayDiagWrongStatus > 0 {
		sb.WriteString(fmt.Sprintf("  Unexpected HTTP status  : %d\n", xrayDiagWrongStatus))
	}
	if xrayDiagConfigError > 0 {
		sb.WriteString(fmt.Sprintf("  Config build errors     : %d\n", xrayDiagConfigError))
	}
	if len(xrayFirstFailures) > 0 {
		sb.WriteString("\n  First few failures:\n")
		for _, f := range xrayFirstFailures {
			sb.WriteString(fmt.Sprintf("    %-20s %s\n", f.ip, f.reason))
		}
	}
	if xrayDiagStartupTimeout > 0 {
		sb.WriteString("\n  NOTE: Core startup timeouts may indicate the device is too slow.\n")
		sb.WriteString("  Try closing background apps and running again.\n")
	}
	sb.WriteString("=============================================\n")
	return sb.String()
}

func findFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("failed to find free port: %v", err)
	}
	p := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return p, nil
}

func waitForSocksReady(p int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", p), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("core not ready on port %d after %v", p, timeout)
}

func cleanStreamSettings(ss map[string]interface{}) map[string]interface{} {
	clean := make(map[string]interface{})
	for k, v := range ss {
		if allowedStreamFields[k] {
			clean[k] = v
		}
	}
	return clean
}

func getDialerProxy(outMap map[string]interface{}) string {
	ss, ok := outMap["streamSettings"].(map[string]interface{})
	if !ok {
		return ""
	}
	sockopt, ok := ss["sockopt"].(map[string]interface{})
	if !ok {
		return ""
	}
	dp, _ := sockopt["dialerProxy"].(string)
	return dp
}

func base64DecodeAny(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	padded := s
	switch len(padded) % 4 {
	case 2:
		padded += "=="
	case 3:
		padded += "="
	}
	if b, err := base64.StdEncoding.DecodeString(padded); err == nil {
		return b, nil
	}
	normalized := strings.NewReplacer("-", "+", "_", "/").Replace(padded)
	if b, err := base64.StdEncoding.DecodeString(normalized); err == nil {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return nil, fmt.Errorf("cannot base64 decode input")
}

func validateURLConfig(rawURL string) error {
	parts := strings.SplitN(rawURL, "://", 2)
	if len(parts) != 2 || parts[1] == "" {
		return fmt.Errorf("not a valid proxy URL format")
	}
	scheme := strings.ToLower(parts[0])
	validSchemes := map[string]bool{"vless": true, "vmess": true, "trojan": true, "ss": true, "shadowsocks": true}
	if !validSchemes[scheme] {
		return fmt.Errorf("unsupported protocol '%s' — supported: vless, vmess, trojan, ss", scheme)
	}
	lowerURL := strings.ToLower(rawURL)
	for _, p := range urlPlaceholders {
		if strings.Contains(lowerURL, strings.ToLower(p)) {
			return fmt.Errorf("config contains placeholder value '%s' — please replace with your real config", p)
		}
	}
	switch scheme {
	case "vmess":
		encoded := strings.TrimPrefix(rawURL, "vmess://")
		if idx := strings.Index(encoded, "#"); idx != -1 {
			encoded = encoded[:idx]
		}
		decoded, err := base64DecodeAny(encoded)
		if err != nil {
			return fmt.Errorf("invalid vmess URL: cannot decode base64 content")
		}
		var v map[string]interface{}
		if err := json.Unmarshal(decoded, &v); err != nil {
			return fmt.Errorf("invalid vmess URL: cannot parse inner JSON")
		}
		if id, _ := v["id"].(string); id == "" {
			return fmt.Errorf("vmess config missing 'id' field")
		}
		if add, _ := v["add"].(string); add == "" {
			return fmt.Errorf("vmess config missing server address ('add' field)")
		}
	default:
		u, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("cannot parse URL: %v", err)
		}
		if u.User.Username() == "" {
			return fmt.Errorf("config missing credentials (uuid or password)")
		}
		if u.Hostname() == "" {
			return fmt.Errorf("config missing server address")
		}
	}
	return nil
}

func validateJSONConfig(content string) error {
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(content), &cfg); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}
	inboundsRaw, ok := cfg["inbounds"]
	if !ok {
		return fmt.Errorf("config missing 'inbounds' field")
	}
	inbounds, ok := inboundsRaw.([]interface{})
	if !ok || len(inbounds) == 0 {
		return fmt.Errorf("'inbounds' must be a non-empty array")
	}
	hasSocks := false
	for _, in := range inbounds {
		inMap, ok := in.(map[string]interface{})
		if !ok {
			continue
		}
		proto, _ := inMap["protocol"].(string)
		if strings.ToLower(proto) == "socks" {
			hasSocks = true
			break
		}
	}
	if !hasSocks {
		return fmt.Errorf("config has no SOCKS inbound — please add a socks inbound")
	}
	outboundsRaw, ok := cfg["outbounds"]
	if !ok {
		return fmt.Errorf("config missing 'outbounds' field")
	}
	outbounds, ok := outboundsRaw.([]interface{})
	if !ok || len(outbounds) == 0 {
		return fmt.Errorf("'outbounds' must be a non-empty array")
	}
	skipProtos := map[string]bool{"freedom": true, "blackhole": true, "dns": true}
	var proxyOut map[string]interface{}
	for _, out := range outbounds {
		outMap, ok := out.(map[string]interface{})
		if !ok {
			continue
		}
		proto, _ := outMap["protocol"].(string)
		if !skipProtos[strings.ToLower(proto)] && proto != "" {
			proxyOut = outMap
			break
		}
	}
	if proxyOut == nil {
		return fmt.Errorf("config has no proxy outbound — add a vless, vmess, trojan or shadowsocks outbound")
	}
	cfgBytes, _ := json.Marshal(cfg)
	cfgStr := string(cfgBytes)
	for _, p := range jsonPlaceholders {
		if strings.Contains(cfgStr, p) {
			return fmt.Errorf("config contains placeholder value '%s' — please replace with your real config", p)
		}
	}
	return nil
}

func ValidateXrayConfig() error {
	if data, err := os.ReadFile(xrayURLConfigPath); err == nil {
		content := strings.TrimSpace(string(data))
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if err := validateURLConfig(line); err != nil {
				return fmt.Errorf("invalid URL config in xray_config.txt: %v", err)
			}
			return nil
		}
	}
	data, err := os.ReadFile(xrayJSONConfigPath)
	if err != nil {
		return fmt.Errorf("no config found — please edit config/xray_config.txt (URL) or config/xray_config.json (JSON)")
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return fmt.Errorf("config/xray_config.json is empty — please add your config")
	}
	if err := validateJSONConfig(content); err != nil {
		return fmt.Errorf("invalid JSON config in xray_config.json: %v", err)
	}
	return nil
}

func readXrayConfig() (string, bool, error) {
	if data, err := os.ReadFile(xrayURLConfigPath); err == nil {
		content := strings.TrimSpace(string(data))
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			return line, true, nil
		}
	}
	data, err := os.ReadFile(xrayJSONConfigPath)
	if err != nil {
		return "", false, fmt.Errorf("no config found: checked %s and %s", xrayURLConfigPath, xrayJSONConfigPath)
	}
	return strings.TrimSpace(string(data)), false, nil
}

func buildStreamSettings(network, security, sni, fp, path, headerHost string, allowInsecure bool, pbk, sid, spx string) map[string]interface{} {
	if network == "" {
		network = "tcp"
	}
	ss := map[string]interface{}{"network": network}
	switch strings.ToLower(security) {
	case "tls":
		tlsSettings := map[string]interface{}{"allowInsecure": allowInsecure}
		if sni != "" {
			tlsSettings["serverName"] = sni
		}
		if fp != "" {
			tlsSettings["fingerprint"] = fp
		}
		ss["security"] = "tls"
		ss["tlsSettings"] = tlsSettings
	case "reality":
		realitySettings := map[string]interface{}{"show": false}
		if sni != "" {
			realitySettings["serverName"] = sni
		}
		if fp != "" {
			realitySettings["fingerprint"] = fp
		}
		if pbk != "" {
			realitySettings["publicKey"] = pbk
		}
		if sid != "" {
			realitySettings["shortId"] = sid
		}
		if spx != "" {
			realitySettings["spiderX"] = spx
		}
		ss["security"] = "reality"
		ss["realitySettings"] = realitySettings
	default:
		ss["security"] = "none"
	}
	switch strings.ToLower(network) {
	case "ws":
		wsSettings := map[string]interface{}{}
		if path != "" {
			wsSettings["path"] = path
		}
		if headerHost != "" {
			wsSettings["headers"] = map[string]interface{}{"Host": headerHost}
		}
		ss["wsSettings"] = wsSettings
	case "grpc":
		grpcSettings := map[string]interface{}{}
		if path != "" {
			grpcSettings["serviceName"] = path
		}
		ss["grpcSettings"] = grpcSettings
	case "http", "h2":
		httpSettings := map[string]interface{}{}
		if path != "" {
			httpSettings["path"] = path
		}
		if headerHost != "" {
			httpSettings["host"] = []interface{}{headerHost}
		}
		ss["network"] = "http"
		ss["httpSettings"] = httpSettings
	case "httpupgrade":
		httpUpSettings := map[string]interface{}{}
		if path != "" {
			httpUpSettings["path"] = path
		}
		if headerHost != "" {
			httpUpSettings["host"] = headerHost
		}
		ss["httpupgradeSettings"] = httpUpSettings
	case "splithttp":
		splitSettings := map[string]interface{}{}
		if path != "" {
			splitSettings["path"] = path
		}
		if headerHost != "" {
			splitSettings["host"] = headerHost
		}
		ss["splithttpSettings"] = splitSettings
	}
	return ss
}

func parseVlessURL(rawURL string, scanIP string) (map[string]interface{}, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid VLESS URL: %v", err)
	}
	uuid := u.User.Username()
	if uuid == "" {
		return nil, fmt.Errorf("config missing UUID")
	}
	p := xrayPort
	if ps := u.Port(); ps != "" {
		if pInt, e := strconv.Atoi(ps); e == nil {
			p = pInt
		}
	}
	q := u.Query()
	network := q.Get("type")
	if network == "" {
		network = "tcp"
	}
	security := q.Get("security")
	sni := q.Get("sni")
	if sni == "" {
		sni = q.Get("peer")
	}
	fp := q.Get("fp")
	path, _ := url.QueryUnescape(q.Get("path"))
	headerHost := q.Get("host")
	flow := q.Get("flow")
	allowInsecure := q.Get("allowInsecure") == "1" || q.Get("insecure") == "1"
	pbk := q.Get("pbk")
	sid := q.Get("sid")
	spx := q.Get("spx")
	user := map[string]interface{}{
		"id": uuid, "encryption": "none", "level": float64(8),
	}
	if flow != "" {
		user["flow"] = flow
	}
	settings := map[string]interface{}{
		"vnext": []interface{}{
			map[string]interface{}{
				"address": scanIP, "port": float64(p),
				"users": []interface{}{user},
			},
		},
	}
	return map[string]interface{}{
		"protocol":       "vless",
		"settings":       settings,
		"streamSettings": buildStreamSettings(network, security, sni, fp, path, headerHost, allowInsecure, pbk, sid, spx),
		"tag":            "proxy",
	}, nil
}

func parseVmessURL(rawURL string, scanIP string) (map[string]interface{}, error) {
	encoded := strings.TrimPrefix(rawURL, "vmess://")
	if idx := strings.Index(encoded, "#"); idx != -1 {
		encoded = encoded[:idx]
	}
	decoded, err := base64DecodeAny(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config: %v", err)
	}
	var v map[string]interface{}
	if err := json.Unmarshal(decoded, &v); err != nil {
		return nil, fmt.Errorf("invalid config format: %v", err)
	}
	p := xrayPort
	switch pv := v["port"].(type) {
	case float64:
		p = int(pv)
	case string:
		if pInt, e := strconv.Atoi(pv); e == nil {
			p = pInt
		}
	}
	id, _ := v["id"].(string)
	aid := 0
	switch a := v["aid"].(type) {
	case float64:
		aid = int(a)
	case string:
		if aInt, e := strconv.Atoi(a); e == nil {
			aid = aInt
		}
	}
	security, _ := v["scy"].(string)
	if security == "" {
		security = "auto"
	}
	network, _ := v["net"].(string)
	if network == "" {
		network = "tcp"
	}
	tlsSecurity, _ := v["tls"].(string)
	sni, _ := v["sni"].(string)
	fp, _ := v["fp"].(string)
	path, _ := v["path"].(string)
	headerHost, _ := v["host"].(string)
	settings := map[string]interface{}{
		"vnext": []interface{}{
			map[string]interface{}{
				"address": scanIP, "port": float64(p),
				"users": []interface{}{
					map[string]interface{}{
						"id": id, "alterId": float64(aid),
						"security": security, "level": float64(8),
					},
				},
			},
		},
	}
	return map[string]interface{}{
		"protocol":       "vmess",
		"settings":       settings,
		"streamSettings": buildStreamSettings(network, tlsSecurity, sni, fp, path, headerHost, false, "", "", ""),
		"tag":            "proxy",
	}, nil
}

func parseTrojanURL(rawURL string, scanIP string) (map[string]interface{}, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid config URL: %v", err)
	}
	password := u.User.Username()
	if password == "" {
		return nil, fmt.Errorf("config missing password")
	}
	p := xrayPort
	if ps := u.Port(); ps != "" {
		if pInt, e := strconv.Atoi(ps); e == nil {
			p = pInt
		}
	}
	q := u.Query()
	network := q.Get("type")
	if network == "" {
		network = "tcp"
	}
	security := q.Get("security")
	if security == "" {
		security = "tls"
	}
	sni := q.Get("sni")
	if sni == "" {
		sni = q.Get("peer")
	}
	fp := q.Get("fp")
	path, _ := url.QueryUnescape(q.Get("path"))
	headerHost := q.Get("host")
	allowInsecure := q.Get("allowInsecure") == "1" || q.Get("insecure") == "1"
	pbk := q.Get("pbk")
	sid := q.Get("sid")
	spx := q.Get("spx")
	settings := map[string]interface{}{
		"servers": []interface{}{
			map[string]interface{}{
				"address": scanIP, "port": float64(p),
				"password": password, "level": float64(8),
			},
		},
	}
	return map[string]interface{}{
		"protocol":       "trojan",
		"settings":       settings,
		"streamSettings": buildStreamSettings(network, security, sni, fp, path, headerHost, allowInsecure, pbk, sid, spx),
		"tag":            "proxy",
	}, nil
}

func parseSSURL(rawURL string, scanIP string) (map[string]interface{}, error) {
	var method, password string
	p := xrayPort
	u, parseErr := url.Parse(rawURL)
	if parseErr == nil && u.Host != "" {
		userInfo := u.User.String()
		if decoded, err := base64DecodeAny(userInfo); err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				method = parts[0]
				password = parts[1]
			}
		} else {
			parts := strings.SplitN(userInfo, ":", 2)
			if len(parts) == 2 {
				method = parts[0]
				password = parts[1]
			}
		}
		if ps := u.Port(); ps != "" {
			if pInt, e := strconv.Atoi(ps); e == nil {
				p = pInt
			}
		}
	} else {
		encoded := strings.TrimPrefix(rawURL, "ss://")
		if idx := strings.Index(encoded, "#"); idx != -1 {
			encoded = encoded[:idx]
		}
		decoded, err := base64DecodeAny(encoded)
		if err != nil {
			return nil, fmt.Errorf("failed to decode config: %v", err)
		}
		decodedStr := string(decoded)
		atIdx := strings.LastIndex(decodedStr, "@")
		if atIdx == -1 {
			return nil, fmt.Errorf("invalid config format: missing @")
		}
		userPart := decodedStr[:atIdx]
		hostPart := decodedStr[atIdx+1:]
		parts := strings.SplitN(userPart, ":", 2)
		if len(parts) == 2 {
			method = parts[0]
			password = parts[1]
		}
		hostParts := strings.SplitN(hostPart, ":", 2)
		if len(hostParts) == 2 {
			if pInt, e := strconv.Atoi(hostParts[1]); e == nil {
				p = pInt
			}
		}
	}
	if method == "" {
		return nil, fmt.Errorf("config: could not parse method/password")
	}
	settings := map[string]interface{}{
		"servers": []interface{}{
			map[string]interface{}{
				"address": scanIP, "port": float64(p),
				"method": method, "password": password, "level": float64(8),
			},
		},
	}
	return map[string]interface{}{
		"protocol": "shadowsocks", "settings": settings, "tag": "proxy",
	}, nil
}

func buildBaseConfig(inbound, outbound map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"log": map[string]interface{}{"loglevel": "none"},
		"inbounds": []interface{}{inbound},
		"outbounds": []interface{}{
			outbound,
			map[string]interface{}{"protocol": "freedom", "settings": map[string]interface{}{}, "tag": "direct"},
			map[string]interface{}{"protocol": "blackhole", "settings": map[string]interface{}{"response": map[string]interface{}{"type": "http"}}, "tag": "block"},
		},
		"routing": map[string]interface{}{
			"domainStrategy": "AsIs",
			"rules":          []interface{}{map[string]interface{}{"type": "field", "outboundTag": "proxy", "network": "tcp,udp"}},
		},
	}
}

func writeTempConfig(cfg map[string]interface{}) (string, error) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %v", err)
	}
	tempFile, err := os.CreateTemp("", "xray_cfg_*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write temp config: %v", err)
	}
	tempFile.Close()
	return tempFile.Name(), nil
}

func buildConfigFromURL(rawURL string, scanIP string, socksPort int) (string, *xraySocksInfo, error) {
	socksInfo := &xraySocksInfo{Address: "127.0.0.1", Port: socksPort}
	inbound := map[string]interface{}{
		"protocol": "socks", "listen": "127.0.0.1", "port": float64(socksPort),
		"settings": map[string]interface{}{"auth": "noauth", "udp": false},
	}
	scheme := strings.ToLower(strings.SplitN(rawURL, "://", 2)[0])
	var (
		outbound map[string]interface{}
		err      error
	)
	switch scheme {
	case "vless":
		outbound, err = parseVlessURL(rawURL, scanIP)
	case "vmess":
		outbound, err = parseVmessURL(rawURL, scanIP)
	case "trojan":
		outbound, err = parseTrojanURL(rawURL, scanIP)
	case "ss", "shadowsocks":
		outbound, err = parseSSURL(rawURL, scanIP)
	default:
		return "", nil, fmt.Errorf("unsupported scheme: %s", scheme)
	}
	if err != nil {
		return "", nil, err
	}
	fp, writeErr := writeTempConfig(buildBaseConfig(inbound, outbound))
	if writeErr != nil {
		return "", nil, writeErr
	}
	return fp, socksInfo, nil
}

func createTempConfigWithIP(ip string, socksPort int) (string, *xraySocksInfo, error) {
	content, isURL, err := readXrayConfig()
	if err != nil {
		return "", nil, err
	}
	if isURL {
		return buildConfigFromURL(content, ip, socksPort)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(content), &cfg); err != nil {
		return "", nil, fmt.Errorf("invalid JSON in config: %v", err)
	}
	inboundsRaw, ok := cfg["inbounds"]
	if !ok {
		return "", nil, fmt.Errorf("no 'inbounds' field in config")
	}
	inboundsSlice, ok := inboundsRaw.([]interface{})
	if !ok {
		return "", nil, fmt.Errorf("'inbounds' is not an array")
	}
	socksInfo := &xraySocksInfo{Address: "127.0.0.1", Port: socksPort}
	var newInbounds []interface{}
	for _, in := range inboundsSlice {
		inMap, ok := in.(map[string]interface{})
		if !ok {
			continue
		}
		protocol, _ := inMap["protocol"].(string)
		if strings.ToLower(protocol) != "socks" {
			continue
		}
		cleanInbound := map[string]interface{}{
			"protocol": "socks", "listen": "127.0.0.1", "port": float64(socksPort),
			"settings": map[string]interface{}{"auth": "noauth", "udp": false},
		}
		if listen, ok := inMap["listen"].(string); ok && listen != "" {
			cleanInbound["listen"] = listen
			socksInfo.Address = listen
		}
		if settings, ok := inMap["settings"].(map[string]interface{}); ok {
			if auth, _ := settings["auth"].(string); auth == "password" {
				if accounts, ok := settings["accounts"].([]interface{}); ok && len(accounts) > 0 {
					if acc, ok := accounts[0].(map[string]interface{}); ok {
						user, _ := acc["user"].(string)
						pass, _ := acc["pass"].(string)
						if user != "" && pass != "" {
							socksInfo.User = user
							socksInfo.Pass = pass
							cleanInbound["settings"] = map[string]interface{}{
								"auth": "password", "udp": false,
								"accounts": []interface{}{map[string]interface{}{"user": user, "pass": pass}},
							}
						}
					}
				}
			}
		}
		newInbounds = append(newInbounds, cleanInbound)
		break
	}
	if len(newInbounds) == 0 {
		return "", nil, fmt.Errorf("no SOCKS inbound found in config")
	}
	outboundsRaw, ok := cfg["outbounds"]
	if !ok {
		return "", nil, fmt.Errorf("no 'outbounds' field in config")
	}
	outboundsSlice, ok := outboundsRaw.([]interface{})
	if !ok {
		return "", nil, fmt.Errorf("'outbounds' is not an array")
	}
	skipProtocols := map[string]bool{"freedom": true, "blackhole": true, "dns": true}
	var proxyOutbound map[string]interface{}
	outboundsByTag := make(map[string]map[string]interface{})
	for _, out := range outboundsSlice {
		outMap, ok := out.(map[string]interface{})
		if !ok {
			continue
		}
		tag, _ := outMap["tag"].(string)
		if tag != "" {
			outboundsByTag[tag] = outMap
		}
		protocol, _ := outMap["protocol"].(string)
		if !skipProtocols[strings.ToLower(protocol)] && proxyOutbound == nil {
			proxyOutbound = outMap
		}
	}
	if proxyOutbound == nil {
		return "", nil, fmt.Errorf("no supported proxy outbound found in config")
	}
	protocol, _ := proxyOutbound["protocol"].(string)
	settings, ok := proxyOutbound["settings"].(map[string]interface{})
	if !ok {
		return "", nil, fmt.Errorf("proxy outbound has no 'settings' field")
	}
	ipUpdated := false
	switch strings.ToLower(protocol) {
	case "vless", "vmess":
		vnextRaw, ok := settings["vnext"]
		if !ok {
			return "", nil, fmt.Errorf("outbound missing 'vnext'")
		}
		vnextSlice, ok := vnextRaw.([]interface{})
		if !ok || len(vnextSlice) == 0 {
			return "", nil, fmt.Errorf("outbound 'vnext' is empty")
		}
		server, ok := vnextSlice[0].(map[string]interface{})
		if !ok {
			return "", nil, fmt.Errorf("outbound server entry is invalid")
		}
		server["address"] = ip
		vnextSlice[0] = server
		settings["vnext"] = vnextSlice
		ipUpdated = true
	case "trojan", "shadowsocks":
		serversRaw, ok := settings["servers"]
		if !ok {
			return "", nil, fmt.Errorf("outbound missing 'servers'")
		}
		serversSlice, ok := serversRaw.([]interface{})
		if !ok || len(serversSlice) == 0 {
			return "", nil, fmt.Errorf("outbound 'servers' is empty")
		}
		server, ok := serversSlice[0].(map[string]interface{})
		if !ok {
			return "", nil, fmt.Errorf("outbound server entry is invalid")
		}
		server["address"] = ip
		serversSlice[0] = server
		settings["servers"] = serversSlice
		ipUpdated = true
	}
	if !ipUpdated {
		return "", nil, fmt.Errorf("unsupported proxy protocol: %s", protocol)
	}
	cleanedProxy := map[string]interface{}{
		"protocol": proxyOutbound["protocol"], "settings": settings, "tag": "proxy",
	}
	var dialerProxyTag string
	if ss, ok := proxyOutbound["streamSettings"].(map[string]interface{}); ok {
		cleanedSS := cleanStreamSettings(ss)
		cleanedProxy["streamSettings"] = cleanedSS
		dialerProxyTag = getDialerProxy(cleanedProxy)
	}
	if mux, ok := proxyOutbound["mux"].(map[string]interface{}); ok {
		if enabled, _ := mux["enabled"].(bool); !enabled {
			cleanedProxy["mux"] = map[string]interface{}{"enabled": false}
		}
	}
	newOutbounds := []interface{}{
		cleanedProxy,
		map[string]interface{}{"protocol": "freedom", "settings": map[string]interface{}{}, "tag": "direct"},
		map[string]interface{}{"protocol": "blackhole", "settings": map[string]interface{}{"response": map[string]interface{}{"type": "http"}}, "tag": "block"},
	}
	if dialerProxyTag != "" {
		if refOut, found := outboundsByTag[dialerProxyTag]; found {
			cleanRef := map[string]interface{}{"protocol": refOut["protocol"], "tag": dialerProxyTag}
			if refSettings, ok := refOut["settings"].(map[string]interface{}); ok {
				cleanRef["settings"] = refSettings
			}
			if refSS, ok := refOut["streamSettings"].(map[string]interface{}); ok {
				cleanRef["streamSettings"] = cleanStreamSettings(refSS)
			}
			newOutbounds = append(newOutbounds, cleanRef)
		}
	}
	cleanCfg := map[string]interface{}{
		"log": map[string]interface{}{"loglevel": "none"},
		"inbounds": newInbounds, "outbounds": newOutbounds,
		"routing": map[string]interface{}{
			"domainStrategy": "AsIs",
			"rules":          []interface{}{map[string]interface{}{"type": "field", "outboundTag": "proxy", "network": "tcp,udp"}},
		},
	}
	filePath, writeErr := writeTempConfig(cleanCfg)
	if writeErr != nil {
		return "", nil, writeErr
	}
	return filePath, socksInfo, nil
}

func createSocksDialer(socksInfo *xraySocksInfo) (proxy.Dialer, error) {
	addr := fmt.Sprintf("%s:%d", socksInfo.Address, socksInfo.Port)
	if socksInfo.User != "" && socksInfo.Pass != "" {
		auth := proxy.Auth{User: socksInfo.User, Password: socksInfo.Pass}
		return proxy.SOCKS5("tcp", addr, &auth, proxy.Direct)
	}
	return proxy.SOCKS5("tcp", addr, nil, proxy.Direct)
}

func makeTestHTTPClient(socksInfo *xraySocksInfo, timeout time.Duration) (*http.Client, error) {
	dialer, err := createSocksDialer(socksInfo)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			DisableKeepAlives:     true,
		},
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
}

func SelfTestXray() error {
	testPort, err := findFreePort()
	if err != nil {
		return fmt.Errorf("Failed to find free port for self-test: %v", err)
	}
	cfg := map[string]interface{}{
		"log":      map[string]interface{}{"loglevel": "warning"},
		"inbounds": []interface{}{map[string]interface{}{"listen": "127.0.0.1", "port": float64(testPort), "protocol": "socks", "settings": map[string]interface{}{"udp": false}}},
		"outbounds": []interface{}{map[string]interface{}{"protocol": "freedom"}},
	}
	cfgPath, err := writeTempConfig(cfg)
	if err != nil {
		return fmt.Errorf("Failed to write self-test config: %v", err)
	}
	defer os.Remove(cfgPath)
	cmd := exec.Command("./xray/xray", "run", "-c", cfgPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Failed to start core binary. Make sure it exists at ./xray/xray and is executable.\nError: %v", err)
	}
	err = waitForSocksReady(testPort, 6*time.Second)
	if cmd.Process != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		cmd.Wait()
	}
	if err != nil {
		errOutput := strings.TrimSpace(out.String())
		if errOutput != "" {
			return fmt.Errorf("Core failed to bind port within 6 seconds.\nOutput:\n%s", errOutput)
		}
		return fmt.Errorf("Core failed to bind port within 6 seconds.\nTry restarting the terminal and running again.")
	}
	return nil
}

func testIPViaXray(ip *net.IPAddr) (recv int, totalDelay time.Duration) {
	socksPort, err := findFreePort()
	if err != nil {
		recordDiag("configError", ip.String(), "no free port available")
		return
	}
	configPath, socksInfo, err := createTempConfigWithIP(ip.String(), socksPort)
	if err != nil {
		recordDiag("configError", ip.String(), fmt.Sprintf("build config: %v", err))
		return
	}
	defer os.Remove(configPath)
	cmd := exec.Command("./xray/xray", "run", "-c", configPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var stderrBuf bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		recordDiag("configError", ip.String(), fmt.Sprintf("start core: %v", err))
		return
	}
	defer func() {
		if cmd.Process != nil {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			cmd.Wait()
		}
		time.Sleep(xrayKillSleep)
	}()
	if err := waitForSocksReady(socksPort, xraySocksReadyTimeout); err != nil {
		stderrOut := strings.TrimSpace(stderrBuf.String())
		recordDiag("startupTimeout", ip.String(), stderrOut)
		return
	}
	httpClient, httpErr := makeTestHTTPClient(socksInfo, xrayPingTimeout)
	if httpErr != nil {
		recordDiag("configError", ip.String(), fmt.Sprintf("dialer: %v", httpErr))
		return
	}
	start := time.Now()
	resp, err := httpClient.Get("https://www.gstatic.com/generate_204")
	if err != nil {
		recordDiag("httpError", ip.String(), fmt.Sprintf("tunnel: %v", err))
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		recv = 1
		totalDelay = time.Since(start)
	} else {
		recordDiag("wrongStatus", ip.String(), fmt.Sprintf("status %d", resp.StatusCode))
	}
	return
}

func PingIPsViaXray(stopCh <-chan struct{}, ips []CompactIP, workers int, cp *Checkpoint, existingResults []PingResult) []PingResult {
	if _, err := os.Stat("./xray/xray"); os.IsNotExist(err) {
		color.New(color.FgRed).Println("ERROR: Core binary not found at ./xray/xray")
		return nil
	}
	resetXrayDiag()
	var results []PingResult
	var mu sync.Mutex
	total := len(ips)
	processedCount := 0
	baseIndex := 0
	if cp != nil {
		baseIndex = cp.ProgressIndex
	}
	color.New(color.FgCyan).Printf("Start latency test (Xray mode - %d workers, timeout %v per IP)\n", workers, xrayPingTimeout)
	bar := newBar(total, "Available:", "")
	ipChan := make(chan CompactIP, total)
	for _, cip := range ips {
		select {
		case <-stopCh:
		default:
			ipChan <- cip
		}
	}
	close(ipChan)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for compIP := range ipChan {
				select {
				case <-stopCh:
					return
				default:
				}
				ipAddr := compIP.ToNetIPAddr()
				recv, totalDelay := testIPViaXray(ipAddr)
				mu.Lock()
				processedCount++
				nowAble := len(results)
				if recv > 0 {
					nowAble++
					results = append(results, PingResult{
						IP:       ipAddr,
						Sended:   xrayPingTimes,
						Received: recv,
						Delay:    totalDelay,
					})
				}
				bar.grow(1, strconv.Itoa(nowAble))
				if cp != nil && processedCount%saveIntervalMode2 == 0 {
					cp.ProgressIndex = baseIndex + processedCount
					merged := make([]PingResult, 0, len(existingResults)+len(results))
					merged = append(merged, existingResults...)
					merged = append(merged, results...)
					cp.SetPingResults(merged)
					cp.Save()
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	bar.done()
	sort.Slice(results, func(i, j int) bool {
		return results[i].Delay < results[j].Delay
	})
	fmt.Println()
	diagInfo := GetXrayDiagInfo()
	if diagInfo != "" {
		color.New(color.FgYellow).Print(diagInfo)
	}
	color.New(color.FgGreen).Printf("Latency test completed (Xray): %d responsive IPs found\n\n", len(results))
	return results
}

func downloadSpeedViaXray(ip *net.IPAddr) float64 {
	socksPort, err := findFreePort()
	if err != nil {
		return 0.0
	}
	configPath, socksInfo, err := createTempConfigWithIP(ip.String(), socksPort)
	if err != nil {
		return 0.0
	}
	defer os.Remove(configPath)
	cmd := exec.Command("./xray/xray", "run", "-c", configPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return 0.0
	}
	defer func() {
		if cmd.Process != nil {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			cmd.Wait()
		}
		time.Sleep(xrayKillSleep)
	}()
	if err := waitForSocksReady(socksPort, xraySocksReadyTimeout); err != nil {
		return 0.0
	}
	httpClient, err := makeTestHTTPClient(socksInfo, xrayDownloadTimeout)
	if err != nil {
		return 0.0
	}
	req, err := http.NewRequest("GET", xrayDownloadURL, nil)
	if err != nil {
		return 0.0
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36")
	response, err := httpClient.Do(req)
	if err != nil {
		return 0.0
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return 0.0
	}
	timeStart := time.Now()
	timeEnd := timeStart.Add(xrayDownloadTimeout)
	buffer := make([]byte, xrayBufferSize)
	var contentRead int64 = 0
	var lastContentRead int64 = 0
	timeSlice := xrayDownloadTimeout / 100
	timeCounter := 1
	nextTime := timeStart.Add(timeSlice * time.Duration(timeCounter))
	e := ewma.NewMovingAverage()
	for {
		currentTime := time.Now()
		if currentTime.After(nextTime) {
			timeCounter++
			nextTime = timeStart.Add(timeSlice * time.Duration(timeCounter))
			e.Add(float64(contentRead - lastContentRead))
			lastContentRead = contentRead
		}
		if currentTime.After(timeEnd) {
			break
		}
		n, readErr := response.Body.Read(buffer)
		contentRead += int64(n)
		if readErr != nil {
			if readErr == io.EOF {
				lastSlice := timeStart.Add(timeSlice * time.Duration(timeCounter - 1))
				now := time.Now()
				elapsed := float64(now.Sub(lastSlice))
				sliceDuration := float64(timeSlice)
				if elapsed > 0 && sliceDuration > 0 {
					ratio := elapsed / sliceDuration
					if ratio > 0 {
						e.Add(float64(contentRead-lastContentRead) / ratio)
					}
				}
			}
			break
		}
	}
	avgBytesPerSec := e.Value() * 100 / xrayDownloadTimeout.Seconds()
	return avgBytesPerSec / (1024 * 1024)
}

func SpeedTestViaXray(stopCh <-chan struct{}, pingResults []PingResult) []IPResult {
	testCount := xrayTestNum
	testNum := testCount
	if len(pingResults) < testCount {
		testNum = len(pingResults)
		testCount = testNum
	}
	barPadding := "     "
	for i := 0; i < len(strconv.Itoa(len(pingResults))); i++ {
		barPadding += " "
	}
	color.New(color.FgCyan).Printf("Start download speed test (Xray mode, Number: %d, Queue: %d)\n", testCount, testNum)
	bar := newBar(testCount, barPadding, "")
	var results []IPResult
	for i := 0; i < testNum; i++ {
		select {
		case <-stopCh:
			goto done
		default:
		}
		pr := pingResults[i]
		speedMBps := downloadSpeedViaXray(pr.IP)
		bar.grow(1, "")
		results = append(results, IPResult{
			IP:            pr.IP,
			Sended:        pr.Sended,
			Received:      pr.Received,
			LossRate:      pr.GetLossRate(),
			Delay:         int(pr.Delay.Milliseconds()),
			DownloadSpeed: speedMBps * 1024 * 1024,
		})
		if len(results) == testCount {
			break
		}
	}
done:
	bar.done()
	if len(results) > 0 {
		sort.Slice(results, func(i, j int) bool {
			return results[i].DownloadSpeed > results[j].DownloadSpeed
		})
	}
	fmt.Println()
	color.New(color.FgGreen).Printf("Speed test completed (Xray): %d clean IPs found\n\n", len(results))
	return results
}