package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	xrayActiveFormatPath = "./config/xray_config.active"
	XrayFormatURL        = "url"
	XrayFormatJSON       = "json"
)

var renameFile = os.Rename

func activeXrayFormat() string {
	data, err := os.ReadFile(xrayActiveFormatPath)
	if err != nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(string(data))) {
	case XrayFormatURL:
		return XrayFormatURL
	case XrayFormatJSON:
		return XrayFormatJSON
	}
	return ""
}

func firstURLConfigLine(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}

func SavedXrayFormat() string {
	if activeXrayFormat() == XrayFormatJSON {
		return XrayFormatJSON
	}
	if data, err := os.ReadFile(xrayURLConfigPath); err == nil {
		if firstURLConfigLine(string(data)) != "" {
			return XrayFormatURL
		}
	}
	return XrayFormatJSON
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		tmp.Close()
		os.Remove(tmpPath)
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	os.Chmod(tmpPath, perm)
	if err := renameFile(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

func existingFileMode(path string, fallback os.FileMode) os.FileMode {
	if fi, err := os.Stat(path); err == nil {
		return fi.Mode().Perm()
	}
	return fallback
}

func setActiveXrayFormat(format string) error {
	if err := writeFileAtomic(xrayActiveFormatPath, []byte(format+"\n"), 0644); err != nil {
		return fmt.Errorf("configuration was saved but the selected format could not be recorded: %v", err)
	}
	return nil
}

func SaveXrayURLConfig(raw string) error {
	line := strings.TrimSpace(raw)
	if line == "" {
		return fmt.Errorf("no URL was entered")
	}
	if strings.ContainsAny(line, "\r\n") {
		return fmt.Errorf("the URL must be on a single line")
	}
	if err := validateURLConfig(line); err != nil {
		return err
	}
	var kept []string
	if data, err := os.ReadFile(xrayURLConfigPath); err == nil {
		for _, l := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
			t := strings.TrimSpace(l)
			if strings.HasPrefix(t, "#") {
				kept = append(kept, l)
			}
		}
	}
	var sb strings.Builder
	for _, l := range kept {
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	sb.WriteString(line)
	sb.WriteString("\n")
	if err := writeFileAtomic(xrayURLConfigPath, []byte(sb.String()), existingFileMode(xrayURLConfigPath, 0644)); err != nil {
		return fmt.Errorf("could not save %s: %v", "config/xray_config.txt", err)
	}
	return setActiveXrayFormat(XrayFormatURL)
}

func SaveXrayJSONConfig(raw string) error {
	content := strings.TrimSpace(raw)
	if content == "" {
		return fmt.Errorf("no JSON was entered")
	}
	if err := validateJSONConfig(content); err != nil {
		return err
	}
	if err := writeFileAtomic(xrayJSONConfigPath, []byte(content+"\n"), existingFileMode(xrayJSONConfigPath, 0644)); err != nil {
		return fmt.Errorf("could not save %s: %v", "config/xray_config.json", err)
	}
	return setActiveXrayFormat(XrayFormatJSON)
}