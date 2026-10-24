package scanner

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

const MaxPorts = 13

var scanPorts []int

func normalizePorts(ps []int) []int {
	seen := make(map[int]bool)
	out := make([]int, 0, len(ps))
	for _, p := range ps {
		if p < 1 || p > 65535 || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
		if len(out) == MaxPorts {
			break
		}
	}
	return out
}

func SetPorts(ps []int) {
	scanPorts = normalizePorts(ps)
}

func MultiPort() bool {
	return len(scanPorts) > 1
}

func tcpPorts() []int {
	if len(scanPorts) == 0 {
		return []int{defaultPort}
	}
	return scanPorts
}

func xrayPorts() []int {
	if len(scanPorts) == 0 {
		return []int{0}
	}
	return scanPorts
}

func speedPortFor(p int) int {
	if p <= 0 || p > 65535 {
		return defaultPort
	}
	return p
}

func normalizeDigits(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '۰' && r <= '۹':
			return '0' + (r - '۰')
		case r >= '٠' && r <= '٩':
			return '0' + (r - '٠')
		}
		return r
	}, s)
}

func isPortSeparator(r rune) bool {
	return r == ',' || r == '،' || r == ';' || r == '؛' || unicode.IsSpace(r)
}

func ParsePorts(input string) ([]int, bool, error) {
	tokens := strings.FieldsFunc(normalizeDigits(input), isPortSeparator)
	if len(tokens) == 0 {
		return nil, false, fmt.Errorf("no port entered")
	}
	seen := make(map[int]bool)
	ports := make([]int, 0, len(tokens))
	duplicates := false
	for _, tok := range tokens {
		n, err := strconv.Atoi(tok)
		if err != nil || n < 1 || n > 65535 {
			return nil, false, fmt.Errorf("\"%s\" is not a valid port (use numbers between 1 and 65535)", tok)
		}
		if seen[n] {
			duplicates = true
			continue
		}
		seen[n] = true
		ports = append(ports, n)
	}
	if len(ports) > MaxPorts {
		return nil, false, fmt.Errorf("too many ports (%d) - the maximum is %d", len(ports), MaxPorts)
	}
	return ports, duplicates, nil
}

func speedPortUsesTLS(p int) bool {
	return !xrayNonTLSPorts[p]
}

func selectSpeedCandidates(pingResults []PingResult, limit int) []PingResult {
	if !MultiPort() {
		return pingResults
	}
	picked := make([]bool, len(pingResults))
	chosen := make([]PingResult, 0, limit)
	seenIP := make(map[string]bool)
	for _, wantTLS := range []bool{true, false} {
		for i, r := range pingResults {
			if len(chosen) >= limit {
				return chosen
			}
			key := r.IP.String()
			if picked[i] || seenIP[key] || speedPortUsesTLS(r.Port) != wantTLS {
				continue
			}
			seenIP[key] = true
			picked[i] = true
			chosen = append(chosen, r)
		}
	}
	for _, wantTLS := range []bool{true, false} {
		for i, r := range pingResults {
			if len(chosen) >= limit {
				return chosen
			}
			if picked[i] || speedPortUsesTLS(r.Port) != wantTLS {
				continue
			}
			picked[i] = true
			chosen = append(chosen, r)
		}
	}
	return chosen
}