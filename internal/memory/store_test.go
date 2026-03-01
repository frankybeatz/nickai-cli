package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func tempStore(t *testing.T) (*Store, func()) {
	t.Helper()
	dir := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	os.MkdirAll(filepath.Join(dir, ".nickai"), 0755)
	return &Store{}, func() { os.Setenv("HOME", orig) }
}

func TestAddAndSearch(t *testing.T) {
	s := &Store{}
	s.Add(Entry{
		ID:      "abc123",
		Type:    TypeInsight,
		Content: "Bitcoin tends to pump on Mondays",
		Tags:    []string{"bitcoin", "pattern"},
		Score:   5,
	})
	s.Add(Entry{
		ID:      "def456",
		Type:    TypePreference,
		Content: "I prefer limit orders",
		Tags:    []string{"trading"},
		Score:   3,
	})

	if len(s.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(s.Entries))
	}

	// Search by content.
	results := s.Search("bitcoin")
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'bitcoin', got %d", len(results))
	}
	if results[0].ID != "abc123" {
		t.Errorf("expected abc123, got %s", results[0].ID)
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	s := &Store{}
	s.Add(Entry{ID: "a", Content: "ETH is great", Tags: []string{"ethereum"}})

	results := s.Search("eth")
	if len(results) != 1 {
		t.Errorf("case-insensitive search: expected 1 result, got %d", len(results))
	}

	results = s.Search("ETHEREUM")
	if len(results) != 1 {
		t.Errorf("tag search: expected 1 result, got %d", len(results))
	}
}

func TestSearchUpdatesAccessedAt(t *testing.T) {
	s := &Store{}
	old := time.Now().Add(-24 * time.Hour)
	s.Add(Entry{ID: "a", Content: "test entry", AccessedAt: old})

	_ = s.Search("test")

	if !s.Entries[0].AccessedAt.After(old) {
		t.Error("expected AccessedAt to be updated after search")
	}
}

func TestRemoveByPrefix(t *testing.T) {
	s := &Store{}
	s.Add(Entry{ID: "abc123"})
	s.Add(Entry{ID: "def456"})

	s.Remove("abc")
	if len(s.Entries) != 1 {
		t.Fatalf("expected 1 entry after remove, got %d", len(s.Entries))
	}
	if s.Entries[0].ID != "def456" {
		t.Errorf("wrong entry remaining: %s", s.Entries[0].ID)
	}
}

func TestRemoveNoMatch(t *testing.T) {
	s := &Store{}
	s.Add(Entry{ID: "abc123"})
	s.Remove("xyz")
	if len(s.Entries) != 1 {
		t.Error("expected no removal when prefix doesn't match")
	}
}

func TestPrune(t *testing.T) {
	s := &Store{}
	now := time.Now()
	s.Add(Entry{ID: "low", Score: 1, AccessedAt: now.Add(-48 * time.Hour)})
	s.Add(Entry{ID: "high", Score: 10, AccessedAt: now})
	s.Add(Entry{ID: "mid", Score: 5, AccessedAt: now.Add(-24 * time.Hour)})

	s.Prune(2)
	if len(s.Entries) != 2 {
		t.Fatalf("expected 2 entries after prune, got %d", len(s.Entries))
	}
	// The lowest-scored entry should be pruned.
	for _, e := range s.Entries {
		if e.ID == "low" {
			t.Error("expected 'low' to be pruned")
		}
	}
}

func TestPruneNoopWhenUnderLimit(t *testing.T) {
	s := &Store{}
	s.Add(Entry{ID: "a"})
	s.Prune(10)
	if len(s.Entries) != 1 {
		t.Error("prune should be a no-op when under limit")
	}
}

func TestForPrompt(t *testing.T) {
	s := &Store{}
	now := time.Now()
	s.Add(Entry{
		ID:         "a",
		Type:       TypeInsight,
		Content:    "BTC pumps on Mondays",
		Score:      10,
		AccessedAt: now,
		CreatedAt:  now,
	})
	s.Add(Entry{
		ID:         "b",
		Type:       TypePreference,
		Content:    "Use limit orders",
		Score:      5,
		AccessedAt: now,
		CreatedAt:  now,
	})

	result := s.ForPrompt(1000)
	if result == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !strings.Contains(result, "[insight]") {
		t.Error("expected [insight] tag in prompt")
	}
	if !strings.Contains(result, "BTC pumps on Mondays") {
		t.Error("expected content in prompt")
	}
}

func TestForPromptEmpty(t *testing.T) {
	s := &Store{}
	if s.ForPrompt(1000) != "" {
		t.Error("expected empty string for empty store")
	}
}

func TestForPromptTruncation(t *testing.T) {
	s := &Store{}
	// Add enough entries to exceed a small token limit.
	for i := 0; i < 50; i++ {
		s.Add(Entry{
			ID:      strings.Repeat("x", 10),
			Type:    TypeContext,
			Content: strings.Repeat("word ", 20),
			Score:   i,
		})
	}
	result := s.ForPrompt(50) // ~200 chars
	// Should be truncated.
	if len(result) > 300 {
		t.Errorf("expected truncation, got %d chars", len(result))
	}
}

func TestSaveAndLoad(t *testing.T) {
	s, cleanup := tempStore(t)
	defer cleanup()

	s.Add(Entry{ID: "abc", Type: TypeInsight, Content: "test memory"})
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded.Entries))
	}
	if loaded.Entries[0].Content != "test memory" {
		t.Errorf("content mismatch: %q", loaded.Entries[0].Content)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, cleanup := tempStore(t)
	defer cleanup()

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}
	if len(loaded.Entries) != 0 {
		t.Errorf("expected empty store, got %d entries", len(loaded.Entries))
	}
}
