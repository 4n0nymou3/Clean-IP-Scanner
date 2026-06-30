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

var port = defaultPort

func SetPort(p int) {
	if p <= 0 || p > 65535 {
		port = defaultPort
		return
	}
	port = p
}

type PingResult struct {
	IP       *net.IPAddr
	Sended   int
	Received int
	Delay    time.Duration
}

func (p *PingResult) GetLossRate() float32 {
	lost := p.Sended - p.Received
	return float32(lost) / float32(p.Sended)
}

func tcping(ip *net.IPAddr) (bool, time.Duration) {
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

func checkConnection(ip *net.IPAddr) (recv int, totalDelay time.Duration) {
	for i := 0; i < defaultPingTimes; i++ {
		if ok, d := tcping(ip); ok {
			recv++
			totalDelay += d
		}
	}
	return
}

func PingIPs(stopCh <-chan struct{}, ips []CompactIP, cp *Checkpoint, existingResults []PingResult) []PingResult {
	var results []PingResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	control := make(chan struct{}, maxRoutines)
	total := len(ips)
	processedCount := 0
	baseIndex := 0
	if cp != nil {
		baseIndex = cp.ProgressIndex
	}

	timeoutMs := int(tcpConnectTimeout.Milliseconds())

	cyan := color.New(color.FgCyan)
	cyan.Printf("Start latency test (Mode: TCP, Port: %d, Range: 0 ~ %d ms, Packet Loss: 1.00)\n", port, timeoutMs)

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
			recv, totalDelay := checkConnection(ipAddr)

			mu.Lock()
			processedCount++
			nowAble := len(results)
			if recv != 0 {
				nowAble++
			}
			bar.grow(1, strconv.Itoa(nowAble))
			if recv > 0 {
				avg := totalDelay / time.Duration(recv)
				results = append(results, PingResult{
					IP:       ipAddr,
					Sended:   defaultPingTimes,
					Received: recv,
					Delay:    avg,
				})
			}
			if cp != nil && processedCount%saveIntervalMode1 == 0 {
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
	color.New(color.FgGreen).Printf("Latency test completed: %d responsive IPs found\n\n", len(results))

	return results
}