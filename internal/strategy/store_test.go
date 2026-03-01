package strategy

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTemp(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	orig := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	os.MkdirAll(filepath.Join(dir, ".nickai"), 0755)
	return func() { os.Setenv("HOME", orig) }
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"4h", 4 * time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"1h30m", 90 * time.Minute, false},
		{"90", 90 * time.Minute, false}, // plain int → minutes
		{"", 0, true},
		{"0", 0, true},   // zero not allowed
		{"-5m", 0, true}, // negative not allowed
	}

	for _, tt := range tests {
		got, err := ParseDuration(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseDuration(%q): err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.expected {
			t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestCalcSlices(t *testing.T) {
	tests := []struct {
		duration   time.Duration
		wantSlices int
		wantMinSec int
	}{
		{4 * time.Hour, 16, 60},      // 240min / 15 = 16 slices
		{30 * time.Minute, 4, 60},    // min 4 slices
		{1 * time.Hour, 4, 60},       // 60/15 = 4 slices
		{15 * time.Minute, 4, 60},    // min 4 slices, min 60s interval
	}

	for _, tt := range tests {
		slices, interval := CalcSlices(tt.duration)
		if slices < tt.wantSlices {
			t.Errorf("CalcSlices(%v): slices=%d, want >=%d", tt.duration, slices, tt.wantSlices)
		}
		if interval < tt.wantMinSec {
			t.Errorf("CalcSlices(%v): interval=%d, want >=%d", tt.duration, interval, tt.wantMinSec)
		}
	}
}

func TestCalcSlicesMinimum(t *testing.T) {
	// Very short duration still gets minimum 4 slices.
	slices, _ := CalcSlices(5 * time.Minute)
	if slices < 4 {
		t.Errorf("expected at least 4 slices, got %d", slices)
	}
}

func TestAddAndLoad(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	err := Add(TWAPStrategy{
		ID:         "strat001",
		Symbol:     "ETH",
		Side:       "buy",
		TotalValue: 2000,
		Duration:   "4h",
		SliceCount: 8,
		SliceValue: 250,
		Status:     "active",
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 strategy, got %d", len(loaded))
	}
	if loaded[0].Symbol != "ETH" {
		t.Errorf("Symbol: got %q, want ETH", loaded[0].Symbol)
	}
}

func TestActive(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	_ = Add(TWAPStrategy{ID: "a1", Status: "active"})
	_ = Add(TWAPStrategy{ID: "a2", Status: "completed"})
	_ = Add(TWAPStrategy{ID: "a3", Status: "active"})

	active, err := Active()
	if err != nil {
		t.Fatalf("Active failed: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("expected 2 active strategies, got %d", len(active))
	}
}

func TestCancel(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	_ = Add(TWAPStrategy{ID: "cancel-me", Status: "active"})

	err := Cancel("cancel")
	if err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	loaded, _ := Load()
	if loaded[0].Status != "cancelled" {
		t.Errorf("expected cancelled, got %s", loaded[0].Status)
	}
}

func TestCancelNotFound(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	_ = Add(TWAPStrategy{ID: "abc", Status: "active"})
	err := Cancel("xyz")
	if err == nil {
		t.Error("expected error for non-existent strategy")
	}
}

func TestMarkSliceExecuted(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	_ = Add(TWAPStrategy{
		ID:         "exec-test",
		Status:     "active",
		SliceCount: 3,
		Executed:   0,
	})

	// Execute first slice.
	_ = MarkSliceExecuted("exec-test", "order-1")
	loaded, _ := Load()
	if loaded[0].Executed != 1 {
		t.Errorf("Executed = %d, want 1", loaded[0].Executed)
	}
	if len(loaded[0].OrderIDs) != 1 || loaded[0].OrderIDs[0] != "order-1" {
		t.Errorf("OrderIDs = %v, want [order-1]", loaded[0].OrderIDs)
	}
	if loaded[0].Status != "active" {
		t.Errorf("Status = %s, want active", loaded[0].Status)
	}

	// Execute remaining slices.
	_ = MarkSliceExecuted("exec-test", "order-2")
	_ = MarkSliceExecuted("exec-test", "order-3")
	loaded, _ = Load()
	if loaded[0].Status != "completed" {
		t.Errorf("expected completed after all slices, got %s", loaded[0].Status)
	}
	if loaded[0].Executed != 3 {
		t.Errorf("Executed = %d, want 3", loaded[0].Executed)
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
