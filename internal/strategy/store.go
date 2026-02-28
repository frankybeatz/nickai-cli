package strategy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TWAPStrategy represents a time-weighted average price execution strategy.
type TWAPStrategy struct {
	ID          string    `json:"id"`
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"`
	TotalValue  float64   `json:"total_value"`
	Duration    string    `json:"duration"`      // "4h", "1h", "30m"
	IntervalSec int       `json:"interval_sec"`
	SliceCount  int       `json:"slice_count"`
	SliceValue  float64   `json:"slice_value"`
	Executed    int       `json:"executed"`
	Status      string    `json:"status"` // "active", "completed", "cancelled"
	CreatedAt   time.Time `json:"created_at"`
	NextSliceAt time.Time `json:"next_slice_at"`
	OrderIDs    []string  `json:"order_ids,omitempty"`
}

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nickai", "strategies.json"), nil
}

// Load reads all strategies from ~/.nickai/strategies.json.
func Load() ([]TWAPStrategy, error) {
	path, err := storePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var strategies []TWAPStrategy
	if err := json.Unmarshal(data, &strategies); err != nil {
		return nil, err
	}
	return strategies, nil
}

// Save writes all strategies to disk.
func Save(strategies []TWAPStrategy) error {
	path, err := storePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(strategies, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Add appends a strategy and saves.
func Add(s TWAPStrategy) error {
	strategies, err := Load()
	if err != nil {
		return err
	}
	strategies = append(strategies, s)
	return Save(strategies)
}

// Active returns all strategies with status "active".
func Active() ([]TWAPStrategy, error) {
	all, err := Load()
	if err != nil {
		return nil, err
	}
	var active []TWAPStrategy
	for _, s := range all {
		if s.Status == "active" {
			active = append(active, s)
		}
	}
	return active, nil
}

// Cancel sets a strategy's status to "cancelled" and saves.
func Cancel(id string) error {
	strategies, err := Load()
	if err != nil {
		return err
	}
	found := false
	for i := range strategies {
		if strategies[i].ID == id || (len(strategies[i].ID) >= len(id) && strategies[i].ID[:len(id)] == id) {
			strategies[i].Status = "cancelled"
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("strategy not found: %s", id)
	}
	return Save(strategies)
}

// MarkSliceExecuted records a completed slice and advances the next slice time.
func MarkSliceExecuted(id, orderID string) error {
	strategies, err := Load()
	if err != nil {
		return err
	}
	for i := range strategies {
		if strategies[i].ID == id {
			strategies[i].Executed++
			strategies[i].OrderIDs = append(strategies[i].OrderIDs, orderID)
			if strategies[i].Executed >= strategies[i].SliceCount {
				strategies[i].Status = "completed"
			} else {
				strategies[i].NextSliceAt = time.Now().Add(time.Duration(strategies[i].IntervalSec) * time.Second)
			}
			break
		}
	}
	return Save(strategies)
}

// ParseDuration parses "4h", "30m", "1h30m" into a time.Duration.
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// Try standard Go duration parsing first.
	if d, err := time.ParseDuration(s); err == nil {
		if d <= 0 {
			return 0, fmt.Errorf("duration must be positive")
		}
		return d, nil
	}

	// Handle simple formats like "4h", "30m", "2h30m".
	// Already handled by time.ParseDuration above, so if we get here
	// it's likely an integer with no unit — treat as minutes.
	if n, err := strconv.Atoi(s); err == nil {
		if n <= 0 {
			return 0, fmt.Errorf("duration must be positive")
		}
		return time.Duration(n) * time.Minute, nil
	}

	return 0, fmt.Errorf("invalid duration: %s (use e.g. 4h, 30m, 1h30m)", s)
}

// CalcSlices computes slice count and interval for a TWAP strategy.
// SliceCount = max(4, duration_minutes/15). Interval = duration/sliceCount.
func CalcSlices(duration time.Duration) (sliceCount int, intervalSec int) {
	minutes := int(duration.Minutes())
	sliceCount = minutes / 15
	if sliceCount < 4 {
		sliceCount = 4
	}
	intervalSec = int(duration.Seconds()) / sliceCount
	if intervalSec < 60 {
		intervalSec = 60
	}
	return
}
