package trigger

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTemp(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	os.MkdirAll(filepath.Join(dir, ".nickai"), 0755)
	return func() { os.Setenv("HOME", orig) }
}

func TestAddAndLoad(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	err := Add(Trigger{
		ID:       "trig001",
		Symbol:   "BTC",
		Operator: "<",
		Target:   60000,
		Action:   Action{Side: "sell", Quantity: 0.5, Type: "market"},
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 trigger, got %d", len(loaded))
	}
	if loaded[0].Symbol != "BTC" {
		t.Errorf("Symbol: got %q, want BTC", loaded[0].Symbol)
	}
}

func TestActive(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	_ = Add(Trigger{ID: "a1", Symbol: "BTC", Fired: false})
	_ = Add(Trigger{ID: "a2", Symbol: "ETH", Fired: true})
	_ = Add(Trigger{ID: "a3", Symbol: "SOL", Fired: false})

	active, err := Active()
	if err != nil {
		t.Fatalf("Active failed: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("expected 2 active triggers, got %d", len(active))
	}
}

func TestMarkFired(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	_ = Add(Trigger{ID: "fire-me", Symbol: "BTC"})

	err := MarkFired("fire-me")
	if err != nil {
		t.Fatalf("MarkFired failed: %v", err)
	}

	loaded, _ := Load()
	if !loaded[0].Fired {
		t.Error("expected trigger to be marked as fired")
	}
}

func TestRemoveByPrefix(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	_ = Add(Trigger{ID: "abc123", Symbol: "BTC"})
	_ = Add(Trigger{ID: "def456", Symbol: "ETH"})

	err := Remove("abc")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	loaded, _ := Load()
	if len(loaded) != 1 {
		t.Fatalf("expected 1 trigger after remove, got %d", len(loaded))
	}
	if loaded[0].ID != "def456" {
		t.Errorf("wrong trigger remaining: %s", loaded[0].ID)
	}
}

func TestClear(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	_ = Add(Trigger{ID: "a", Symbol: "BTC"})
	_ = Add(Trigger{ID: "b", Symbol: "ETH"})

	err := Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	loaded, _ := Load()
	if len(loaded) != 0 {
		t.Errorf("expected 0 triggers after clear, got %d", len(loaded))
	}
}

func TestLoadMissingFile(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}
	if loaded != nil {
		t.Errorf("expected nil slice, got %v", loaded)
	}
}
