package scanner

import (
	"encoding/json"
	"hash/fnv"
	"io"
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
	Mode           int             `json:"mode"`
	Workers        int             `json:"workers"`
	Phase          CheckpointPhase `json:"phase"`
	Completed      bool            `json:"completed"`
	ProgressIndex  int             `json:"progress_index"`
	TotalIPs       int             `json:"total_ips"`
	Seed           int64           `json:"seed"`
	IPRangesHash   uint64          `json:"ip_ranges_hash"`
	PingResults    []cpPingResult  `json:"ping_results"`
	SavedAt        string          `json:"saved_at"`
}

var asyncSaveMu sync.Mutex
var asyncSaveRunning bool

func ComputeFileHash(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	h := fnv.New64a()
	if _, err := io.Copy(h, f); err != nil {
		return 0, err
	}
	return h.Sum64(), nil
}

func NewCheckpoint(mode, workers int, totalIPs int, seed int64, ipRangesHash uint64) *Checkpoint {
	return &Checkpoint{
		Mode:         mode,
		Workers:      workers,
		Phase:        PhasePing,
		TotalIPs:     totalIPs,
		Seed:         seed,
		IPRangesHash: ipRangesHash,
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

type LoadResult int

const (
	LoadResultNone        LoadResult = iota
	LoadResultOK
	LoadResultHashChanged
)

func LoadCheckpointChecked(ipRangesPath string) (*Checkpoint, LoadResult) {
	data, err := os.ReadFile(checkpointFile)
	if err != nil {
		return nil, LoadResultNone
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		os.Remove(checkpointFile)
		return nil, LoadResultNone
	}

	if cp.Completed || cp.TotalIPs == 0 || cp.Seed == 0 {
		return nil, LoadResultNone
	}

	currentHash, err := ComputeFileHash(ipRangesPath)
	if err != nil {
		return &cp, LoadResultOK
	}

	if cp.IPRangesHash != 0 && currentHash != cp.IPRangesHash {
		os.Remove(checkpointFile)
		os.Remove(checkpointFile + ".tmp")
		return nil, LoadResultHashChanged
	}

	return &cp, LoadResultOK
}

func LoadCheckpoint() *Checkpoint {
	cp, _ := LoadCheckpointChecked("")
	return cp
}

func DeleteCheckpoint() {
	os.Remove(checkpointFile)
	os.Remove(checkpointFile + ".tmp")
}