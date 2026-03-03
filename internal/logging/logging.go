package logging

import (
	"log/slog"
	"os"
	"path/filepath"
)

var logger *slog.Logger

// Init sets up structured logging. When debug is true, logs go to
// ~/.nickai/debug.log. Otherwise logging is a no-op (discards output).
func Init(debug bool) {
	if !debug {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1}))
		slog.SetDefault(logger)
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to stderr if we can't find home dir.
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
		slog.SetDefault(logger)
		return
	}

	dir := filepath.Join(home, ".nickai")
	os.MkdirAll(dir, 0700)

	logPath := filepath.Join(dir, "debug.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
		slog.SetDefault(logger)
		return
	}

	logger = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Info logs an info message.
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}
