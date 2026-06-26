package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/4n0nymou3/Clean-IP-Scanner/config"
	"github.com/4n0nymou3/Clean-IP-Scanner/scanner"
	"github.com/4n0nymou3/Clean-IP-Scanner/utils"
	"github.com/fatih/color"
)

const version = "3.4.0"

func clearScreen() {
	fmt.Print("\033[H\033[2J\033[3J")
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func printScanStats(elapsed time.Duration, interrupted bool) {
	fmt.Println()
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println("========================================")
	if interrupted {
		color.New(color.FgYellow, color.Bold).Println("         Scan stopped by user")
	} else {
		cyan.Println("      Scan completed successfully!")
	}
	cyan.Println("========================================")
	fmt.Println()
	color.New(color.FgCyan).Printf("  Scan Duration : %s\n", formatDuration(elapsed))
	fmt.Println()
}

func askScanMode() int {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println()
		color.New(color.FgCyan, color.Bold).Println("Select scan mode:")
		color.New(color.FgWhite).Println("  [1] Normal scan (TCP ping + speed test)")
		color.New(color.FgWhite).Println("  [2] Xray scan (uses Xray core with your config)")
		fmt.Print("Enter 1 or 2: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "1" {
			return 1
		} else if input == "2" {
			if _, err := os.Stat("./xray/xray"); os.IsNotExist(err) {
				fmt.Println()
				color.New(color.FgRed).Println("Error: Xray binary not found. Please reinstall the tool.")
				os.Exit(1)
			}
			if err := scanner.ValidateXrayConfig(); err != nil {
				fmt.Println()
				color.New(color.FgRed).Println("Error: " + err.Error())
				fmt.Println()
				color.New(color.FgYellow).Println("How to fix:")
				color.New(color.FgWhite).Println("  For URL config : edit config/xray_config.txt  (paste vless:// or vmess:// or trojan:// or ss:// link)")
				color.New(color.FgWhite).Println("  For JSON config: edit config/xray_config.json (paste full Xray JSON config)")
				os.Exit(1)
			}
			fmt.Println()
			color.New(color.FgCyan).Println("Running Xray environment self-test...")
			if err := scanner.SelfTestXray(); err != nil {
				fmt.Println()
				color.New(color.FgRed, color.Bold).Println("XRAY SELF-TEST FAILED!")
				color.New(color.FgYellow).Println("The Xray core could not start correctly on your device.")
				fmt.Println()
				color.New(color.FgWhite).Println(err.Error())
				os.Exit(1)
			}
			color.New(color.FgGreen).Println("Xray self-test passed successfully!")
			return 2
		}
		color.New(color.FgRed).Println("Invalid choice. Please enter 1 or 2.")
	}
}

func askWorkerCount() int {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println()
		color.New(color.FgCyan, color.Bold).Println("Select number of parallel workers for Xray scan:")
		color.New(color.FgWhite).Println("  4  - Recommended for weak/older devices")
		color.New(color.FgWhite).Println("  8  - Recommended for most devices (default)")
		color.New(color.FgWhite).Println("  16 - Recommended for powerful devices")
		color.New(color.FgWhite).Println("  Valid range: 1 to 16")
		fmt.Print("Enter worker count (press Enter for default 8): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			return 8
		}
		n, err := strconv.Atoi(input)
		if err != nil || n < 1 || n > 16 {
			color.New(color.FgRed).Println("Invalid input. Please enter a number between 1 and 16.")
			continue
		}
		return n
	}
}

func askResumeOrNew() bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter R to resume previous scan or N to start a new scan: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToUpper(input))
		if input == "R" {
			return true
		} else if input == "N" {
			return false
		}
		color.New(color.FgRed).Println("Invalid input. Please enter R or N.")
	}
}

func main() {
	clearScreen()

	utils.PrintHeader()
	utils.PrintDesigner()

	cyan := color.New(color.FgCyan)
	cyan.Printf("Version: %s\n", version)
	fmt.Println()

	color.New(color.FgYellow).Println("Optimized for Iran network conditions")
	color.New(color.FgYellow).Println("Press Ctrl+C at any time to stop and see results found so far.")
	fmt.Println()

	var (
		mode          int
		workers       int
		cp            *scanner.Checkpoint
		resuming      bool
		existingPRs   []scanner.PingResult
		skipPingPhase bool
	)

	existingCP := scanner.LoadCheckpoint()
	if existingCP != nil {
		fmt.Println()
		color.New(color.FgYellow, color.Bold).Println("+-------------------------------------------------+")
		color.New(color.FgYellow, color.Bold).Println("|         UNFINISHED SCAN DETECTED                |")
		color.New(color.FgYellow, color.Bold).Println("+-------------------------------------------------+")
		fmt.Println()

		modeStr := "Normal (TCP)"
		if existingCP.Mode == 2 {
			modeStr = fmt.Sprintf("Xray (%d workers)", existingCP.Workers)
		}
		scanned := existingCP.ProgressIndex
		total := existingCP.TotalIPs
		found := len(existingCP.PingResults)
		pct := 0
		if total > 0 {
			pct = scanned * 100 / total
		}

		color.New(color.FgCyan).Printf("  Previous scan mode : %s\n", modeStr)
		color.New(color.FgCyan).Printf("  Progress           : %d / %d IPs (%d%%)\n", scanned, total, pct)
		color.New(color.FgCyan).Printf("  Responsive IPs     : %d found so far\n", found)
		color.New(color.FgCyan).Printf("  Saved at           : %s\n", existingCP.SavedAt)
		fmt.Println()

		resuming = askResumeOrNew()

		if resuming {
			cp = existingCP
			mode = cp.Mode
			workers = cp.Workers
			existingPRs = cp.GetPingResults()
			skipPingPhase = cp.Phase == scanner.PhaseSpeed
			fmt.Println()
			color.New(color.FgGreen).Println("Resuming previous scan...")
		} else {
			scanner.DeleteCheckpoint()
			fmt.Println()
		}
	}

	if !resuming {
		mode = askScanMode()
		workers = 8
		if mode == 2 {
			workers = askWorkerCount()
		}
	}

	time.Sleep(500 * time.Millisecond)

	stopPingCh := make(chan struct{})
	stopSpeedCh := make(chan struct{})
	inSpeedPhase := int32(0)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			<-sigChan
			fmt.Println()
			if atomic.LoadInt32(&inSpeedPhase) == 0 {
				color.New(color.FgYellow, color.Bold).Println("Interrupt received. Stopping ping phase and proceeding to speed test with IPs found so far...")
				select {
				case <-stopPingCh:
				default:
					close(stopPingCh)
				}
			} else {
				color.New(color.FgYellow, color.Bold).Println("Interrupt received. Stopping speed test and collecting results...")
				signal.Reset(os.Interrupt)
				select {
				case <-stopSpeedCh:
				default:
					close(stopSpeedCh)
				}
				return
			}
		}
	}()

	startTime := time.Now()

	pingResults := make([]scanner.PingResult, 0)
	pingWasStopped := false

	if skipPingPhase {
		color.New(color.FgCyan).Printf("Resuming at speed test phase with %d responsive IPs...\n\n", len(existingPRs))
		pingResults = existingPRs
	} else {
		ipRanges := config.GetCloudflareRanges()
		var scanIPs []scanner.CompactIP

		if resuming && cp != nil {
			color.New(color.FgCyan).Println("Rebuilding IP list from saved seed...")
			allIPs := scanner.GenerateIPsWithSeed(ipRanges, cp.Seed)
			start := cp.ProgressIndex
			if start >= len(allIPs) {
				start = len(allIPs)
			}
			scanIPs = allIPs[start:]
			color.New(color.FgCyan).Printf("Resuming ping phase: %d of %d IPs remaining\n\n", len(scanIPs), cp.TotalIPs)
		} else {
			allIPs, seed := scanner.GenerateIPs(ipRanges)
			scanIPs = allIPs
			cp = scanner.NewCheckpoint(mode, workers, len(allIPs), seed)
			cp.Save()
			fmt.Println()
		}

		if len(scanIPs) == 0 {
			color.New(color.FgRed, color.Bold).Println("No IPs to scan!")
			fmt.Println()
			color.New(color.FgYellow).Println("Check config/ip_ranges.txt — all entries may have been skipped.")
			color.New(color.FgYellow).Println("For IPv6, use subnets of /108 or smaller (e.g., 2606:4700::/108).")
			os.Exit(1)
		}

		var newPingResults []scanner.PingResult
		if mode == 1 {
			newPingResults = scanner.PingIPs(stopPingCh, scanIPs, cp, existingPRs)
		} else {
			newPingResults = scanner.PingIPsViaXray(stopPingCh, scanIPs, workers, cp, existingPRs)
		}

		pingResults = append(existingPRs, newPingResults...)

		select {
		case <-stopPingCh:
			pingWasStopped = true
		default:
		}

		if pingWasStopped && len(pingResults) == 0 {
			elapsed := time.Since(startTime)
			color.New(color.FgYellow).Println("Scan stopped during latency test. No responsive IPs found yet.")
			if cp != nil {
				cp.SetPingResults(pingResults)
				cp.Save()
			}
			printScanStats(elapsed, true)
			return
		}

		if !pingWasStopped && len(pingResults) == 0 {
			color.New(color.FgRed, color.Bold).Println("No responsive IPs found!")
			fmt.Println()
			color.New(color.FgYellow).Println("Try running again. Network conditions may vary.")
			elapsed := time.Since(startTime)
			scanner.DeleteCheckpoint()
			printScanStats(elapsed, false)
			return
		}

		if cp != nil {
			if pingWasStopped {
				cp.SetPingResults(pingResults)
				cp.Save()
			} else {
				cp.MarkPingDone(pingResults)
			}
		}
	}

	fmt.Println()
	atomic.StoreInt32(&inSpeedPhase, 1)

	var results []scanner.IPResult
	if mode == 1 {
		results = scanner.SpeedTest(stopSpeedCh, pingResults)
	} else {
		results = scanner.SpeedTestViaXray(stopSpeedCh, pingResults)
	}

	elapsed := time.Since(startTime)

	interrupted := false
	select {
	case <-stopSpeedCh:
		interrupted = true
	default:
	}

	if !interrupted && cp != nil {
		cp.MarkCompleted()
	}

	if len(results) == 0 {
		red := color.New(color.FgRed, color.Bold)
		if interrupted {
			red.Println("No clean IPs found before scan was stopped.")
		} else {
			red.Println("No clean IPs found.")
			fmt.Println()
			color.New(color.FgYellow).Println("Try running again at a different time.")
		}
		printScanStats(elapsed, interrupted)
		return
	}

	topResults := results
	if len(results) > 10 {
		topResults = results[:10]
	}

	if interrupted {
		color.New(color.FgYellow, color.Bold).Printf(
			"\nShowing %d clean IP(s) found before scan was stopped:\n", len(results))
	}

	utils.PrintResults(topResults)

	if err := utils.SaveResults(results, "clean_ips.txt"); err != nil {
		color.New(color.FgRed).Printf("Error saving file: %v\n", err)
	} else {
		color.New(color.FgGreen).Println("Results saved to clean_ips.txt")
		color.New(color.FgGreen).Printf("Total clean IPs found: %d\n", len(results))
	}

	if err := utils.SaveSimpleResults(topResults, pingResults, "clean_ips_list.txt"); err != nil {
		color.New(color.FgRed).Printf("Error saving simple list: %v\n", err)
	} else {
		color.New(color.FgGreen).Println("Simple IP list saved to clean_ips_list.txt")
	}

	printScanStats(elapsed, interrupted)
}