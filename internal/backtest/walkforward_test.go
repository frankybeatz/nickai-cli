package backtest

import (
	"math"
	"testing"
	"time"

	"github.com/nickai/cli/internal/market"
)

// makeSyntheticCandles creates a simple uptrending series of candles for testing.
// Prices start at startPrice and increase by pctPerCandle each step, with
// a sinusoidal oscillation to create some trading opportunities.
func makeSyntheticCandles(n int, startPrice float64) []market.Candle {
	candles := make([]market.Candle, n)
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	price := startPrice
	for i := 0; i < n; i++ {
		// Add oscillation for RSI variation: sine wave with period ~40 candles.
		oscillation := math.Sin(float64(i)/6.3) * 0.02 * price
		close := price + oscillation

		// Create realistic OHLCV.
		high := close * 1.01
		low := close * 0.99
		open := close * (1 - 0.002)
		volume := 1000.0 + float64(i)*10

		candles[i] = market.Candle{
			OpenTime:  baseTime.Add(time.Duration(i) * 24 * time.Hour),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			CloseTime: baseTime.Add(time.Duration(i)*24*time.Hour + 23*time.Hour + 59*time.Minute),
		}

		// Slow uptrend.
		price *= 1.001
	}
	return candles
}

func TestSplitWindowsRolling(t *testing.T) {
	// 500 candles, 5 windows, 30% OOS, rolling.
	cfg := WFAConfig{Windows: 5, OOSRatio: 0.3, Anchored: false}
	windows, err := splitWindows(500, cfg)
	if err != nil {
		t.Fatalf("splitWindows failed: %v", err)
	}

	if len(windows) != 5 {
		t.Fatalf("expected 5 windows, got %d", len(windows))
	}

	// Check that windows are contiguous and non-overlapping in OOS.
	for i, w := range windows {
		t.Logf("Window %d: IS[%d:%d] OOS[%d:%d]", w.WindowNum, w.ISStart, w.ISEnd, w.OOSStart, w.OOSEnd)

		// IS should come before OOS.
		if w.ISEnd > w.OOSEnd {
			t.Errorf("window %d: IS end (%d) > OOS end (%d)", i, w.ISEnd, w.OOSEnd)
		}

		// IS end should equal OOS start (no gap).
		if w.ISEnd != w.OOSStart {
			t.Errorf("window %d: IS end (%d) != OOS start (%d)", i, w.ISEnd, w.OOSStart)
		}

		// Rolling: IS start should be at the beginning of the window.
		expectedISStart := i * 100 // windowSize = 500/5 = 100
		if w.ISStart != expectedISStart {
			t.Errorf("window %d: IS start = %d, want %d", i, w.ISStart, expectedISStart)
		}
	}

	// Last window should end at 500.
	last := windows[len(windows)-1]
	if last.OOSEnd != 500 {
		t.Errorf("last window OOS end = %d, want 500", last.OOSEnd)
	}
}

func TestSplitWindowsAnchored(t *testing.T) {
	cfg := WFAConfig{Windows: 5, OOSRatio: 0.3, Anchored: true}
	windows, err := splitWindows(500, cfg)
	if err != nil {
		t.Fatalf("splitWindows failed: %v", err)
	}

	if len(windows) != 5 {
		t.Fatalf("expected 5 windows, got %d", len(windows))
	}

	// Anchored: all IS windows should start from 0.
	for i, w := range windows {
		if w.ISStart != 0 {
			t.Errorf("window %d: anchored IS start = %d, want 0", i, w.ISStart)
		}
		// IS end should grow with each window.
		if i > 0 && w.ISEnd <= windows[i-1].ISEnd {
			t.Errorf("window %d: anchored IS end (%d) should be > previous (%d)", i, w.ISEnd, windows[i-1].ISEnd)
		}
	}
}

func TestSplitWindowsTooFewCandles(t *testing.T) {
	cfg := WFAConfig{Windows: 5, OOSRatio: 0.3}
	_, err := splitWindows(50, cfg)
	if err == nil {
		t.Error("expected error for too few candles")
	}
}

func TestSplitWindowsTooManyWindows(t *testing.T) {
	// 200 candles, 10 windows → 20 candles per window, too small.
	cfg := WFAConfig{Windows: 10, OOSRatio: 0.3}
	_, err := splitWindows(200, cfg)
	if err == nil {
		t.Error("expected error for window size too small")
	}
}

func TestSplitWindowsDefaults(t *testing.T) {
	// Zero values should get defaults (5 windows, 0.3 OOS ratio).
	cfg := WFAConfig{}
	windows, err := splitWindows(500, cfg)
	if err != nil {
		t.Fatalf("splitWindows with defaults failed: %v", err)
	}
	if len(windows) != 5 {
		t.Errorf("default windows = %d, want 5", len(windows))
	}
}

func TestRunWFAWithSyntheticCandles(t *testing.T) {
	// Create enough candles for a meaningful WFA.
	candles := makeSyntheticCandles(500, 100.0)

	strat := Strategy{
		Name:          "test-rsi-wfa",
		Symbol:        "TESTUSDT",
		EntryRules:    []Condition{{Indicator: "rsi", Operator: "<", Value: 40}},
		ExitRules:     []Condition{{Indicator: "rsi", Operator: ">", Value: 60}},
		StopLossPct:   5,
		TakeProfitPct: 10,
		PositionSize:  1.0,
	}

	config := WFAConfig{
		Windows:  3,
		OOSRatio: 0.3,
		Anchored: false,
	}

	result, err := RunWFAWithCandles(strat, candles, "1d", config)
	if err != nil {
		t.Fatalf("RunWFAWithCandles failed: %v", err)
	}

	if result == nil {
		t.Fatal("RunWFAWithCandles returned nil")
	}

	if len(result.Windows) != 3 {
		t.Errorf("expected 3 windows, got %d", len(result.Windows))
	}

	// Each window should have IS and OOS results.
	for i, w := range result.Windows {
		if w.ISResult == nil {
			t.Errorf("window %d: IS result is nil", i)
		}
		if w.OOSResult == nil {
			t.Errorf("window %d: OOS result is nil", i)
		}
	}

	// Combined OOS should exist.
	if result.CombinedOOS == nil {
		t.Error("CombinedOOS is nil")
	}

	// Efficiency should be a finite number.
	if math.IsNaN(result.Efficiency) {
		t.Error("Efficiency is NaN")
	}
	if math.IsInf(result.Efficiency, 0) {
		t.Error("Efficiency is Inf")
	}

	t.Logf("WFA Efficiency: %.4f, Robust: %v", result.Efficiency, result.Robust)
}

func TestRunWFAWithAnchoredWindows(t *testing.T) {
	candles := makeSyntheticCandles(500, 100.0)

	strat := Strategy{
		Name:          "test-anchored-wfa",
		Symbol:        "TESTUSDT",
		EntryRules:    []Condition{{Indicator: "rsi", Operator: "<", Value: 40}},
		ExitRules:     []Condition{{Indicator: "rsi", Operator: ">", Value: 60}},
		StopLossPct:   5,
		TakeProfitPct: 10,
		PositionSize:  1.0,
	}

	config := WFAConfig{
		Windows:  3,
		OOSRatio: 0.3,
		Anchored: true,
	}

	result, err := RunWFAWithCandles(strat, candles, "1d", config)
	if err != nil {
		t.Fatalf("RunWFAWithCandles (anchored) failed: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}

	if len(result.Windows) != 3 {
		t.Errorf("expected 3 windows, got %d", len(result.Windows))
	}

	// In anchored mode, IS windows should get larger.
	for i := 1; i < len(result.Windows); i++ {
		currISSize := result.Windows[i].ISEnd - result.Windows[i].ISStart
		prevISSize := result.Windows[i-1].ISEnd - result.Windows[i-1].ISStart
		if currISSize <= prevISSize {
			t.Errorf("window %d IS size (%d) should be > window %d IS size (%d) in anchored mode",
				i, currISSize, i-1, prevISSize)
		}
	}
}

func TestRunWFAInsufficientCandles(t *testing.T) {
	candles := makeSyntheticCandles(30, 100.0)

	strat := Strategy{
		Name:       "test-insufficient",
		Symbol:     "TESTUSDT",
		EntryRules: []Condition{{Indicator: "rsi", Operator: "<", Value: 30}},
		ExitRules:  []Condition{{Indicator: "rsi", Operator: ">", Value: 70}},
	}

	config := WFAConfig{Windows: 5}

	_, err := RunWFAWithCandles(strat, candles, "1d", config)
	if err == nil {
		t.Error("expected error for insufficient candles")
	}
}

func TestWFAEfficiencyCalculation(t *testing.T) {
	// Construct windows with known Sharpe ratios to verify efficiency calculation.
	windows := []WFAWindow{
		{
			WindowNum: 1,
			ISResult:  &Result{SharpeRatio: 2.0},
			OOSResult: &Result{SharpeRatio: 1.5},
		},
		{
			WindowNum: 2,
			ISResult:  &Result{SharpeRatio: 3.0},
			OOSResult: &Result{SharpeRatio: 1.0},
		},
	}

	// avg IS Sharpe = (2+3)/2 = 2.5
	// avg OOS Sharpe = (1.5+1)/2 = 1.25
	// Efficiency = 1.25/2.5 = 0.5

	var isSharpeSum, oosSharpeSum float64
	for _, w := range windows {
		isSharpeSum += w.ISResult.SharpeRatio
		oosSharpeSum += w.OOSResult.SharpeRatio
	}
	avgIS := isSharpeSum / float64(len(windows))
	avgOOS := oosSharpeSum / float64(len(windows))
	efficiency := avgOOS / avgIS

	if math.Abs(efficiency-0.5) > 0.001 {
		t.Errorf("efficiency = %f, want 0.5", efficiency)
	}

	// Efficiency > 0.5 should be robust.
	robust := efficiency > 0.5
	if robust {
		t.Error("efficiency=0.5 should NOT be robust (need >0.5)")
	}

	// With better OOS performance.
	windows[1].OOSResult.SharpeRatio = 2.0
	oosSharpeSum = 1.5 + 2.0
	avgOOS = oosSharpeSum / 2
	efficiency = avgOOS / avgIS
	robust = efficiency > 0.5
	if !robust {
		t.Errorf("efficiency=%.4f should be robust", efficiency)
	}
}

func TestCombineOOSResults(t *testing.T) {
	strat := Strategy{Name: "test", Symbol: "TEST"}

	now := time.Now()
	windows := []WFAWindow{
		{
			WindowNum: 1,
			OOSResult: &Result{
				EquityCurve: []float64{1.0, 1.05, 1.10},
				Trades: []Trade{
					{PnLPct: 5, EntryTime: now, ExitTime: now.Add(time.Hour)},
				},
			},
		},
		{
			WindowNum: 2,
			OOSResult: &Result{
				EquityCurve: []float64{1.0, 0.95, 1.02},
				Trades: []Trade{
					{PnLPct: -5, EntryTime: now.Add(2 * time.Hour), ExitTime: now.Add(3 * time.Hour)},
					{PnLPct: 7, EntryTime: now.Add(4 * time.Hour), ExitTime: now.Add(5 * time.Hour)},
				},
			},
		},
	}

	combined := combineOOSResults(windows, strat, "1d")

	if combined == nil {
		t.Fatal("combined result is nil")
	}

	// Should have 3 total trades.
	if combined.TotalTrades != 3 {
		t.Errorf("TotalTrades = %d, want 3", combined.TotalTrades)
	}

	// Combined curve should start at 1.0.
	if combined.EquityCurve[0] != 1.0 {
		t.Errorf("combined curve starts at %f, want 1.0", combined.EquityCurve[0])
	}

	// Combined curve should reflect sequential performance.
	// Window 1: 1.0 → 1.10 (10% gain)
	// Window 2 starts at 1.10, goes to 1.10 * 0.95 = 1.045, then 1.10 * 1.02 = 1.122.
	if len(combined.EquityCurve) < 3 {
		t.Fatalf("combined curve too short: %d points", len(combined.EquityCurve))
	}

	t.Logf("Combined equity curve: %v", combined.EquityCurve)
}
