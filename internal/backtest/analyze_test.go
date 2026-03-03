package backtest

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestFormatResultForAnalysis_AllSections(t *testing.T) {
	now := time.Now()
	result := &Result{
		Strategy:     "rsi-reversal",
		Symbol:       "BTC",
		Period:       "180d",
		TotalTrades:  3,
		WinRate:      66.67,
		TotalReturn:  20.0,
		SharpeRatio:  1.85,
		MaxDrawdown:  4.55,
		ProfitFactor: 5.33,
		BestTrade:    14.29,
		WorstTrade:   -4.55,
		EquityCurve:  []float64{1.0, 1.10, 1.05, 1.20},
		Trades: []Trade{
			{EntryPrice: 100, ExitPrice: 110, PnLPct: 10, EntryTime: now, ExitTime: now.Add(24 * time.Hour), Reason: "exit_signal"},
			{EntryPrice: 110, ExitPrice: 105, PnLPct: -4.55, EntryTime: now.Add(48 * time.Hour), ExitTime: now.Add(72 * time.Hour), Reason: "stop_loss"},
			{EntryPrice: 105, ExitPrice: 120, PnLPct: 14.29, EntryTime: now.Add(96 * time.Hour), ExitTime: now.Add(120 * time.Hour), Reason: "take_profit"},
		},
	}

	output := FormatResultForAnalysis(result)

	if output == "" {
		t.Fatal("FormatResultForAnalysis returned empty string")
	}

	// Check all major sections are present.
	requiredSections := []string{
		"Strategy Overview",
		"Performance Metrics",
		"Trade Statistics",
		"Exit Reason Breakdown",
		"Equity Curve Summary",
		"Recent Trades",
		"Questions for Analysis",
	}
	for _, section := range requiredSections {
		if !strings.Contains(output, section) {
			t.Errorf("output missing section: %q", section)
		}
	}

	// Check key data is present.
	requiredData := []string{
		"rsi-reversal",
		"BTC",
		"180d",
		"20.00%",
		"1.85",
		"4.55%",
		"5.33",
		"66.7%",
		"exit_signal",
		"stop_loss",
		"take_profit",
	}
	for _, data := range requiredData {
		if !strings.Contains(output, data) {
			t.Errorf("output missing data: %q", data)
		}
	}

	// Check analysis questions reference actual values.
	if !strings.Contains(output, "3 trades") {
		t.Error("analysis questions should reference trade count")
	}
	if !strings.Contains(output, "Statistical Significance") {
		t.Error("missing statistical significance question")
	}
	if !strings.Contains(output, "Overfitting Risk") {
		t.Error("missing overfitting risk question")
	}
	if !strings.Contains(output, "Benchmark Comparison") {
		t.Error("missing benchmark comparison question")
	}
	if !strings.Contains(output, "Risk Assessment") {
		t.Error("missing risk assessment question")
	}
	if !strings.Contains(output, "Improvement Suggestions") {
		t.Error("missing improvement suggestions question")
	}
	if !strings.Contains(output, "Regime Risk") {
		t.Error("missing regime risk question")
	}
}

func TestFormatResultForAnalysis_ZeroTrades(t *testing.T) {
	result := &Result{
		Strategy:    "momentum",
		Symbol:      "ETH",
		Period:      "90d",
		TotalTrades: 0,
		WinRate:     0,
		TotalReturn: 0,
		EquityCurve: []float64{1.0},
	}

	output := FormatResultForAnalysis(result)

	if output == "" {
		t.Fatal("FormatResultForAnalysis returned empty for zero-trade result")
	}

	// Should still have strategy overview and metrics.
	if !strings.Contains(output, "ETH") {
		t.Error("missing symbol in zero-trade output")
	}
	if !strings.Contains(output, "Total Trades: 0") {
		t.Error("missing trade count in zero-trade output")
	}

	// Should NOT have "Recent Trades" section since there are none.
	if strings.Contains(output, "Recent Trades") {
		t.Error("zero-trade result should not have Recent Trades section")
	}

	// Should NOT have "Exit Reason Breakdown" since no trades.
	if strings.Contains(output, "Exit Reason Breakdown") {
		t.Error("zero-trade result should not have Exit Reason Breakdown section")
	}
}

func TestFormatResultForAnalysis_NilResult(t *testing.T) {
	output := FormatResultForAnalysis(nil)

	if output == "" {
		t.Fatal("FormatResultForAnalysis returned empty for nil result")
	}
	if !strings.Contains(output, "No backtest result") {
		t.Error("nil result should return descriptive message")
	}
}

func TestFormatResultForAnalysis_EmptyEquityCurve(t *testing.T) {
	result := &Result{
		Strategy:    "test",
		Symbol:      "SOL",
		Period:      "30d",
		TotalTrades: 0,
		EquityCurve: nil,
	}

	output := FormatResultForAnalysis(result)

	if output == "" {
		t.Fatal("FormatResultForAnalysis returned empty for nil equity curve")
	}

	// Should NOT have equity curve section with nil curve.
	if strings.Contains(output, "Equity Curve Summary") {
		t.Error("nil equity curve should not produce Equity Curve Summary section")
	}
}

func TestFormatResultForAnalysis_ManyTrades(t *testing.T) {
	// Verify that only the last 10 trades are included.
	now := time.Now()
	trades := make([]Trade, 25)
	for i := range trades {
		trades[i] = Trade{
			EntryPrice: 100 + float64(i),
			ExitPrice:  105 + float64(i),
			PnLPct:     5.0,
			EntryTime:  now.Add(time.Duration(i*24) * time.Hour),
			ExitTime:   now.Add(time.Duration((i+1)*24) * time.Hour),
			Reason:     "exit_signal",
		}
	}

	result := &Result{
		Strategy:     "test",
		Symbol:       "BTC",
		Period:       "365d",
		TotalTrades:  25,
		WinRate:      100,
		TotalReturn:  125,
		SharpeRatio:  3.0,
		MaxDrawdown:  2.0,
		ProfitFactor: 999,
		BestTrade:    5.0,
		WorstTrade:   5.0,
		Trades:       trades,
		EquityCurve:  []float64{1.0, 2.25},
	}

	output := FormatResultForAnalysis(result)

	// Count trade lines (lines starting with "  #").
	lines := strings.Split(output, "\n")
	tradeLines := 0
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") && strings.Contains(line, "Entry $") {
			tradeLines++
		}
	}
	if tradeLines != 10 {
		t.Errorf("expected 10 trade lines, got %d", tradeLines)
	}

	// First trade shown should be #16 (trades[15]).
	if !strings.Contains(output, "#16:") {
		t.Error("first displayed trade should be #16")
	}
	// Last trade shown should be #25.
	if !strings.Contains(output, "#25:") {
		t.Error("last displayed trade should be #25")
	}
}

func TestFormatResultForAnalysis_InfProfitFactor(t *testing.T) {
	result := &Result{
		Strategy:     "all-winners",
		Symbol:       "BTC",
		Period:       "30d",
		TotalTrades:  2,
		WinRate:      100,
		TotalReturn:  10,
		ProfitFactor: positiveInf(),
		Trades: []Trade{
			{EntryPrice: 100, ExitPrice: 105, PnLPct: 5, EntryTime: time.Now(), ExitTime: time.Now(), Reason: "take_profit"},
			{EntryPrice: 105, ExitPrice: 110, PnLPct: 4.76, EntryTime: time.Now(), ExitTime: time.Now(), Reason: "take_profit"},
		},
		EquityCurve: []float64{1.0, 1.05, 1.10},
	}

	output := FormatResultForAnalysis(result)

	if !strings.Contains(output, "Inf") {
		t.Error("infinite profit factor should be formatted as Inf")
	}
}

func TestAnalysisSystemPrompt(t *testing.T) {
	if AnalysisSystemPrompt == "" {
		t.Fatal("AnalysisSystemPrompt should not be empty")
	}
	if !strings.Contains(AnalysisSystemPrompt, "quantitative") {
		t.Error("system prompt should mention quantitative analysis")
	}
	if !strings.Contains(AnalysisSystemPrompt, "overfitting") {
		t.Error("system prompt should mention overfitting")
	}
}

// positiveInf returns positive infinity for testing.
func positiveInf() float64 {
	return math.Inf(1)
}
