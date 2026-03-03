package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDisabled_NoOp(t *testing.T) {
	dir := t.TempDir()

	// Write a config with enabled=false.
	cfgData, _ := json.Marshal(Config{Enabled: false, RetainDays: 30})
	os.WriteFile(filepath.Join(dir, "telemetry.json"), cfgData, 0644)

	Init(dir)

	Record("error", "test", "should_not_record", fmt.Errorf("boom"), nil)
	RecordError("test", "also_skip", fmt.Errorf("boom"))
	Flush()

	// Events file should not exist.
	eventsPath := filepath.Join(dir, "telemetry_events.json")
	if _, err := os.Stat(eventsPath); err == nil {
		t.Fatal("expected no events file when telemetry is disabled")
	}
}

func TestInit_DefaultDisabled(t *testing.T) {
	dir := t.TempDir()
	Init(dir)

	global.mu.Lock()
	enabled := global.config.Enabled
	global.mu.Unlock()

	if enabled {
		t.Fatal("expected telemetry to be disabled by default")
	}
}

func TestEnabled_Record(t *testing.T) {
	dir := t.TempDir()
	Init(dir)
	SetEnabled(true)

	Record("info", "test", "action1", nil, map[string]string{"key": "val"})
	Record("error", "test", "action2", fmt.Errorf("oops"), nil)

	global.mu.Lock()
	count := len(global.events)
	global.mu.Unlock()

	if count != 2 {
		t.Fatalf("expected 2 events in buffer, got %d", count)
	}
}

func TestFlush_WritesDisk(t *testing.T) {
	dir := t.TempDir()
	Init(dir)
	SetEnabled(true)

	Record("info", "api", "test_call", nil, nil)
	Record("error", "mcp", "connect", fmt.Errorf("timeout"), nil)
	Flush()

	eventsPath := filepath.Join(dir, "telemetry_events.json")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("failed to read events file: %v", err)
	}

	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		t.Fatalf("failed to parse events: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events on disk, got %d", len(events))
	}

	if events[0].Category != "api" {
		t.Errorf("expected category 'api', got %q", events[0].Category)
	}
	if events[1].Error != "timeout" {
		t.Errorf("expected error 'timeout', got %q", events[1].Error)
	}

	// Flush again with more events — should append.
	Record("warn", "order", "risk_check", nil, nil)
	Flush()

	data, _ = os.ReadFile(eventsPath)
	json.Unmarshal(data, &events)
	if len(events) != 3 {
		t.Fatalf("expected 3 events after second flush, got %d", len(events))
	}
}

func TestPrune_RemovesOld(t *testing.T) {
	dir := t.TempDir()
	Init(dir)
	SetEnabled(true)

	// Set retain to 1 day.
	global.mu.Lock()
	global.config.RetainDays = 1
	global.mu.Unlock()

	// Write events: one old, one recent.
	old := Event{
		Timestamp: time.Now().Add(-72 * time.Hour),
		Level:     "error",
		Category:  "api",
		Action:    "old_call",
		Error:     "old error",
	}
	recent := Event{
		Timestamp: time.Now(),
		Level:     "info",
		Category:  "api",
		Action:    "recent_call",
	}

	eventsPath := filepath.Join(dir, "telemetry_events.json")
	data, _ := json.MarshalIndent([]Event{old, recent}, "", "  ")
	os.WriteFile(eventsPath, data, 0644)

	Prune()

	data, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("failed to read events after prune: %v", err)
	}
	var events []Event
	json.Unmarshal(data, &events)

	if len(events) != 1 {
		t.Fatalf("expected 1 event after prune, got %d", len(events))
	}
	if events[0].Action != "recent_call" {
		t.Errorf("expected recent event to survive, got action %q", events[0].Action)
	}
}

func TestSummary_Format(t *testing.T) {
	dir := t.TempDir()
	Init(dir)
	SetEnabled(true)

	Record("error", "mcp", "connect", fmt.Errorf("refused"), nil)
	Record("info", "api", "get_prices", nil, nil)
	Record("warn", "order", "risk_check", nil, nil)
	Flush()

	s := Summary()
	if !strings.Contains(s, "3 events") {
		t.Errorf("summary should mention 3 events, got: %s", s)
	}
	if !strings.Contains(s, "errors=1") {
		t.Errorf("summary should mention errors=1, got: %s", s)
	}
	if !strings.Contains(s, "refused") {
		t.Errorf("summary should include recent error text, got: %s", s)
	}
}

func TestRecordError_Helper(t *testing.T) {
	dir := t.TempDir()
	Init(dir)
	SetEnabled(true)

	RecordError("ai", "api_call", fmt.Errorf("rate limited"))

	global.mu.Lock()
	count := len(global.events)
	ev := global.events[0]
	global.mu.Unlock()

	if count != 1 {
		t.Fatalf("expected 1 event, got %d", count)
	}
	if ev.Level != "error" {
		t.Errorf("expected level 'error', got %q", ev.Level)
	}
	if ev.Category != "ai" {
		t.Errorf("expected category 'ai', got %q", ev.Category)
	}
	if ev.Error != "rate limited" {
		t.Errorf("expected error 'rate limited', got %q", ev.Error)
	}

	// RecordError with nil error should be a no-op.
	RecordError("ai", "nil_err", nil)
	global.mu.Lock()
	count = len(global.events)
	global.mu.Unlock()
	if count != 1 {
		t.Errorf("RecordError(nil) should be no-op, got %d events", count)
	}
}

func TestRecordLatency_Helper(t *testing.T) {
	dir := t.TempDir()
	Init(dir)
	SetEnabled(true)

	RecordLatency("api", "get_prices", 150*time.Millisecond)

	global.mu.Lock()
	count := len(global.events)
	ev := global.events[0]
	global.mu.Unlock()

	if count != 1 {
		t.Fatalf("expected 1 event, got %d", count)
	}
	if ev.Duration != 150 {
		t.Errorf("expected duration_ms=150, got %d", ev.Duration)
	}
	if ev.Level != "info" {
		t.Errorf("expected level 'info', got %q", ev.Level)
	}
}

func TestConcurrency(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	Init(dir)
	SetEnabled(true)

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			Record("info", "test", fmt.Sprintf("action_%d", n), nil, nil)
		}(i)
	}
	wg.Wait()

	global.mu.Lock()
	count := len(global.events)
	global.mu.Unlock()

	if count != 100 {
		t.Fatalf("expected 100 events from concurrent writes, got %d", count)
	}

	Flush()

	eventsPath := filepath.Join(dir, "telemetry_events.json")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("failed to read events after concurrent flush: %v", err)
	}
	var events []Event
	json.Unmarshal(data, &events)
	if len(events) != 100 {
		t.Fatalf("expected 100 events on disk, got %d", len(events))
	}
}
