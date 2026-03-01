package market

import (
	"testing"
)

func TestNormalizeSymbol(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"BTC", "BTCUSDT"},
		{"btc", "BTCUSDT"},
		{"BTCUSDT", "BTCUSDT"},
		{"ETHUSDC", "ETHUSDC"},
		{"SOLUSD", "SOLUSD"},
		{"  eth  ", "ETHUSDT"},
	}
	for _, tt := range tests {
		got := NormalizeSymbol(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeSymbol(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestIntervalForPeriod(t *testing.T) {
	tests := []struct {
		days     int
		expected string
	}{
		{1, "1h"},
		{2, "1h"},
		{3, "4h"},
		{14, "4h"},
		{15, "1d"},
		{30, "1d"},
		{365, "1d"},
	}
	for _, tt := range tests {
		got := IntervalForPeriod(tt.days)
		if got != tt.expected {
			t.Errorf("IntervalForPeriod(%d) = %q, want %q", tt.days, got, tt.expected)
		}
	}
}

func TestClosePrices(t *testing.T) {
	candles := []Candle{
		{Close: 100.0},
		{Close: 101.5},
		{Close: 99.8},
	}
	prices := ClosePrices(candles)
	if len(prices) != 3 {
		t.Fatalf("ClosePrices returned %d prices, want 3", len(prices))
	}
	if prices[0] != 100.0 {
		t.Errorf("prices[0] = %f, want 100.0", prices[0])
	}
	if prices[1] != 101.5 {
		t.Errorf("prices[1] = %f, want 101.5", prices[1])
	}
	if prices[2] != 99.8 {
		t.Errorf("prices[2] = %f, want 99.8", prices[2])
	}
}

func TestClosePricesEmpty(t *testing.T) {
	prices := ClosePrices(nil)
	if len(prices) != 0 {
		t.Errorf("ClosePrices(nil) returned %d prices, want 0", len(prices))
	}
}

func TestFetchKlinesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	t.Run("BTCUSDT daily", func(t *testing.T) {
		candles, err := FetchKlines("BTC", "1d", 10)
		if err != nil {
			t.Fatalf("FetchKlines failed: %v", err)
		}
		if len(candles) == 0 {
			t.Fatal("FetchKlines returned no candles")
		}
		for _, c := range candles {
			if c.Close <= 0 {
				t.Errorf("unexpected close price: %f", c.Close)
			}
			if c.Volume < 0 {
				t.Errorf("unexpected volume: %f", c.Volume)
			}
			if c.OpenTime.IsZero() {
				t.Error("OpenTime is zero")
			}
		}
	})
}
