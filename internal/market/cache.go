package market

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// cacheDir returns the path to the candle cache directory, creating it if needed.
func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".nickai", "cache", "candles")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// cacheKey returns a unique filename for a cache entry.
// Format: SYMBOL_INTERVAL_TOTAL.json
func cacheKey(symbol, interval string, total int) string {
	return fmt.Sprintf("%s_%s_%d.json", symbol, interval, total)
}

// cacheEntry wraps cached candles with a timestamp.
type cacheEntry struct {
	CachedAt time.Time `json:"cached_at"`
	Symbol   string    `json:"symbol"`
	Interval string    `json:"interval"`
	Total    int       `json:"total"`
	Candles  []Candle  `json:"candles"`
}

// loadCache attempts to load candles from the file cache.
// Returns nil if the cache is missing, stale (>1h for intraday, >6h for daily), or corrupt.
func loadCache(symbol, interval string, total int) []Candle {
	dir, err := cacheDir()
	if err != nil {
		return nil
	}

	path := filepath.Join(dir, cacheKey(symbol, interval, total))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil
	}

	// Staleness check: intraday intervals expire faster.
	maxAge := 6 * time.Hour // daily
	switch interval {
	case "1h":
		maxAge = 1 * time.Hour
	case "4h":
		maxAge = 2 * time.Hour
	}

	if time.Since(entry.CachedAt) > maxAge {
		return nil
	}

	// Verify data matches request.
	if entry.Symbol != symbol || entry.Interval != interval {
		return nil
	}

	return entry.Candles
}

// saveCache writes candles to the file cache.
func saveCache(symbol, interval string, total int, candles []Candle) {
	dir, err := cacheDir()
	if err != nil {
		return
	}

	entry := cacheEntry{
		CachedAt: time.Now(),
		Symbol:   symbol,
		Interval: interval,
		Total:    total,
		Candles:  candles,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	path := filepath.Join(dir, cacheKey(symbol, interval, total))
	_ = os.WriteFile(path, data, 0644)
}
