package logging

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestInitNonDebug(t *testing.T) {
	// When debug=false, the logger should suppress everything up to and including Error.
	Init(false)

	// Capture slog output to a buffer via a custom handler.
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	testLogger := slog.New(handler)

	// The default logger should NOT output debug/info/warn/error since its
	// level is set above Error. We verify by checking that slog.Default()
	// handler reports those levels as disabled.
	defaultHandler := slog.Default().Handler()

	if defaultHandler.Enabled(nil, slog.LevelDebug) {
		t.Error("debug=false: LevelDebug should be disabled")
	}
	if defaultHandler.Enabled(nil, slog.LevelInfo) {
		t.Error("debug=false: LevelInfo should be disabled")
	}
	if defaultHandler.Enabled(nil, slog.LevelWarn) {
		t.Error("debug=false: LevelWarn should be disabled")
	}
	if defaultHandler.Enabled(nil, slog.LevelError) {
		t.Error("debug=false: LevelError should be disabled")
	}

	// Sanity: the test logger we created separately should still work.
	testLogger.Info("test message")
	if buf.Len() == 0 {
		t.Error("test logger should produce output")
	}
}

func TestInitDebugCreatesLogFile(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	Init(true)

	logPath := filepath.Join(dir, ".nickai", "debug.log")
	_, err := os.Stat(logPath)
	if err != nil {
		t.Errorf("debug=true: expected log file at %s, got error: %v", logPath, err)
	}

	// The default logger should enable debug level.
	defaultHandler := slog.Default().Handler()
	if !defaultHandler.Enabled(nil, slog.LevelDebug) {
		t.Error("debug=true: LevelDebug should be enabled")
	}
}

func TestInitDebugAllLevelsEnabled(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	Init(true)

	defaultHandler := slog.Default().Handler()
	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		if !defaultHandler.Enabled(nil, level) {
			t.Errorf("debug=true: %v should be enabled", level)
		}
	}
}

func TestLogFunctionsDontPanic(t *testing.T) {
	// Initialize with debug=false (discard mode), then call all log functions.
	// They should not panic.
	Init(false)

	Debug("debug message", "key", "value")
	Info("info message", "count", 42)
	Warn("warn message", "missing", true)
	Error("error message", "code", 500)
}
