package automation

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

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		{"hourly", 3600, false},
		{"daily", 86400, false},
		{"weekly", 604800, false},
		{"every 6h", 21600, false},
		{"every 30m", 1800, false},
		{"every 90s", 90, false},
		{"6h", 21600, false},
		{"30m", 1800, false},
		{"90s", 90, false},
		{"", 0, true},
		{"bogus", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseSchedule(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseSchedule(%q): err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.expected {
			t.Errorf("ParseSchedule(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestGetIndicatorValue(t *testing.T) {
	snap := IndicatorSnapshot{
		RSI:            65.5,
		MACD:           1.2,
		MACDSignal:     0.8,
		MACDHistogram:  0.4,
		BollingerUpper: 110,
		BollingerLower: 90,
		SMA20:          100,
		SMA50:          98,
		EMA12:          101,
		EMA26:          99,
		Price:          105,
	}

	tests := []struct {
		indicator string
		expected  float64
	}{
		{"rsi", 65.5},
		{"RSI", 65.5}, // case insensitive
		{"macd", 1.2},
		{"price", 105},
		{"sma20", 100},
		{"unknown", 0},
	}

	for _, tt := range tests {
		got := GetIndicatorValue(tt.indicator, snap)
		if got != tt.expected {
			t.Errorf("GetIndicatorValue(%q) = %f, want %f", tt.indicator, got, tt.expected)
		}
	}
}

func TestEvalIndicatorConditions(t *testing.T) {
	snap := IndicatorSnapshot{RSI: 25, Price: 100, MACD: -0.5}
	prevSnap := &IndicatorSnapshot{RSI: 35, Price: 95, MACD: 0.5}

	// RSI < 30 → true.
	conds := []IndicatorCondition{{Indicator: "rsi", Operator: "<", Value: 30}}
	if !EvalIndicatorConditions(conds, snap, prevSnap) {
		t.Error("expected RSI < 30 to be true")
	}

	// RSI > 50 → false.
	conds = []IndicatorCondition{{Indicator: "rsi", Operator: ">", Value: 50}}
	if EvalIndicatorConditions(conds, snap, prevSnap) {
		t.Error("expected RSI > 50 to be false")
	}

	// MACD crosses below 0 (prev=0.5, curr=-0.5).
	conds = []IndicatorCondition{{Indicator: "macd", Operator: "crosses_below", Value: 0}}
	if !EvalIndicatorConditions(conds, snap, prevSnap) {
		t.Error("expected MACD crosses_below 0 to be true")
	}

	// Crosses_above without prev snapshot → false.
	conds = []IndicatorCondition{{Indicator: "macd", Operator: "crosses_above", Value: 0}}
	if EvalIndicatorConditions(conds, snap, nil) {
		t.Error("expected crosses_above to be false without prev snapshot")
	}
}

func TestEvalMultipleConditionsAND(t *testing.T) {
	snap := IndicatorSnapshot{RSI: 25, Price: 100}

	// Both true.
	conds := []IndicatorCondition{
		{Indicator: "rsi", Operator: "<", Value: 30},
		{Indicator: "price", Operator: "<", Value: 110},
	}
	if !EvalIndicatorConditions(conds, snap, nil) {
		t.Error("expected both conditions to pass")
	}

	// One false.
	conds = []IndicatorCondition{
		{Indicator: "rsi", Operator: "<", Value: 30},
		{Indicator: "price", Operator: ">", Value: 110},
	}
	if EvalIndicatorConditions(conds, snap, nil) {
		t.Error("expected AND to fail when one condition is false")
	}
}

func TestStateMachine(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	_ = Add(AutoRule{
		ID:          "rule001",
		Description: "Test rule",
		Type:        RuleSchedule,
		Schedule:    "daily",
		Status:      "active",
		MaxFires:    3,
	})

	// Pause.
	err := Pause("rule")
	if err != nil {
		t.Fatalf("Pause failed: %v", err)
	}
	rules, _ := Load()
	if rules[0].Status != "paused" {
		t.Errorf("expected paused, got %s", rules[0].Status)
	}

	// Resume.
	err = Resume("rule")
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	rules, _ = Load()
	if rules[0].Status != "active" {
		t.Errorf("expected active, got %s", rules[0].Status)
	}

	// Fire 3 times (max fires).
	for i := 0; i < 3; i++ {
		_ = MarkFired("rule")
	}
	rules, _ = Load()
	if rules[0].Status != "completed" {
		t.Errorf("expected completed after max fires, got %s", rules[0].Status)
	}
	if rules[0].FireCount != 3 {
		t.Errorf("FireCount = %d, want 3", rules[0].FireCount)
	}
}

func TestActive(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	_ = Add(AutoRule{ID: "a1", Status: "active"})
	_ = Add(AutoRule{ID: "a2", Status: "paused"})
	_ = Add(AutoRule{ID: "a3", Status: "active"})

	active, err := Active()
	if err != nil {
		t.Fatalf("Active failed: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("expected 2 active rules, got %d", len(active))
	}
}

func TestRemove(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	_ = Add(AutoRule{ID: "abc123", Status: "active"})
	_ = Add(AutoRule{ID: "def456", Status: "active"})

	err := Remove("abc")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	rules, _ := Load()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
}

func TestMarkFiredUpdatesTime(t *testing.T) {
	cleanup := setupTemp(t)
	defer cleanup()

	_ = Add(AutoRule{ID: "r1", Status: "active", MaxFires: 0})
	before := time.Now()
	_ = MarkFired("r1")

	rules, _ := Load()
	if rules[0].LastFired.Before(before) {
		t.Error("expected LastFired to be updated")
	}
}
