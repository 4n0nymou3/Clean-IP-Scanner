package scanner

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testVLESS = "vless://11111111-2222-3333-4444-555555555555@real-host.net:443?type=ws&security=tls&sni=real-host.net&path=%2F#Test"
const testTrojan = "trojan://secretpass@other-host.net:443?security=tls&sni=other-host.net#Test2"

const testJSON = `{
  "inbounds": [
    {
      "port": 10808,
      "protocol": "socks",
      "settings": {
        "udp": true
      }
    }
  ],
  "outbounds": [
    {
      "protocol": "vless",
      "settings": {
        "vnext": [
          {
            "address": "real-host.net",
            "port": 443,
            "users": [
              {
                "id": "11111111-2222-3333-4444-555555555555",
                "encryption": "none"
              }
            ]
          }
        ]
      }
    }
  ]
}`

const defaultRepoJSON = `{
  "inbounds": [{"port": 10808, "protocol": "socks", "settings": {"udp": true}}],
  "outbounds": [{"protocol": "freedom"}]
}`

func enterTempDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
	if err := os.MkdirAll("config", 0755); err != nil {
		t.Fatal(err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func runWizard(input string) (string, error) {
	var out bytes.Buffer
	err := EnsureXrayConfig(bufio.NewReader(strings.NewReader(input)), &out)
	return out.String(), err
}

func leftoverTempFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			found = append(found, e.Name())
		}
	}
	return found
}

func TestNoConfigIsNotValid(t *testing.T) {
	enterTempDir(t)
	if err := ValidateXrayConfig(); err == nil {
		t.Fatal("expected error when no config exists")
	}
}

func TestDefaultRepositoryJSONIsNotValid(t *testing.T) {
	enterTempDir(t)
	writeTestFile(t, xrayJSONConfigPath, defaultRepoJSON)
	if err := ValidateXrayConfig(); err == nil {
		t.Fatal("freedom-only JSON must not count as a valid config")
	}
}

func TestPlaceholderURLFileIsNotValid(t *testing.T) {
	enterTempDir(t)
	writeTestFile(t, xrayURLConfigPath, "# Xray URL Config\n# vless://your-uuid@your-server.com:443\n")
	if err := ValidateXrayConfig(); err == nil {
		t.Fatal("comment-only URL file must not count as valid")
	}
}

func TestLegacyManualURLConfigDetected(t *testing.T) {
	enterTempDir(t)
	writeTestFile(t, xrayURLConfigPath, "# header\n"+testVLESS+"\n")
	if err := ValidateXrayConfig(); err != nil {
		t.Fatalf("legacy URL config should be valid: %v", err)
	}
	if SavedXrayFormat() != XrayFormatURL {
		t.Fatalf("expected url format, got %q", SavedXrayFormat())
	}
	content, isURL, err := readXrayConfig()
	if err != nil || !isURL || content != testVLESS {
		t.Fatalf("unexpected read result: %q %v %v", content, isURL, err)
	}
}

func TestLegacyManualJSONConfigDetected(t *testing.T) {
	enterTempDir(t)
	writeTestFile(t, xrayJSONConfigPath, testJSON)
	if err := ValidateXrayConfig(); err != nil {
		t.Fatalf("legacy JSON config should be valid: %v", err)
	}
	if SavedXrayFormat() != XrayFormatJSON {
		t.Fatalf("expected json format, got %q", SavedXrayFormat())
	}
	_, isURL, err := readXrayConfig()
	if err != nil || isURL {
		t.Fatalf("expected JSON to be used: isURL=%v err=%v", isURL, err)
	}
}

func TestLegacyBothValidKeepsURLPriority(t *testing.T) {
	enterTempDir(t)
	writeTestFile(t, xrayURLConfigPath, testVLESS+"\n")
	writeTestFile(t, xrayJSONConfigPath, testJSON)
	_, isURL, err := readXrayConfig()
	if err != nil || !isURL {
		t.Fatalf("legacy priority must stay URL: isURL=%v err=%v", isURL, err)
	}
}

func TestSaveJSONSelectsJSONOverExistingURL(t *testing.T) {
	enterTempDir(t)
	original := "# header\n" + testVLESS + "\n"
	writeTestFile(t, xrayURLConfigPath, original)
	if err := SaveXrayJSONConfig(testJSON); err != nil {
		t.Fatal(err)
	}
	content, isURL, err := readXrayConfig()
	if err != nil || isURL {
		t.Fatalf("JSON must be selected: isURL=%v err=%v", isURL, err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("selected content is not the saved JSON: %v", err)
	}
	if err := ValidateXrayConfig(); err != nil {
		t.Fatal(err)
	}
	if readTestFile(t, xrayURLConfigPath) != original {
		t.Fatal("existing URL config must not be modified")
	}
}

func TestSaveURLSelectsURLOverExistingJSON(t *testing.T) {
	enterTempDir(t)
	if err := SaveXrayJSONConfig(testJSON); err != nil {
		t.Fatal(err)
	}
	jsonBefore := readTestFile(t, xrayJSONConfigPath)
	if err := SaveXrayURLConfig(testVLESS); err != nil {
		t.Fatal(err)
	}
	content, isURL, err := readXrayConfig()
	if err != nil || !isURL || content != testVLESS {
		t.Fatalf("URL must be selected: %q %v %v", content, isURL, err)
	}
	if readTestFile(t, xrayJSONConfigPath) != jsonBefore {
		t.Fatal("existing JSON config must not be modified")
	}
}

func TestSaveURLKeepsCommentHeaderAndReplacesOldURL(t *testing.T) {
	enterTempDir(t)
	writeTestFile(t, xrayURLConfigPath, "# Xray URL Config\n# note\n"+testVLESS+"\n")
	if err := SaveXrayURLConfig(testTrojan); err != nil {
		t.Fatal(err)
	}
	got := readTestFile(t, xrayURLConfigPath)
	if got != "# Xray URL Config\n# note\n"+testTrojan+"\n" {
		t.Fatalf("unexpected file content: %q", got)
	}
	content, isURL, err := readXrayConfig()
	if err != nil || !isURL || content != testTrojan {
		t.Fatalf("unexpected read: %q %v %v", content, isURL, err)
	}
}

func TestSaveRejectsInvalidInputWithoutWriting(t *testing.T) {
	enterTempDir(t)
	if err := SaveXrayURLConfig("http://not-supported"); err == nil {
		t.Fatal("unsupported scheme must be rejected")
	}
	if err := SaveXrayURLConfig("vless://your-uuid@your-server.com:443"); err == nil {
		t.Fatal("placeholder URL must be rejected")
	}
	if err := SaveXrayJSONConfig(`{"inbounds": [`); err == nil {
		t.Fatal("broken JSON must be rejected")
	}
	if err := SaveXrayJSONConfig(defaultRepoJSON); err == nil {
		t.Fatal("JSON without proxy outbound must be rejected")
	}
	for _, p := range []string{xrayURLConfigPath, xrayJSONConfigPath, xrayActiveFormatPath} {
		if _, err := os.Stat(p); err == nil {
			t.Fatalf("%s must not be created by rejected input", p)
		}
	}
}

func TestAllSupportedURLSchemesAccepted(t *testing.T) {
	enterTempDir(t)
	vmessInner := `{"v":"2","ps":"t","add":"real-host.net","port":"443","id":"11111111-2222-3333-4444-555555555555","net":"ws","tls":"tls"}`
	vmess := "vmess://" + base64.StdEncoding.EncodeToString([]byte(vmessInner))
	urls := []string{
		testVLESS,
		vmess,
		testTrojan,
		"ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ@real-host.net:8388#T",
	}
	for _, u := range urls {
		if err := SaveXrayURLConfig(u); err != nil {
			t.Fatalf("%s rejected: %v", strings.SplitN(u, "://", 2)[0], err)
		}
	}
}

func TestWizardFirstTimeURL(t *testing.T) {
	enterTempDir(t)
	out, err := runWizard("1\n" + testVLESS + "\n")
	if err != nil {
		t.Fatalf("wizard failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "No valid Xray configuration was found") {
		t.Fatalf("missing explanation message:\n%s", out)
	}
	if !strings.Contains(out, "URL configuration") || !strings.Contains(out, "JSON configuration") {
		t.Fatalf("missing format menu:\n%s", out)
	}
	if strings.Contains(out, "real-host.net") || strings.Contains(out, "11111111-2222") {
		t.Fatalf("credentials must not be printed:\n%s", out)
	}
	content, isURL, err := readXrayConfig()
	if err != nil || !isURL || content != testVLESS {
		t.Fatalf("saved config not used: %q %v %v", content, isURL, err)
	}
}

func TestWizardFirstTimeMultilineJSON(t *testing.T) {
	enterTempDir(t)
	out, err := runWizard("2\n" + testJSON + "\nEND\n")
	if err != nil {
		t.Fatalf("wizard failed: %v\n%s", err, out)
	}
	saved := readTestFile(t, xrayJSONConfigPath)
	if strings.TrimSpace(saved) != testJSON {
		t.Fatalf("saved JSON differs from input:\n%s", saved)
	}
	if strings.Contains(out, "real-host.net") || strings.Contains(out, "11111111-2222") {
		t.Fatalf("credentials must not be printed:\n%s", out)
	}
	_, isURL, err := readXrayConfig()
	if err != nil || isURL {
		t.Fatalf("JSON must be the used config: isURL=%v err=%v", isURL, err)
	}
}

func TestWizardJSONEndIsCaseInsensitiveAndCRLF(t *testing.T) {
	enterTempDir(t)
	input := "2\r\n" + strings.ReplaceAll(testJSON, "\n", "\r\n") + "\r\n  end  \r\n"
	if _, err := runWizard(input); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(readTestFile(t, xrayJSONConfigPath)), &parsed); err != nil {
		t.Fatalf("saved JSON invalid: %v", err)
	}
}

func TestWizardInvalidJSONNotSavedThenRetry(t *testing.T) {
	enterTempDir(t)
	writeTestFile(t, xrayJSONConfigPath, defaultRepoJSON)
	input := "2\n" + `{"inbounds": [` + "\nEND\n" + testJSON + "\nEND\n"
	out, err := runWizard(input)
	if err != nil {
		t.Fatalf("wizard failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Invalid configuration") || !strings.Contains(out, "Nothing was saved") {
		t.Fatalf("expected rejection message:\n%s", out)
	}
	if strings.TrimSpace(readTestFile(t, xrayJSONConfigPath)) != testJSON {
		t.Fatal("final saved JSON should be the valid one")
	}
}

func TestWizardInvalidJSONNeverOverwritesExisting(t *testing.T) {
	enterTempDir(t)
	if err := SaveXrayJSONConfig(testJSON); err != nil {
		t.Fatal(err)
	}
	before := readTestFile(t, xrayJSONConfigPath)
	_, err := runWizard("2\n2\n{\"a\": 1}\nEND\n")
	if err == nil {
		t.Fatal("expected error when input ends without a valid config")
	}
	if readTestFile(t, xrayJSONConfigPath) != before {
		t.Fatal("existing config must stay untouched when new input is invalid")
	}
}

func TestWizardInvalidURLNotSavedThenRetry(t *testing.T) {
	enterTempDir(t)
	out, err := runWizard("1\nhello\nvless://your-uuid@your-server.com:443\n" + testTrojan + "\n")
	if err != nil {
		t.Fatalf("wizard failed: %v\n%s", err, out)
	}
	content, isURL, err := readXrayConfig()
	if err != nil || !isURL || content != testTrojan {
		t.Fatalf("expected the valid URL to be saved: %q %v %v", content, isURL, err)
	}
}

func TestWizardBackReturnsToFormatMenu(t *testing.T) {
	enterTempDir(t)
	out, err := runWizard("1\nBACK\n2\n" + testJSON + "\nEND\n")
	if err != nil {
		t.Fatalf("wizard failed: %v\n%s", err, out)
	}
	if strings.Count(out, "Select the format of your Xray configuration") != 2 {
		t.Fatalf("format menu should be shown twice:\n%s", out)
	}
	if _, err := os.Stat(xrayURLConfigPath); err == nil {
		t.Fatal("URL file must not be created after BACK")
	}
}

func TestWizardBackFromJSON(t *testing.T) {
	enterTempDir(t)
	if _, err := runWizard("2\nback\n1\n" + testVLESS + "\n"); err != nil {
		t.Fatal(err)
	}
	_, isURL, _ := readXrayConfig()
	if !isURL {
		t.Fatal("URL should be active after switching")
	}
}

func TestWizardUsesExistingByDefault(t *testing.T) {
	enterTempDir(t)
	writeTestFile(t, xrayURLConfigPath, testVLESS+"\n")
	before := readTestFile(t, xrayURLConfigPath)
	out, err := runWizard("\n")
	if err != nil {
		t.Fatalf("wizard failed: %v", err)
	}
	if !strings.Contains(out, "previously saved Xray configuration was found") {
		t.Fatalf("missing found message:\n%s", out)
	}
	if !strings.Contains(out, "Do you want to edit your existing Xray configuration?") {
		t.Fatalf("missing question:\n%s", out)
	}
	if strings.Contains(out, "Select the format") {
		t.Fatalf("must not ask for a new config:\n%s", out)
	}
	if readTestFile(t, xrayURLConfigPath) != before {
		t.Fatal("existing config must not change")
	}
}

func TestWizardExplicitUseExisting(t *testing.T) {
	enterTempDir(t)
	writeTestFile(t, xrayJSONConfigPath, testJSON)
	out, err := runWizard("1\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "JSON format") {
		t.Fatalf("expected JSON format to be reported:\n%s", out)
	}
}

func TestWizardInvalidChoiceThenDefault(t *testing.T) {
	enterTempDir(t)
	writeTestFile(t, xrayURLConfigPath, testVLESS+"\n")
	out, err := runWizard("9\n\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Invalid choice") {
		t.Fatalf("expected invalid choice message:\n%s", out)
	}
}

func TestWizardEditReplacesConfigAndSwitchesFormat(t *testing.T) {
	enterTempDir(t)
	writeTestFile(t, xrayURLConfigPath, testVLESS+"\n")
	out, err := runWizard("2\n2\n" + testJSON + "\nEND\n")
	if err != nil {
		t.Fatalf("wizard failed: %v\n%s", err, out)
	}
	_, isURL, err := readXrayConfig()
	if err != nil || isURL {
		t.Fatalf("JSON must now be used: isURL=%v err=%v", isURL, err)
	}
	if !strings.Contains(readTestFile(t, xrayURLConfigPath), testVLESS) {
		t.Fatal("old URL config must be preserved")
	}
	out2, err := runWizard("\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2, "JSON format") {
		t.Fatalf("second run should recognize saved JSON config:\n%s", out2)
	}
}

func TestWizardEOFReturnsError(t *testing.T) {
	enterTempDir(t)
	if _, err := runWizard(""); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF error, got %v", err)
	}
	if _, err := runWizard("1\n"); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF error, got %v", err)
	}
	if _, err := runWizard("2\n"); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF error, got %v", err)
	}
}

func TestWizardJSONWithoutENDAtEOFStillValidated(t *testing.T) {
	enterTempDir(t)
	if _, err := runWizard("2\n" + testJSON + "\n"); err != nil {
		t.Fatalf("complete JSON at EOF should be accepted: %v", err)
	}
	enterTempDir(t)
	if _, err := runWizard("2\n{\"inbounds\": [\n"); err == nil {
		t.Fatal("incomplete JSON at EOF must be rejected")
	}
	if _, err := os.Stat(xrayJSONConfigPath); err == nil {
		t.Fatal("incomplete JSON must not be saved")
	}
}

func TestAtomicWriteFailureKeepsOriginalAndCleansTemp(t *testing.T) {
	enterTempDir(t)
	original := testVLESS + "\n"
	writeTestFile(t, xrayURLConfigPath, original)
	saved := renameFile
	renameFile = func(string, string) error { return errors.New("simulated rename failure") }
	defer func() { renameFile = saved }()
	if err := SaveXrayURLConfig(testTrojan); err == nil {
		t.Fatal("expected save to fail")
	}
	renameFile = saved
	if readTestFile(t, xrayURLConfigPath) != original {
		t.Fatal("original config must remain intact")
	}
	if left := leftoverTempFiles(t, "config"); len(left) != 0 {
		t.Fatalf("temporary files left behind: %v", left)
	}
	if _, err := os.Stat(xrayActiveFormatPath); err == nil {
		t.Fatal("selection marker must not be written when the save failed")
	}
}

func TestWizardReportsSaveFailure(t *testing.T) {
	enterTempDir(t)
	saved := renameFile
	renameFile = func(string, string) error { return errors.New("simulated rename failure") }
	defer func() { renameFile = saved }()
	_, err := runWizard("1\n" + testVLESS + "\n")
	if err == nil {
		t.Fatal("expected wizard to report the failure")
	}
	if _, statErr := os.Stat(filepath.Join("config", "xray_config.txt")); statErr == nil {
		t.Fatal("no config file should exist after failed save")
	}
}

func TestMarkerIgnoredWhenGarbage(t *testing.T) {
	enterTempDir(t)
	writeTestFile(t, xrayActiveFormatPath, "banana\n")
	writeTestFile(t, xrayURLConfigPath, testVLESS+"\n")
	writeTestFile(t, xrayJSONConfigPath, testJSON)
	_, isURL, err := readXrayConfig()
	if err != nil || !isURL {
		t.Fatalf("garbage marker must fall back to legacy behaviour: %v %v", isURL, err)
	}
}

func TestInvalidSelectedJSONBlocksScanInsteadOfSilentlyUsingURL(t *testing.T) {
	enterTempDir(t)
	writeTestFile(t, xrayURLConfigPath, testVLESS+"\n")
	writeTestFile(t, xrayJSONConfigPath, "{broken")
	writeTestFile(t, xrayActiveFormatPath, "json\n")
	if err := ValidateXrayConfig(); err == nil {
		t.Fatal("selected but invalid JSON must be reported")
	}
}