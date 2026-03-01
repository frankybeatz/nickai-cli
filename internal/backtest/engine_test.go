package backtest

import (
	"math"
	"testing"
	"time"
)

func TestParsePeriod(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		wantErr  bool
	}{
		{"180d", 180, false},
		{"30d", 30, false},
		{"6m", 180, false},
		{"1y", 365, false},
		{"90", 90, false},
		{"", 180, false},     // default
		{"abc", 0, true},
		{"10x", 0, true},
	}
	for _, tt := range tests {
		got, err := ParsePeriod(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParsePeriod(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.expected {
			t.Errorf("ParsePeriod(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestEvalCondition(t *testing.T) {
	snapshots := []indicatorSnapshot{
		{RSI: 25, Price: 100, MACD: -0.5, MACDHistogram: -1.0},
		{RSI: 35, Price: 105, MACD: 0.5, MACDHistogram: 0.5},
		{RSI: 72, Price: 110, MACD: 1.0, MACDHistogram: 1.5},
	}

	tests := []struct {
		cond     Condition
		idx      int
		expected bool
	}{
		// RSI < 30 at index 0 (RSI = 25)
		{Condition{Indicator: "rsi", Operator: "<", Value: 30}, 0, true},
		// RSI < 30 at index 1 (RSI = 35)
		{Condition{Indicator: "rsi", Operator: "<", Value: 30}, 1, false},
		// RSI > 70 at index 2 (RSI = 72)
		{Condition{Indicator: "rsi", Operator: ">", Value: 70}, 2, true},
		// MACD histogram crosses above 0 at index 1 (prev=-1, curr=0.5)
		{Condition{Indicator: "macd_histogram", Operator: "crosses_above", Value: 0}, 1, true},
		// MACD histogram crosses above 0 at index 0 (no prev)
		{Condition{Indicator: "macd_histogram", Operator: "crosses_above", Value: 0}, 0, false},
		// Price > 108 at index 2 (price=110)
		{Condition{Indicator: "price", Operator: ">", Value: 108}, 2, true},
	}

	for i, tt := range tests {
		got := evalCondition(tt.cond, snapshots, tt.idx)
		if got != tt.expected {
			t.Errorf("test %d: evalCondition(%+v, idx=%d) = %v, want %v", i, tt.cond, tt.idx, got, tt.expected)
		}
	}
}

func TestGetIndicatorValue(t *testing.T) {
	snap := indicatorSnapshot{
		RSI:            45.5,
		MACD:           1.23,
		MACDSignal:     0.98,
		MACDHistogram:  0.25,
		BollingerUpper: 110.0,
		BollingerLower: 90.0,
		SMA20:          100.0,
		SMA50:          98.0,
		EMA12:          101.0,
		EMA26:          99.0,
		Price:          105.0,
		FearGreed:      30.0,
	}

	tests := []struct {
		indicator string
		expected  float64
	}{
		{"rsi", 45.5},
		{"macd", 1.23},
		{"macd_signal", 0.98},
		{"macd_histogram", 0.25},
		{"bollinger_upper", 110.0},
		{"bollinger_lower", 90.0},
		{"sma20", 100.0},
		{"sma50", 98.0},
		{"ema12", 101.0},
		{"ema26", 99.0},
		{"price", 105.0},
		{"fear_greed", 30.0},
		{"unknown", 0.0},
	}

	for _, tt := range tests {
		got := getIndicatorValue(tt.indicator, snap)
		if got != tt.expected {
			t.Errorf("getIndicatorValue(%q) = %f, want %f", tt.indicator, got, tt.expected)
		}
	}
}

func TestComputeMetrics(t *testing.T) {
	now := time.Now()
	trades := []Trade{
		{EntryPrice: 100, ExitPrice: 110, PnLPct: 10, EntryTime: now, ExitTime: now.Add(24 * time.Hour), Reason: "exit_signal"},
		{EntryPrice: 110, ExitPrice: 105, PnLPct: -4.55, EntryTime: now.Add(48 * time.Hour), ExitTime: now.Add(72 * time.Hour), Reason: "stop_loss"},
		{EntryPrice: 105, ExitPrice: 120, PnLPct: 14.29, EntryTime: now.Add(96 * time.Hour), ExitTime: now.Add(120 * time.Hour), Reason: "take_profit"},
	}

	curve := []float64{1.0, 1.10, 1.05, 1.20}

	strat := Strategy{Name: "test", Symbol: "BTC", Period: "30d"}
	result := computeMetrics(strat, trades, curve)

	if result.TotalTrades != 3 {
		t.Errorf("TotalTrades = %d, want 3", result.TotalTrades)
	}

	// 2 wins out of 3.
	expectedWinRate := 2.0 / 3.0 * 100
	if math.Abs(result.WinRate-expectedWinRate) > 0.1 {
		t.Errorf("WinRate = %.2f, want ~%.2f", result.WinRate, expectedWinRate)
	}

	if result.BestTrade != 14.29 {
		t.Errorf("BestTrade = %f, want 14.29", result.BestTrade)
	}
	if result.WorstTrade != -4.55 {
		t.Errorf("WorstTrade = %f, want -4.55", result.WorstTrade)
	}

	// Total return from equity curve.
	expectedReturn := (1.20 - 1.0) / 1.0 * 100
	if math.Abs(result.TotalReturn-expectedReturn) > 0.01 {
		t.Errorf("TotalReturn = %.2f, want %.2f", result.TotalReturn, expectedReturn)
	}

	// Max drawdown: peak was 1.10, trough was 1.05 => dd = (1.10-1.05)/1.10 * 100.
	expectedDD := (1.10 - 1.05) / 1.10 * 100
	if math.Abs(result.MaxDrawdown-expectedDD) > 0.01 {
		t.Errorf("MaxDrawdown = %.2f, want ~%.2f", result.MaxDrawdown, expectedDD)
	}

	// Profit factor: totalProfit / totalLoss = (10+14.29) / 4.55.
	expectedPF := (10.0 + 14.29) / 4.55
	if math.Abs(result.ProfitFactor-expectedPF) > 0.01 {
		t.Errorf("ProfitFactor = %.2f, want ~%.2f", result.ProfitFactor, expectedPF)
	}
}

func TestComputeMetricsNoTrades(t *testing.T) {
	strat := Strategy{Name: "empty", Symbol: "ETH", Period: "30d"}
	result := computeMetrics(strat, nil, []float64{1.0})

	if result.TotalTrades != 0 {
		t.Errorf("TotalTrades = %d, want 0", result.TotalTrades)
	}
	if result.WinRate != 0 {
		t.Errorf("WinRate = %f, want 0", result.WinRate)
	}
}

func TestMaxDrawdown(t *testing.T) {
	tests := []struct {
		curve    []float64
		expected float64
	}{
		{[]float64{1.0, 1.1, 1.05, 1.2}, (1.1 - 1.05) / 1.1 * 100},
		{[]float64{1.0, 0.9, 0.8, 0.7}, (1.0 - 0.7) / 1.0 * 100}, // 30%
		{[]float64{1.0, 1.1, 1.2, 1.3}, 0},                        // no drawdown
		{nil, 0},
	}
	for i, tt := range tests {
		got := maxDrawdown(tt.curve)
		if math.Abs(got-tt.expected) > 0.01 {
			t.Errorf("test %d: maxDrawdown = %f, want %f", i, got, tt.expected)
		}
	}
}

func TestRunIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	strat := Strategy{
		Name:   "test-rsi",
		Symbol: "BTC",
		EntryRules: []Condition{
			{Indicator: "rsi", Operator: "<", Value: 30},
		},
		ExitRules: []Condition{
			{Indicator: "rsi", Operator: ">", Value: 70},
		},
		StopLossPct:   5,
		TakeProfitPct: 15,
		PositionSize:  1.0,
		Period:        "90d",
	}

	result, err := Run(strat)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result == nil {
		t.Fatal("Run returned nil result")
	}
	if result.Symbol != "BTC" {
		t.Errorf("Symbol = %q, want BTC", result.Symbol)
	}
	if len(result.EquityCurve) == 0 {
		t.Error("EquityCurve is empty")
	}
	// Equity curve should start at 1.0.
	if result.EquityCurve[0] != 1.0 {
		t.Errorf("EquityCurve[0] = %f, want 1.0", result.EquityCurve[0])
	}
}
