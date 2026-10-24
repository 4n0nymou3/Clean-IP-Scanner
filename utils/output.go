package utils

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/4n0nymou3/Clean-IP-Scanner/scanner"
	"github.com/fatih/color"
)

const tableWidth = 66

const portsPerLine = 7

var gradientStart = [3]int{57, 255, 20}
var gradientEnd = [3]int{255, 230, 0}

func rankColor(rank, total int) *color.Color {
	if total <= 1 {
		return color.RGB(gradientStart[0], gradientStart[1], gradientStart[2]).Add(color.Bold)
	}
	t := float64(rank) / float64(total-1)
	r := gradientStart[0] + int(float64(gradientEnd[0]-gradientStart[0])*t+0.5)
	g := gradientStart[1] + int(float64(gradientEnd[1]-gradientStart[1])*t+0.5)
	b := gradientStart[2] + int(float64(gradientEnd[2]-gradientStart[2])*t+0.5)
	return color.RGB(r, g, b).Add(color.Bold)
}

func endpoint(ip *net.IPAddr, port int, showPort bool) string {
	if !showPort {
		return ip.String()
	}
	return net.JoinHostPort(ip.String(), strconv.Itoa(port))
}

func PrintScanPorts(ports []int) {
	cyan := color.New(color.FgCyan)
	cyan.Printf("Scan ports (%d):\n", len(ports))
	for i := 0; i < len(ports); i += portsPerLine {
		end := i + portsPerLine
		if end > len(ports) {
			end = len(ports)
		}
		parts := make([]string, 0, end-i)
		for _, p := range ports[i:end] {
			parts = append(parts, strconv.Itoa(p))
		}
		line := "  " + strings.Join(parts, ", ")
		if end < len(ports) {
			line += ","
		}
		cyan.Println(line)
	}
	fmt.Println()
	color.New(color.FgYellow).Printf("Note: every IP is tested on all %d ports (about %dx longer).\n", len(ports), len(ports))
	fmt.Println()
}

func PrintResults(results []scanner.IPResult, showPort bool) {
	width := tableWidth
	ipWidth := 20
	if showPort {
		ipWidth = 16
		for _, r := range results {
			if l := len(r.IP.String()) + 1; l > ipWidth {
				ipWidth = l
			}
		}
		width = 50 + ipWidth
	}

	fmt.Println()
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Println(strings.Repeat("=", width))
	cyan.Println(CenterText(width, "CLEAN IPs FOUND"))
	cyan.Println(strings.Repeat("=", width))
	fmt.Println()

	green := color.New(color.FgGreen, color.Bold)

	if showPort {
		layout := fmt.Sprintf("%%-4s%%-%ds%%-6s%%-5s%%-5s%%-8s%%-11s%%-11s\n", ipWidth)
		green.Printf(layout, "#", "IP", "Port", "S/R", "Loss", "Delay", "DL", "UL")
		cyan.Println(strings.Repeat("-", width))

		total := len(results)
		for i, r := range results {
			rank := fmt.Sprintf("%d.", i+1)
			sr := fmt.Sprintf("%d/%d", r.Sended, r.Received)
			loss := fmt.Sprintf("%.2f", r.LossRate)
			delay := fmt.Sprintf("%dms", r.Delay)
			downloadSpeed := fmt.Sprintf("%.2fMB/s", r.DownloadSpeed/1024/1024)
			uploadSpeed := fmt.Sprintf("%.2fMB/s", r.UploadSpeed/1024/1024)

			rankColor(i, total).Printf(layout,
				rank, r.IP.String(), strconv.Itoa(r.Port), sr, loss, delay, downloadSpeed, uploadSpeed)
		}

		cyan.Println(strings.Repeat("=", width))
		return
	}

	green.Printf("%-4s%-20s%-6s%-6s%-8s%-11s%-11s\n",
		"#", "IP", "S/R", "Loss", "Delay", "DL", "UL")
	cyan.Println(strings.Repeat("-", width))

	total := len(results)
	for i, r := range results {
		rank := fmt.Sprintf("%d.", i+1)
		sr := fmt.Sprintf("%d/%d", r.Sended, r.Received)
		loss := fmt.Sprintf("%.2f", r.LossRate)
		delay := fmt.Sprintf("%dms", r.Delay)
		downloadSpeed := fmt.Sprintf("%.2fMB/s", r.DownloadSpeed/1024/1024)
		uploadSpeed := fmt.Sprintf("%.2fMB/s", r.UploadSpeed/1024/1024)

		rankColor(i, total).Printf("%-4s%-20s%-6s%-6s%-8s%-11s%-11s\n",
			rank, r.IP.String(), sr, loss, delay, downloadSpeed, uploadSpeed)
	}

	cyan.Println(strings.Repeat("=", width))
}

func SaveResults(results []scanner.IPResult, filename string, showPort bool) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	file.WriteString("# Clean IPs\n")
	file.WriteString(fmt.Sprintf("# Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	if showPort {
		file.WriteString(fmt.Sprintf("# Total IP:Port pairs found: %d\n", len(results)))
	} else {
		file.WriteString(fmt.Sprintf("# Total IPs found: %d\n", len(results)))
	}
	file.WriteString("#\n")
	if showPort {
		file.WriteString("# Format: Rank | IP:Port | Sent | Received | Loss | Avg Delay | Download Speed | Upload Speed\n")
	} else {
		file.WriteString("# Format: Rank | IP | Sent | Received | Loss | Avg Delay | Download Speed | Upload Speed\n")
	}
	file.WriteString("#===========================================================================\n\n")

	for i, r := range results {
		line := fmt.Sprintf("%d. %s | Sent: %d | Recv: %d | Loss: %.2f | %dms | %.2f MB/s | %.2f MB/s\n",
			i+1,
			endpoint(r.IP, r.Port, showPort),
			r.Sended,
			r.Received,
			r.LossRate,
			r.Delay,
			r.DownloadSpeed/1024/1024,
			r.UploadSpeed/1024/1024,
		)
		file.WriteString(line)
	}

	file.WriteString("\n# End of results\n")
	return nil
}

func SaveSimpleResults(topResults []scanner.IPResult, pingResults []scanner.PingResult, filename string, showPort bool) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, r := range topResults {
		file.WriteString(endpoint(r.IP, r.Port, showPort) + "\n")
	}

	file.WriteString("--------------------\n")

	for _, r := range pingResults {
		file.WriteString(endpoint(r.IP, r.Port, showPort) + "\n")
	}

	return nil
}