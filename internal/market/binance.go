package market

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Candle represents a single OHLCV candlestick.
type Candle struct {
	OpenTime  time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	CloseTime time.Time
}

// NormalizeSymbol ensures a symbol has the USDT suffix for Binance.
// "BTC" → "BTCUSDT", "BTCUSDT" passes through.
func NormalizeSymbol(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if strings.HasSuffix(s, "USDT") || strings.HasSuffix(s, "USDC") || strings.HasSuffix(s, "USD") {
		return s
	}
	return s + "USDT"
}

// FetchKlines fetches OHLCV candles from Binance public API.
// No API key is required. interval examples: "1h", "4h", "1d".
func FetchKlines(symbol, interval string, limit int) ([]Candle, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	sym := NormalizeSymbol(symbol)
	url := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=%s&interval=%s&limit=%d", sym, interval, limit)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("binance request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("binance API returned %d", resp.StatusCode)
	}

	// Binance returns an array of arrays.
	var raw [][]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode klines: %w", err)
	}

	candles := make([]Candle, 0, len(raw))
	for _, row := range raw {
		if len(row) < 7 {
			continue
		}
		c, err := parseKlineRow(row)
		if err != nil {
			continue // skip malformed rows
		}
		candles = append(candles, c)
	}

	return candles, nil
}

func parseKlineRow(row []json.RawMessage) (Candle, error) {
	var openTimeMs, closeTimeMs int64
	if err := json.Unmarshal(row[0], &openTimeMs); err != nil {
		return Candle{}, err
	}
	if err := json.Unmarshal(row[6], &closeTimeMs); err != nil {
		return Candle{}, err
	}

	parseFloat := func(raw json.RawMessage) (float64, error) {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return 0, err
		}
		return strconv.ParseFloat(s, 64)
	}

	open, err := parseFloat(row[1])
	if err != nil {
		return Candle{}, err
	}
	high, err := parseFloat(row[2])
	if err != nil {
		return Candle{}, err
	}
	low, err := parseFloat(row[3])
	if err != nil {
		return Candle{}, err
	}
	close_, err := parseFloat(row[4])
	if err != nil {
		return Candle{}, err
	}
	vol, err := parseFloat(row[5])
	if err != nil {
		return Candle{}, err
	}

	return Candle{
		OpenTime:  time.UnixMilli(openTimeMs),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close_,
		Volume:    vol,
		CloseTime: time.UnixMilli(closeTimeMs),
	}, nil
}

// ClosePrices extracts a slice of close prices from candles.
func ClosePrices(candles []Candle) []float64 {
	prices := make([]float64, len(candles))
	for i, c := range candles {
		prices[i] = c.Close
	}
	return prices
}

// IntervalForPeriod returns an appropriate Binance interval for a given number of days.
func IntervalForPeriod(days int) string {
	switch {
	case days <= 2:
		return "1h"
	case days <= 14:
		return "4h"
	default:
		return "1d"
	}
}
