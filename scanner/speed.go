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
	uploadURL       = "https://speed.cloudflare.com/__up"
	downloadTimeout = 10 * time.Second
	uploadTimeout   = 10 * time.Second
	uploadBytes     = 26214400
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
	UploadSpeed   float64
}

type uploadReader struct {
	total       int64
	remaining   int64
	sentTotal   int64
	lastSent    int64
	timeStart   time.Time
	timeSlice   time.Duration
	timeCounter int
	deadline    time.Time
	e           ewma.MovingAverage
	corrected   bool
}

func newUploadReader(total int64, timeout time.Duration) *uploadReader {
	now := time.Now()
	return &uploadReader{
		total:     total,
		remaining: total,
		timeStart: now,
		timeSlice: timeout / 100,
		timeCounter: 1,
		deadline:  now.Add(timeout),
		e:         ewma.NewMovingAverage(),
	}
}

func (r *uploadReader) applyFinalCorrection(now time.Time) {
	if r.corrected {
		return
	}
	r.corrected = true
	lastSlice := r.timeStart.Add(r.timeSlice * time.Duration(r.timeCounter-1))
	elapsed := float64(now.Sub(lastSlice))
	sliceDuration := float64(r.timeSlice)
	if elapsed > 0 && sliceDuration > 0 {
		ratio := elapsed / sliceDuration
		if ratio > 0 {
			r.e.Add(float64(r.sentTotal-r.lastSent) / ratio)
		}
	}
}

func (r *uploadReader) Read(p []byte) (int, error) {
	now := time.Now()
	nextTime := r.timeStart.Add(r.timeSlice * time.Duration(r.timeCounter))
	if now.After(nextTime) {
		r.timeCounter++
		r.e.Add(float64(r.sentTotal - r.lastSent))
		r.lastSent = r.sentTotal
	}
	if now.After(r.deadline) || r.remaining <= 0 {
		r.applyFinalCorrection(now)
		return 0, io.EOF
	}
	n := len(p)
	if int64(n) > r.remaining {
		n = int(r.remaining)
	}
	for i := 0; i < n; i++ {
		p[i] = 0xAB
	}
	r.sentTotal += int64(n)
	r.remaining -= int64(n)
	return n, nil
}

func (r *uploadReader) speedBytesPerSec(timeout time.Duration) float64 {
	return r.e.Value() * 100 / timeout.Seconds()
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

func uploadHandler(ip *net.IPAddr) float64 {
	reader := newUploadReader(uploadBytes, uploadTimeout)
	client := &http.Client{
		Transport: &http.Transport{
			DialContext:           getDialContext(ip),
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			DisableKeepAlives:     true,
		},
		Timeout: uploadTimeout + 5*time.Second,
	}

	req, err := http.NewRequest("POST", uploadURL, reader)
	if err != nil {
		return 0.0
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = uploadBytes

	response, err := client.Do(req)
	speed := reader.speedBytesPerSec(uploadTimeout)
	if err != nil {
		return speed
	}
	defer response.Body.Close()
	io.Copy(io.Discard, response.Body)

	if response.StatusCode != 200 && response.StatusCode != 201 {
		return 0.0
	}

	return speed
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

	color.New(color.FgCyan).Printf("Start download & upload speed test (Minimum speed: %.2f MB/s, Number: %d, Queue: %d)\n", minSpeed, testCount, testNum)

	bar := newBar(testCount, barPadding, "")

	var results []IPResult

	for i := 0; i < testNum; i++ {
		select {
		case <-stopCh:
			goto done
		default:
		}

		pr := pingResults[i]
		downloadSpeed := downloadHandler(pr.IP)

		if downloadSpeed >= minSpeed {
			uploadSpeed := uploadHandler(pr.IP)
			bar.grow(1, "")
			results = append(results, IPResult{
				IP:            pr.IP,
				Sended:        pr.Sended,
				Received:      pr.Received,
				LossRate:      pr.GetLossRate(),
				Delay:         int(pr.Delay.Milliseconds()),
				DownloadSpeed: downloadSpeed,
				UploadSpeed:   uploadSpeed,
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
			return results[i].DownloadSpeed+results[i].UploadSpeed > results[j].DownloadSpeed+results[j].UploadSpeed
		})
	}

	fmt.Println()
	color.New(color.FgGreen).Printf("Speed test completed: %d clean IPs found\n\n", len(results))
	return results
}