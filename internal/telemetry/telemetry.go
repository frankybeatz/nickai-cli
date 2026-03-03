package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Event represents a single telemetry event.
type Event struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`              // "error", "warn", "info"
	Category  string            `json:"category"`           // "mcp", "api", "order", "ai"
	Action    string            `json:"action"`
	Error     string            `json:"error,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
	Duration  int64             `json:"duration_ms,omitempty"`
}

// Config controls telemetry behavior.
type Config struct {
	Enabled    bool `json:"enabled"`
	ErrorsOnly bool `json:"errors_only"`
	RetainDays int  `json:"retain_days"`
}

// Collector accumulates telemetry events in memory and flushes to disk.
type Collector struct {
	mu        sync.Mutex
	config    Config
	configDir string
	events    []Event
}

// global collector instance.
var global *Collector

// Init initializes the global telemetry collector. configDir is the directory
// for config and event files (defaults to ~/.nickai if empty).
func Init(configDir string) {
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			// Cannot determine home dir — create a disabled collector.
			global = &Collector{config: Config{Enabled: false}}
			return
		}
		configDir = filepath.Join(home, ".nickai")
	}

	c := &Collector{configDir: configDir}
	c.loadConfig()
	global = c
}

// loadConfig reads telemetry.json from configDir. If absent, defaults apply.
func (c *Collector) loadConfig() {
	path := filepath.Join(c.configDir, "telemetry.json")
	data, err := os.ReadFile(path)
	if err != nil {
		// Default: enabled, retain 30 days.
		c.config = Config{Enabled: true, RetainDays: 30}
		return
	}
	if err := json.Unmarshal(data, &c.config); err != nil {
		c.config = Config{Enabled: true, RetainDays: 30}
		return
	}
	if c.config.RetainDays <= 0 {
		c.config.RetainDays = 30
	}
}

// saveConfig persists the current config to telemetry.json.
func (c *Collector) saveConfig() error {
	if err := os.MkdirAll(c.configDir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c.config, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(c.configDir, "telemetry.json")
	return atomicWrite(path, data, 0644)
}

// Record logs a telemetry event. No-op when disabled.
func Record(level, category, action string, err error, meta map[string]string) {
	if global == nil {
		return
	}
	global.record(level, category, action, err, meta, 0)
}

// RecordError is a convenience for recording error-level events.
func RecordError(category, action string, err error) {
	if global == nil || err == nil {
		return
	}
	global.record("error", category, action, err, nil, 0)
}

// RecordLatency records a latency measurement.
func RecordLatency(category, action string, d time.Duration) {
	if global == nil {
		return
	}
	global.record("info", category, action, nil, nil, d.Milliseconds())
}

func (c *Collector) record(level, category, action string, err error, meta map[string]string, durationMs int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.config.Enabled {
		return
	}
	if c.config.ErrorsOnly && level != "error" {
		return
	}

	ev := Event{
		Timestamp: time.Now(),
		Level:     level,
		Category:  category,
		Action:    action,
		Meta:      meta,
		Duration:  durationMs,
	}
	if err != nil {
		ev.Error = err.Error()
	}
	c.events = append(c.events, ev)
}

// Flush writes in-memory events to disk, appending to existing events.
func Flush() {
	if global == nil {
		return
	}
	global.flush()
}

func (c *Collector) flush() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.config.Enabled || len(c.events) == 0 {
		return
	}

	path := filepath.Join(c.configDir, "telemetry_events.json")

	// Read existing events from disk.
	var existing []Event
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &existing)
	}

	// Append new events.
	all := append(existing, c.events...)
	c.events = nil

	if err := os.MkdirAll(c.configDir, 0700); err != nil {
		return
	}

	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return
	}
	_ = atomicWrite(path, data, 0644)
}

// Prune removes events older than RetainDays.
func Prune() {
	if global == nil {
		return
	}
	global.prune()
}

func (c *Collector) prune() {
	c.mu.Lock()
	defer c.mu.Unlock()

	path := filepath.Join(c.configDir, "telemetry_events.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return
	}

	cutoff := time.Now().Add(-time.Duration(c.config.RetainDays) * 24 * time.Hour)
	var kept []Event
	for _, ev := range events {
		if ev.Timestamp.After(cutoff) {
			kept = append(kept, ev)
		}
	}

	out, err := json.MarshalIndent(kept, "", "  ")
	if err != nil {
		return
	}
	_ = atomicWrite(path, out, 0644)
}

// Summary returns a human-readable summary of recent events.
func Summary() string {
	if global == nil {
		return "Telemetry not initialized."
	}
	return global.summary()
}

func (c *Collector) summary() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	path := filepath.Join(c.configDir, "telemetry_events.json")
	data, err := os.ReadFile(path)
	if err != nil {
		// Also check in-memory events.
		if len(c.events) == 0 {
			return "No telemetry events recorded."
		}
		return c.summarizeEvents(c.events)
	}

	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return "Failed to read telemetry events."
	}

	// Combine disk + in-memory.
	all := append(events, c.events...)
	if len(all) == 0 {
		return "No telemetry events recorded."
	}
	return c.summarizeEvents(all)
}

func (c *Collector) summarizeEvents(events []Event) string {
	counts := map[string]int{"error": 0, "warn": 0, "info": 0}
	categories := map[string]int{}
	for _, ev := range events {
		counts[ev.Level]++
		categories[ev.Category]++
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Telemetry: %d events", len(events)))
	sb.WriteString(fmt.Sprintf(" (errors=%d, warnings=%d, info=%d)\n", counts["error"], counts["warn"], counts["info"]))
	sb.WriteString("Categories:")
	for cat, count := range categories {
		sb.WriteString(fmt.Sprintf(" %s=%d", cat, count))
	}

	// Show last 5 errors.
	var recentErrors []Event
	for i := len(events) - 1; i >= 0 && len(recentErrors) < 5; i-- {
		if events[i].Level == "error" {
			recentErrors = append(recentErrors, events[i])
		}
	}
	if len(recentErrors) > 0 {
		sb.WriteString("\nRecent errors:")
		for _, ev := range recentErrors {
			sb.WriteString(fmt.Sprintf("\n  [%s] %s/%s: %s",
				ev.Timestamp.Format("15:04:05"), ev.Category, ev.Action, ev.Error))
		}
	}

	return sb.String()
}

// SetEnabled toggles telemetry on/off and persists the config.
func SetEnabled(enabled bool) {
	if global == nil {
		return
	}
	global.mu.Lock()
	global.config.Enabled = enabled
	global.mu.Unlock()
	_ = global.saveConfig()
}

// atomicWrite writes data to path via temp file + rename, matching
// the safefile.AtomicWrite pattern used elsewhere in the codebase.
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
