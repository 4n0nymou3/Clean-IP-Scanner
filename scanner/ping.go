package scanner

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/fatih/color"
)

const (
	tcpConnectTimeout = 1 * time.Second
	defaultPort       = 443
	maxRoutines       = 200
	defaultPingTimes  = 4
)

type PingResult struct {
	IP       *net.IPAddr
	Port     int
	Sended   int
	Received int
	Delay    time.Duration
}

func (p *PingResult) GetLossRate() float32 {
	lost := p.Sended - p.Received
	return float32(lost) / float32(p.Sended)
}

func tcping(ip *net.IPAddr, port int) (bool, time.Duration) {
	start := time.Now()
	var addr string
	if isIPv4(ip.String()) {
		addr = fmt.Sprintf("%s:%d", ip.String(), port)
	} else {
		addr = fmt.Sprintf("[%s]:%d", ip.String(), port)
	}
	conn, err := net.DialTimeout("tcp", addr, tcpConnectTimeout)
	if err != nil {
		return false, 0
	}
	conn.Close()
	return true, time.Since(start)
}

func checkConnection(ip *net.IPAddr, port int) (recv int, totalDelay time.Duration) {
	for i := 0; i < defaultPingTimes; i++ {
		if ok, d := tcping(ip, port); ok {
			recv++
			totalDelay += d
		}
	}
	return
}

func probePort(ip *net.IPAddr, port int) PingResult {
	recv, totalDelay := checkConnection(ip, port)
	r := PingResult{
		IP:       ip,
		Port:     port,
		Sended:   defaultPingTimes,
		Received: recv,
	}
	if recv > 0 {
		r.Delay = totalDelay / time.Duration(recv)
	}
	return r
}

func probePorts(ip *net.IPAddr, ports []int) []PingResult {
	outcomes := make([]PingResult, len(ports))
	if len(ports) == 1 {
		outcomes[0] = probePort(ip, ports[0])
	} else {
		var pwg sync.WaitGroup
		for i, p := range ports {
			pwg.Add(1)
			go func(idx, prt int) {
				defer pwg.Done()
				outcomes[idx] = probePort(ip, prt)
			}(i, p)
		}
		pwg.Wait()
	}
	found := make([]PingResult, 0, len(ports))
	for _, r := range outcomes {
		if r.Received > 0 {
			found = append(found, r)
		}
	}
	return found
}

func countUniqueIPs(results []PingResult) int {
	seen := make(map[string]bool, len(results))
	for _, r := range results {
		seen[r.IP.String()] = true
	}
	return len(seen)
}

func PingIPs(stopCh <-chan struct{}, ips []CompactIP, cp *Checkpoint, existingResults []PingResult) []PingResult {
	var results []PingResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	ports := tcpPorts()
	ipConcurrency := maxRoutines / len(ports)
	if ipConcurrency < 1 {
		ipConcurrency = 1
	}
	saveInterval := saveIntervalMode1 / len(ports)
	if saveInterval < 1 {
		saveInterval = 1
	}

	control := make(chan struct{}, ipConcurrency)
	total := len(ips)
	processedCount := 0
	baseIndex := 0
	if cp != nil {
		baseIndex = cp.ProgressIndex
	}

	timeoutMs := int(tcpConnectTimeout.Milliseconds())

	cyan := color.New(color.FgCyan)
	if len(ports) > 1 {
		cyan.Printf("Start latency test (Mode: TCP, Ports: %d, Range: 0 ~ %d ms, Packet Loss: 1.00)\n", len(ports), timeoutMs)
	} else {
		cyan.Printf("Start latency test (Mode: TCP, Port: %d, Range: 0 ~ %d ms, Packet Loss: 1.00)\n", ports[0], timeoutMs)
	}

	bar := newBar(total, "Available:", "")

	for _, cip := range ips {
		select {
		case <-stopCh:
			goto done
		case control <- struct{}{}:
		}

		wg.Add(1)
		go func(compIP CompactIP) {
			defer wg.Done()
			defer func() { <-control }()

			ipAddr := compIP.ToNetIPAddr()
			found := probePorts(ipAddr, ports)

			mu.Lock()
			processedCount++
			results = append(results, found...)
			bar.grow(1, strconv.Itoa(len(results)))
			if cp != nil && processedCount%saveInterval == 0 {
				cp.ProgressIndex = baseIndex + processedCount
				merged := make([]PingResult, 0, len(existingResults)+len(results))
				merged = append(merged, existingResults...)
				merged = append(merged, results...)
				cpCopy := *cp
				cpCopy.SetPingResults(merged)
				cpCopy.SaveAsync()
			}
			mu.Unlock()
		}(cip)
	}

done:
	wg.Wait()
	bar.done()

	sort.Slice(results, func(i, j int) bool {
		li, lj := results[i].GetLossRate(), results[j].GetLossRate()
		if li != lj {
			return li < lj
		}
		return results[i].Delay < results[j].Delay
	})

	fmt.Println()
	if len(ports) > 1 {
		color.New(color.FgGreen).Printf("Latency test completed: %d responsive IP:Port pairs found (%d unique IPs)\n\n", len(results), countUniqueIPs(results))
	} else {
		color.New(color.FgGreen).Printf("Latency test completed: %d responsive IPs found\n\n", len(results))
	}

	return results
}