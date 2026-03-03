package alert

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

	// Create .nickai dir.
	os.MkdirAll(filepath.Join(dir, ".nickai"), 0o700)

	return dir, func() {
		os.Setenv("HOME", origHome)
	}
}

func TestAlertSerialization(t *testing.T) {
	a := Alert{
		Symbol:   "BTCUSDT",
		Operator: ">",
		Target:   70000.0,
		Created:  time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Alert
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Symbol != a.Symbol {
		t.Errorf("Symbol = %q, want %q", decoded.Symbol, a.Symbol)
	}
	if decoded.Operator != a.Operator {
		t.Errorf("Operator = %q, want %q", decoded.Operator, a.Operator)
	}
	if decoded.Target != a.Target {
		t.Errorf("Target = %v, want %v", decoded.Target, a.Target)
	}
}

func TestAlertTriggering(t *testing.T) {
	tests := []struct {
		name     string
		alert    Alert
		price    float64
		wantFire bool
	}{
		{
			name:     "above target, operator >",
			alert:    Alert{Symbol: "BTC", Operator: ">", Target: 70000},
			price:    71000,
			wantFire: true,
		},
		{
			name:     "below target, operator >",
			alert:    Alert{Symbol: "BTC", Operator: ">", Target: 70000},
			price:    69000,
			wantFire: false,
		},
		{
			name:     "below target, operator <",
			alert:    Alert{Symbol: "ETH", Operator: "<", Target: 3000},
			price:    2800,
			wantFire: true,
		},
		{
			name:     "above target, operator <",
			alert:    Alert{Symbol: "ETH", Operator: "<", Target: 3000},
			price:    3200,
			wantFire: false,
		},
		{
			name:     "exact match, operator >",
			alert:    Alert{Symbol: "SOL", Operator: ">", Target: 100},
			price:    100,
			wantFire: false,
		},
		{
			name:     "exact match, operator <",
			alert:    Alert{Symbol: "SOL", Operator: "<", Target: 100},
			price:    100,
			wantFire: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var fired bool
			switch tc.alert.Operator {
			case ">":
				fired = tc.price > tc.alert.Target
			case "<":
				fired = tc.price < tc.alert.Target
			}
			if fired != tc.wantFire {
				t.Errorf("alert %s %s %.0f with price %.0f: fired=%v, want %v",
					tc.alert.Symbol, tc.alert.Operator, tc.alert.Target,
					tc.price, fired, tc.wantFire)
			}
		})
	}
}

func TestLoadSavePersistence(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Load from nonexistent file returns empty.
	alerts, err := Load()
	if err != nil {
		t.Fatalf("Load (empty): %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("Load (empty) returned %d alerts, want 0", len(alerts))
	}

	// Save then reload.
	testAlerts := []Alert{
		{Symbol: "BTCUSDT", Operator: ">", Target: 70000, Created: time.Now()},
		{Symbol: "ETHUSDT", Operator: "<", Target: 2000, Created: time.Now()},
	}
	if err := Save(testAlerts); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("Load returned %d alerts, want 2", len(loaded))
	}
	if loaded[0].Symbol != "BTCUSDT" {
		t.Errorf("loaded[0].Symbol = %q, want BTCUSDT", loaded[0].Symbol)
	}
	if loaded[1].Target != 2000 {
		t.Errorf("loaded[1].Target = %v, want 2000", loaded[1].Target)
	}
}

func TestAddAndRemove(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	a1 := Alert{Symbol: "BTCUSDT", Operator: ">", Target: 70000, Created: time.Now()}
	a2 := Alert{Symbol: "ETHUSDT", Operator: "<", Target: 2000, Created: time.Now()}

	if err := Add(a1); err != nil {
		t.Fatalf("Add(a1): %v", err)
	}
	if err := Add(a2); err != nil {
		t.Fatalf("Add(a2): %v", err)
	}

	alerts, _ := Load()
	if len(alerts) != 2 {
		t.Fatalf("after Add x2: got %d alerts, want 2", len(alerts))
	}

	// Remove first alert.
	if err := Remove("BTCUSDT", ">", 70000); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	alerts, _ = Load()
	if len(alerts) != 1 {
		t.Fatalf("after Remove: got %d alerts, want 1", len(alerts))
	}
	if alerts[0].Symbol != "ETHUSDT" {
		t.Errorf("remaining alert = %q, want ETHUSDT", alerts[0].Symbol)
	}
}

func TestClear(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	Save([]Alert{
		{Symbol: "BTC", Operator: ">", Target: 100},
		{Symbol: "ETH", Operator: "<", Target: 50},
	})

	if err := Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	alerts, _ := Load()
	if len(alerts) != 0 {
		t.Errorf("after Clear: got %d alerts, want 0", len(alerts))
	}
}
