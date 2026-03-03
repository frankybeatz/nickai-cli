package journal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/nickai/cli/internal/safefile"
)

// JournalEntry records a trade with AI rationale.
type JournalEntry struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Quantity  float64   `json:"quantity"`
	Price     float64   `json:"price"`
	Rationale string    `json:"rationale,omitempty"`
	Source    string    `json:"source"` // "ai", "manual", "trigger", "strategy"
	Timestamp time.Time `json:"timestamp"`
}

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nickai", "journal.json"), nil
}

// Load reads all journal entries from ~/.nickai/journal.json.
func Load() ([]JournalEntry, error) {
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
	var entries []JournalEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// Save writes all journal entries to disk.
func Save(entries []JournalEntry) error {
	path, err := storePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return safefile.AtomicWrite(path, data, 0o600)
}

// Add appends a journal entry and saves.
func Add(entry JournalEntry) error {
	path, _ := storePath()
	mu := safefile.Lock(path)
	mu.Lock()
	defer mu.Unlock()
	entries, err := Load()
	if err != nil {
		return err
	}
	entries = append(entries, entry)
	return Save(entries)
}

// All returns all journal entries.
func All() ([]JournalEntry, error) {
	return Load()
}
