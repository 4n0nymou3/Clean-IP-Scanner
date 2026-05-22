package scanner

import (
	"encoding/json"
	"net"
	"os"
	"sync"
	"time"
)

const (
	checkpointFile    = "scan_checkpoint.json"
	saveIntervalMode1 = 2000
	saveIntervalMode2 = 500
)

type CheckpointPhase string

const (
	PhasePing  CheckpointPhase = "ping"
	PhaseSpeed CheckpointPhase = "speed"
	PhaseDone  CheckpointPhase = "done"
)

type cpPingResult struct {
	IP       string `json:"ip"`
	Sended   int    `json:"sended"`
	Received int    `json:"received"`
	DelayMs  int64  `json:"delay_ms"`
}

type Checkpoint struct {
	Mode          int             `json:"mode"`
	Workers       int             `json:"workers"`
	Phase         CheckpointPhase `json:"phase"`
	Completed     bool            `json:"completed"`
	ProgressIndex int             `json:"progress_index"`
	TotalIPs      int             `json:"total_ips"`
	Seed          int64           `json:"seed"`
	PingResults   []cpPingResult  `json:"ping_results"`
	SavedAt       string          `json:"saved_at"`
}

var asyncSaveMu sync.Mutex
var asyncSaveRunning bool

func NewCheckpoint(mode, workers int, totalIPs int, seed int64) *Checkpoint {
	return &Checkpoint{
		Mode:     mode,
		Workers:  workers,
		Phase:    PhasePing,
		TotalIPs: totalIPs,
		Seed:     seed,
	}
}

func (c *Checkpoint) SetPingResults(results []PingResult) {
	c.PingResults = make([]cpPingResult, len(results))
	for i, r := range results {
		c.PingResults[i] = cpPingResult{
			IP:       r.IP.String(),
			Sended:   r.Sended,
			Received: r.Received,
			DelayMs:  r.Delay.Milliseconds(),
		}
	}
}

func (c *Checkpoint) GetPingResults() []PingResult {
	results := make([]PingResult, 0, len(c.PingResults))
	for _, r := range c.PingResults {
		ipAddr, err := net.ResolveIPAddr("ip", r.IP)
		if err != nil {
			continue
		}
		results = append(results, PingResult{
			IP:       ipAddr,
			Sended:   r.Sended,
			Received: r.Received,
			Delay:    time.Duration(r.DelayMs) * time.Millisecond,
		})
	}
	return results
}

func (c *Checkpoint) save() error {
	c.SavedAt = time.Now().Format("2006-01-02 15:04:05")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := checkpointFile + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, checkpointFile)
}

func (c *Checkpoint) Save() {
	c.save()
}

func (c *Checkpoint) SaveAsync() {
	asyncSaveMu.Lock()
	if asyncSaveRunning {
		asyncSaveMu.Unlock()
		return
	}
	asyncSaveRunning = true
	snapshot := *c
	asyncSaveMu.Unlock()

	go func() {
		snapshot.save()
		asyncSaveMu.Lock()
		asyncSaveRunning = false
		asyncSaveMu.Unlock()
	}()
}

func (c *Checkpoint) MarkPingDone(allPingResults []PingResult) {
	c.Phase = PhaseSpeed
	c.ProgressIndex = c.TotalIPs
	c.SetPingResults(allPingResults)
	c.save()
}

func (c *Checkpoint) MarkCompleted() {
	c.Completed = true
	c.Phase = PhaseDone
	c.save()
}

func LoadCheckpoint() *Checkpoint {
	data, err := os.ReadFile(checkpointFile)
	if err != nil {
		return nil
	}
	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil
	}
	if cp.Completed || cp.TotalIPs == 0 || cp.Seed == 0 {
		return nil
	}
	return &cp
}

func DeleteCheckpoint() {
	os.Remove(checkpointFile)
	os.Remove(checkpointFile + ".tmp")
}