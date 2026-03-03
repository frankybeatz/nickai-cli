package trigger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/nickai/cli/internal/safefile"
)

// Trigger represents a conditional trading rule that fires when a price condition is met.
type Trigger struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Operator  string    `json:"operator"` // ">" or "<"
	Target    float64   `json:"target"`
	Action    Action    `json:"action"`
	CreatedAt time.Time `json:"created_at"`
	Fired     bool      `json:"fired"`
}

// Action describes what trade to execute when a trigger fires.
type Action struct {
	Side     string  `json:"side"`     // "buy" or "sell"
	Quantity float64 `json:"quantity"`
	Type     string  `json:"type"`     // "market" or "limit"
	Price    float64 `json:"price,omitempty"`
}

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nickai", "triggers.json"), nil
}

// Load reads all triggers from ~/.nickai/triggers.json.
func Load() ([]Trigger, error) {
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
	var triggers []Trigger
	if err := json.Unmarshal(data, &triggers); err != nil {
		return nil, err
	}
	return triggers, nil
}

// Save writes all triggers to disk.
func Save(triggers []Trigger) error {
	path, err := storePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(triggers, "", "  ")
	if err != nil {
		return err
	}
	return safefile.AtomicWrite(path, data, 0o600)
}

// Add appends a trigger and saves.
func Add(t Trigger) error {
	path, _ := storePath()
	mu := safefile.Lock(path)
	mu.Lock()
	defer mu.Unlock()
	triggers, err := Load()
	if err != nil {
		return err
	}
	triggers = append(triggers, t)
	return Save(triggers)
}

// Remove deletes a trigger by ID prefix and saves.
func Remove(idPrefix string) error {
	path, _ := storePath()
	mu := safefile.Lock(path)
	mu.Lock()
	defer mu.Unlock()
	triggers, err := Load()
	if err != nil {
		return err
	}
	filtered := triggers[:0]
	for _, t := range triggers {
		if len(t.ID) >= len(idPrefix) && t.ID[:len(idPrefix)] == idPrefix {
			continue
		}
		filtered = append(filtered, t)
	}
	return Save(filtered)
}

// MarkFired sets a trigger's Fired flag and saves.
func MarkFired(id string) error {
	path, _ := storePath()
	mu := safefile.Lock(path)
	mu.Lock()
	defer mu.Unlock()
	triggers, err := Load()
	if err != nil {
		return err
	}
	for i := range triggers {
		if triggers[i].ID == id {
			triggers[i].Fired = true
			break
		}
	}
	return Save(triggers)
}

// Active returns all triggers that haven't fired yet.
func Active() ([]Trigger, error) {
	all, err := Load()
	if err != nil {
		return nil, err
	}
	var active []Trigger
	for _, t := range all {
		if !t.Fired {
			active = append(active, t)
		}
	}
	return active, nil
}

// Clear removes all triggers.
func Clear() error {
	return Save(nil)
}
