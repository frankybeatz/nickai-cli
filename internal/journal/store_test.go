package journal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	os.MkdirAll(filepath.Join(dir, ".nickai"), 0o700)
	return dir, func() {
		os.Setenv("HOME", origHome)
	}
}

func TestJournalEntrySerialization(t *testing.T) {
	entry := JournalEntry{
		ID:        "j-001",
		OrderID:   "o-123",
		Symbol:    "BTCUSDT",
		Side:      "buy",
		Quantity:  0.5,
		Price:     68000.50,
		Rationale: "RSI oversold, support at 67k",
		Source:    "ai",
		Timestamp: time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded JournalEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.ID != entry.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, entry.ID)
	}
	if decoded.Symbol != entry.Symbol {
		t.Errorf("Symbol = %q, want %q", decoded.Symbol, entry.Symbol)
	}
	if decoded.Side != entry.Side {
		t.Errorf("Side = %q, want %q", decoded.Side, entry.Side)
	}
	if decoded.Quantity != entry.Quantity {
		t.Errorf("Quantity = %v, want %v", decoded.Quantity, entry.Quantity)
	}
	if decoded.Price != entry.Price {
		t.Errorf("Price = %v, want %v", decoded.Price, entry.Price)
	}
	if decoded.Rationale != entry.Rationale {
		t.Errorf("Rationale = %q, want %q", decoded.Rationale, entry.Rationale)
	}
	if decoded.Source != entry.Source {
		t.Errorf("Source = %q, want %q", decoded.Source, entry.Source)
	}
}

func TestJournalEntryOmitEmptyRationale(t *testing.T) {
	entry := JournalEntry{
		ID:     "j-002",
		Symbol: "ETHUSDT",
		Side:   "sell",
		Source: "manual",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Rationale should be omitted from JSON when empty.
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	if _, exists := raw["rationale"]; exists {
		t.Error("empty Rationale should be omitted from JSON (omitempty tag)")
	}
}

func TestLoadSavePersistence(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Load from nonexistent file returns empty.
	entries, err := Load()
	if err != nil {
		t.Fatalf("Load (empty): %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Load (empty) returned %d entries, want 0", len(entries))
	}

	// Save then reload.
	testEntries := []JournalEntry{
		{ID: "j-1", Symbol: "BTCUSDT", Side: "buy", Quantity: 1.0, Price: 65000, Source: "ai", Timestamp: time.Now()},
		{ID: "j-2", Symbol: "ETHUSDT", Side: "sell", Quantity: 10.0, Price: 3500, Source: "manual", Timestamp: time.Now()},
	}
	if err := Save(testEntries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("Load returned %d entries, want 2", len(loaded))
	}
	if loaded[0].ID != "j-1" {
		t.Errorf("loaded[0].ID = %q, want j-1", loaded[0].ID)
	}
	if loaded[1].Side != "sell" {
		t.Errorf("loaded[1].Side = %q, want sell", loaded[1].Side)
	}
}

func TestAddAndAll(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	e1 := JournalEntry{ID: "j-10", Symbol: "SOLUSDT", Side: "buy", Quantity: 100, Price: 150, Source: "ai", Timestamp: time.Now()}
	e2 := JournalEntry{ID: "j-11", Symbol: "BTCUSDT", Side: "sell", Quantity: 0.1, Price: 70000, Source: "strategy", Timestamp: time.Now()}

	if err := Add(e1); err != nil {
		t.Fatalf("Add(e1): %v", err)
	}
	if err := Add(e2); err != nil {
		t.Fatalf("Add(e2): %v", err)
	}

	all, err := All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("All returned %d entries, want 2", len(all))
	}
	if all[0].ID != "j-10" {
		t.Errorf("all[0].ID = %q, want j-10", all[0].ID)
	}
	if all[1].ID != "j-11" {
		t.Errorf("all[1].ID = %q, want j-11", all[1].ID)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	// .nickai dir does NOT exist yet -- Save should create it.
	entries := []JournalEntry{
		{ID: "j-x", Symbol: "BTC", Side: "buy", Source: "manual"},
	}
	if err := Save(entries); err != nil {
		t.Fatalf("Save (create dir): %v", err)
	}

	// Verify directory was created.
	info, err := os.Stat(filepath.Join(dir, ".nickai"))
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal(".nickai is not a directory")
	}
}
