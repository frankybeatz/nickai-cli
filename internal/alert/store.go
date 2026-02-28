package alert

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Alert represents a persistent price alert.
type Alert struct {
	Symbol   string    `json:"symbol"`
	Operator string    `json:"operator"` // ">" or "<"
	Target   float64   `json:"target"`
	Created  time.Time `json:"created_at"`
}

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nickai", "alerts.json"), nil
}

// Load reads all alerts from ~/.nickai/alerts.json.
// Returns an empty slice if the file doesn't exist.
func Load() ([]Alert, error) {
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
	var alerts []Alert
	if err := json.Unmarshal(data, &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}

// Save writes all alerts to disk.
func Save(alerts []Alert) error {
	path, err := storePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(alerts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Add appends an alert and saves.
func Add(a Alert) error {
	alerts, err := Load()
	if err != nil {
		return err
	}
	alerts = append(alerts, a)
	return Save(alerts)
}

// Remove deletes a matching alert and saves.
func Remove(symbol, op string, target float64) error {
	alerts, err := Load()
	if err != nil {
		return err
	}
	filtered := alerts[:0]
	removed := false
	for _, a := range alerts {
		if !removed && a.Symbol == symbol && a.Operator == op && a.Target == target {
			removed = true
			continue
		}
		filtered = append(filtered, a)
	}
	return Save(filtered)
}

// Clear deletes all alerts.
func Clear() error {
	return Save(nil)
}
