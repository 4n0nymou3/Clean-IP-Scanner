package scanner

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

var errWizardBack = errors.New("back to format menu")

func wizardReadLine(in *bufio.Reader) (string, error) {
	s, err := in.ReadString('\n')
	if err != nil && s == "" {
		return "", err
	}
	return strings.TrimRight(s, "\r\n"), nil
}

func wizardPrint(out io.Writer, c *color.Color, format string, args ...interface{}) {
	c.Fprintf(out, format, args...)
}

func wizardAskExistingChoice(in *bufio.Reader, out io.Writer) (int, error) {
	for {
		fmt.Fprintln(out)
		wizardPrint(out, color.New(color.FgCyan, color.Bold), "Do you want to edit your existing Xray configuration?\n")
		wizardPrint(out, color.New(color.FgWhite), "  [1] Use existing configuration (default)\n")
		wizardPrint(out, color.New(color.FgWhite), "  [2] Edit configuration\n")
		fmt.Fprint(out, "Enter 1 or 2 (press Enter for default 1): ")
		line, err := wizardReadLine(in)
		if err != nil {
			return 0, err
		}
		switch strings.TrimSpace(line) {
		case "", "1":
			return 1, nil
		case "2":
			return 2, nil
		}
		wizardPrint(out, color.New(color.FgRed), "Invalid choice. Please enter 1 or 2.\n")
	}
}

func wizardAskFormat(in *bufio.Reader, out io.Writer) (string, error) {
	for {
		fmt.Fprintln(out)
		wizardPrint(out, color.New(color.FgCyan, color.Bold), "Select the format of your Xray configuration:\n")
		wizardPrint(out, color.New(color.FgWhite), "  [1] URL configuration\n")
		wizardPrint(out, color.New(color.FgWhite), "  [2] JSON configuration\n")
		fmt.Fprint(out, "Enter 1 or 2: ")
		line, err := wizardReadLine(in)
		if err != nil {
			return "", err
		}
		switch strings.TrimSpace(line) {
		case "1":
			return XrayFormatURL, nil
		case "2":
			return XrayFormatJSON, nil
		}
		wizardPrint(out, color.New(color.FgRed), "Invalid choice. Please enter 1 or 2.\n")
	}
}

func wizardEnterURL(in *bufio.Reader, out io.Writer) error {
	fmt.Fprintln(out)
	wizardPrint(out, color.New(color.FgCyan, color.Bold), "Paste your Xray URL configuration and press Enter.\n")
	wizardPrint(out, color.New(color.FgWhite), "  Supported: vless://, vmess://, trojan://, ss://\n")
	wizardPrint(out, color.New(color.FgWhite), "  Type BACK to return to the format menu.\n")
	for {
		fmt.Fprint(out, "URL: ")
		line, err := wizardReadLine(in)
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.EqualFold(line, "BACK") {
			return errWizardBack
		}
		if err := validateURLConfig(line); err != nil {
			wizardPrint(out, color.New(color.FgRed), "Invalid configuration: %v\n", err)
			wizardPrint(out, color.New(color.FgYellow), "Nothing was saved. Please paste a valid URL or type BACK.\n")
			continue
		}
		if err := SaveXrayURLConfig(line); err != nil {
			return err
		}
		wizardPrint(out, color.New(color.FgGreen), "URL configuration validated and saved to config/xray_config.txt\n")
		return nil
	}
}

func wizardEnterJSON(in *bufio.Reader, out io.Writer) error {
	fmt.Fprintln(out)
	wizardPrint(out, color.New(color.FgCyan, color.Bold), "Paste your complete Xray JSON configuration (multiple lines are supported).\n")
	wizardPrint(out, color.New(color.FgWhite), "  When you are done, type END on a separate line and press Enter.\n")
	wizardPrint(out, color.New(color.FgWhite), "  Type BACK on the first line to return to the format menu.\n")
	fmt.Fprintln(out)
	for {
		var lines []string
		reachedEOF := false
		for {
			line, err := wizardReadLine(in)
			if err != nil {
				if len(lines) == 0 {
					return err
				}
				reachedEOF = true
				break
			}
			trimmed := strings.TrimSpace(line)
			if len(lines) == 0 && strings.EqualFold(trimmed, "BACK") {
				return errWizardBack
			}
			if strings.EqualFold(trimmed, "END") {
				break
			}
			lines = append(lines, line)
		}
		content := strings.TrimSpace(strings.Join(lines, "\n"))
		if content == "" {
			wizardPrint(out, color.New(color.FgRed), "No configuration was entered. Nothing was saved.\n")
		} else if err := validateJSONConfig(content); err != nil {
			wizardPrint(out, color.New(color.FgRed), "Invalid configuration: %v\n", err)
			wizardPrint(out, color.New(color.FgYellow), "Nothing was saved.\n")
		} else {
			if err := SaveXrayJSONConfig(content); err != nil {
				return err
			}
			wizardPrint(out, color.New(color.FgGreen), "JSON configuration validated and saved to config/xray_config.json\n")
			return nil
		}
		if reachedEOF {
			return io.ErrUnexpectedEOF
		}
		wizardPrint(out, color.New(color.FgCyan), "Paste the complete JSON configuration again, then type END on a separate line (or BACK to return).\n")
		fmt.Fprintln(out)
	}
}

func wizardEnterConfig(in *bufio.Reader, out io.Writer) error {
	for {
		format, err := wizardAskFormat(in, out)
		if err != nil {
			return err
		}
		if format == XrayFormatURL {
			err = wizardEnterURL(in, out)
		} else {
			err = wizardEnterJSON(in, out)
		}
		if errors.Is(err, errWizardBack) {
			continue
		}
		if err != nil {
			return err
		}
		if err := ValidateXrayConfig(); err != nil {
			return fmt.Errorf("the saved configuration could not be verified: %v", err)
		}
		_, isURL, err := readXrayConfig()
		if err != nil {
			return err
		}
		if isURL != (format == XrayFormatURL) {
			return fmt.Errorf("the saved configuration is not the one selected for scanning")
		}
		return nil
	}
}

func EnsureXrayConfig(in *bufio.Reader, out io.Writer) error {
	if ValidateXrayConfig() == nil {
		fmt.Fprintln(out)
		format := "URL"
		if SavedXrayFormat() == XrayFormatJSON {
			format = "JSON"
		}
		wizardPrint(out, color.New(color.FgGreen), "A previously saved Xray configuration was found (%s format).\n", format)
		choice, err := wizardAskExistingChoice(in, out)
		if err != nil {
			return err
		}
		if choice == 1 {
			return nil
		}
	} else {
		fmt.Fprintln(out)
		wizardPrint(out, color.New(color.FgYellow), "No valid Xray configuration was found.\n")
		wizardPrint(out, color.New(color.FgWhite), "An Xray configuration must be provided first.\n")
		wizardPrint(out, color.New(color.FgWhite), "It will be saved automatically and reused in future scans.\n")
	}
	return wizardEnterConfig(in, out)
}