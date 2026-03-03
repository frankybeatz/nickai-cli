package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nickai/cli/internal/safefile"
)

// MemoryType categorises a memory entry.
type MemoryType string

const (
	TypeInsight    MemoryType = "insight"    // trade learnings
	TypePreference MemoryType = "preference" // user habits
	TypeContext    MemoryType = "context"    // key facts user shared
)

// Entry is a single memory record.
type Entry struct {
	ID         string     `json:"id"`
	Type       MemoryType `json:"type"`
	Content    string     `json:"content"`
	Tags       []string   `json:"tags,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	AccessedAt time.Time  `json:"accessed_at"`
	Score      int        `json:"score"`
}

// Store holds all persistent memory entries.
type Store struct {
	Entries []Entry `json:"entries"`
}

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nickai", "memory.json"), nil
}

// Load reads the memory store from ~/.nickai/memory.json.
// Returns an empty Store if the file does not exist.
func Load() (*Store, error) {
	path, err := storePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Store{}, nil
		}
		return nil, err
	}
	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Save writes the memory store to disk.
func (s *Store) Save() error {
	path, err := storePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return safefile.AtomicWrite(path, data, 0o600)
}

// Add appends an entry to the store.
func (s *Store) Add(entry Entry) {
	s.Entries = append(s.Entries, entry)
}

// Remove deletes the first entry whose ID starts with idPrefix.
func (s *Store) Remove(idPrefix string) {
	for i, e := range s.Entries {
		if strings.HasPrefix(e.ID, idPrefix) {
			s.Entries = append(s.Entries[:i], s.Entries[i+1:]...)
			return
		}
	}
}

// Search returns entries whose Content or Tags contain query (case-insensitive).
// It updates AccessedAt on every match.
func (s *Store) Search(query string) []Entry {
	q := strings.ToLower(query)
	var results []Entry
	now := time.Now()
	for i, e := range s.Entries {
		if strings.Contains(strings.ToLower(e.Content), q) {
			s.Entries[i].AccessedAt = now
			results = append(results, s.Entries[i])
			continue
		}
		for _, tag := range e.Tags {
			if strings.Contains(strings.ToLower(tag), q) {
				s.Entries[i].AccessedAt = now
				results = append(results, s.Entries[i])
				break
			}
		}
	}
	return results
}

// sortByScoreThenAccessed sorts entries by Score descending, then AccessedAt descending.
func sortByScoreThenAccessed(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}
		return entries[i].AccessedAt.After(entries[j].AccessedAt)
	})
}

// Prune trims the store to maxEntries by keeping the highest-scored,
// most-recently-accessed entries.
func (s *Store) Prune(maxEntries int) {
	if len(s.Entries) <= maxEntries {
		return
	}
	sortByScoreThenAccessed(s.Entries)
	s.Entries = s.Entries[:maxEntries]
}

// ForPrompt formats entries for system-prompt injection.
// Entries are sorted by Score desc then AccessedAt desc, formatted as:
//
//	[type] content (YYYY-MM-DD)
//
// The output is truncated at approximately maxTokens * 4 characters.
// Returns an empty string when there are no entries.
func (s *Store) ForPrompt(maxTokens int) string {
	if len(s.Entries) == 0 {
		return ""
	}

	// Work on a copy so we don't reorder the real slice.
	sorted := make([]Entry, len(s.Entries))
	copy(sorted, s.Entries)
	sortByScoreThenAccessed(sorted)

	maxChars := maxTokens * 4
	var b strings.Builder
	for i, e := range sorted {
		line := fmt.Sprintf("[%s] %s (%s)", e.Type, e.Content, e.CreatedAt.Format("2006-01-02"))
		if i > 0 {
			// Account for the newline separator.
			if b.Len()+1+len(line) > maxChars {
				break
			}
			b.WriteByte('\n')
		} else {
			if len(line) > maxChars {
				b.WriteString(line[:maxChars])
				return b.String()
			}
		}
		b.WriteString(line)
		if b.Len() >= maxChars {
			return b.String()[:maxChars]
		}
	}
	return b.String()
}
