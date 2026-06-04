package scanner

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/VividCortex/ewma"
	"github.com/fatih/color"
)

const (
	bufferSize      = 32768
	downloadURL     = "https://speed.cloudflare.com/__down?bytes=52428800"
	downloadTimeout = 10 * time.Second
	defaultTestNum  = 10
	minSpeed        = 0.0
)

type IPResult struct {
	IP            *net.IPAddr
	Sended        int
	Received      int
	LossRate      float32
	Delay         int
	DownloadSpeed float64
}

func getDialContext(ip *net.IPAddr) func(ctx context.Context, network, address string) (net.Conn, error) {
	var targetAddr string
	if isIPv4(ip.String()) {
		targetAddr = fmt.Sprintf("%s:%d", ip.String(), port)
	} else {
		targetAddr = fmt.Sprintf("[%s]:%d", ip.String(), port)
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{
			Timeout: downloadTimeout,
		}).DialContext(ctx, network, targetAddr)
	}
}

func downloadHandler(ip *net.IPAddr) float64 {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext:           getDialContext(ip),
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			DisableKeepAlives:     true,
		},
		Timeout: downloadTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return 0.0
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36")

	response, err := client.Do(req)
	if err != nil {
		return 0.0
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return 0.0
	}

	timeStart := time.Now()
	timeEnd := timeStart.Add(downloadTimeout)
	buffer := make([]byte, bufferSize)

	var (
		contentRead     int64 = 0
		lastContentRead int64 = 0
		timeSlice             = downloadTimeout / 100
		timeCounter           = 1
	)

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

	return e.Value() * 100 / downloadTimeout.Seconds()
}

func SpeedTest(stopCh <-chan struct{}, pingResults []PingResult) []IPResult {
	testCount := defaultTestNum
	testNum := testCount
	if len(pingResults) < testCount {
		testNum = len(pingResults)
		testCount = testNum
	}

	barPadding := "     "
	for i := 0; i < len(strconv.Itoa(len(pingResults))); i++ {
		barPadding += " "
	}

	color.New(color.FgCyan).Printf("Start download speed test (Minimum speed: %.2f MB/s, Number: %d, Queue: %d)\n", minSpeed, testCount, testNum)

	bar := newBar(testCount, barPadding, "")

	var results []IPResult

	for i := 0; i < testNum; i++ {
		select {
		case <-stopCh:
			goto done
		default:
		}

		pr := pingResults[i]
		speed := downloadHandler(pr.IP)

		if speed >= minSpeed {
			bar.grow(1, "")
			results = append(results, IPResult{
				IP:            pr.IP,
				Sended:        pr.Sended,
				Received:      pr.Received,
				LossRate:      pr.GetLossRate(),
				Delay:         int(pr.Delay.Milliseconds()),
				DownloadSpeed: speed,
			})
			if len(results) == testCount {
				break
			}
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
	color.New(color.FgGreen).Printf("Speed test completed: %d clean IPs found\n\n", len(results))
	return results
}