package backtest

import (
	"math"
	"testing"
)

func TestCartesianProductSingle(t *testing.T) {
	params := []ParamRange{
		{Name: "stop_loss_pct", Values: []float64{3, 5, 8}},
	}

	combos := cartesianProduct(params)
	if len(combos) != 3 {
		t.Fatalf("expected 3 combos, got %d", len(combos))
	}

	// Check values.
	expected := []float64{3, 5, 8}
	for i, combo := range combos {
		if combo["stop_loss_pct"] != expected[i] {
			t.Errorf("combo %d: stop_loss_pct = %f, want %f", i, combo["stop_loss_pct"], expected[i])
		}
	}
}

func TestCartesianProductMultiple(t *testing.T) {
	params := []ParamRange{
		{Name: "stop_loss_pct", Values: []float64{3, 5}},
		{Name: "take_profit_pct", Values: []float64{10, 15, 20}},
	}

	combos := cartesianProduct(params)
	// 2 * 3 = 6 combinations.
	if len(combos) != 6 {
		t.Fatalf("expected 6 combos, got %d", len(combos))
	}

	// Verify all combinations exist.
	type pair struct{ sl, tp float64 }
	seen := make(map[pair]bool)
	for _, combo := range combos {
		seen[pair{combo["stop_loss_pct"], combo["take_profit_pct"]}] = true
	}
	expectedPairs := []pair{
		{3, 10}, {3, 15}, {3, 20},
		{5, 10}, {5, 15}, {5, 20},
	}
	for _, p := range expectedPairs {
		if !seen[p] {
			t.Errorf("missing combination: stop_loss=%f, take_profit=%f", p.sl, p.tp)
		}
	}
}

func TestCartesianProductThreeParams(t *testing.T) {
	params := []ParamRange{
		{Name: "a", Values: []float64{1, 2}},
		{Name: "b", Values: []float64{10, 20}},
		{Name: "c", Values: []float64{100, 200, 300}},
	}

	combos := cartesianProduct(params)
	// 2 * 2 * 3 = 12 combinations.
	if len(combos) != 12 {
		t.Fatalf("expected 12 combos, got %d", len(combos))
	}
}

func TestCartesianProductEmpty(t *testing.T) {
	// No params.
	combos := cartesianProduct(nil)
	if len(combos) != 0 {
		t.Errorf("expected 0 combos for nil params, got %d", len(combos))
	}

	// Empty values.
	params := []ParamRange{
		{Name: "a", Values: nil},
	}
	combos = cartesianProduct(params)
	if len(combos) != 0 {
		t.Errorf("expected 0 combos for empty values, got %d", len(combos))
	}
}

func TestApplyParamsStopLossAndTakeProfit(t *testing.T) {
	strat := Strategy{
		Name:          "test",
		StopLossPct:   5,
		TakeProfitPct: 15,
	}

	params := map[string]float64{
		"stop_loss_pct":   3,
		"take_profit_pct": 20,
	}

	modified := applyParams(strat, params)

	if modified.StopLossPct != 3 {
		t.Errorf("StopLossPct = %f, want 3", modified.StopLossPct)
	}
	if modified.TakeProfitPct != 20 {
		t.Errorf("TakeProfitPct = %f, want 20", modified.TakeProfitPct)
	}

	// Original should not be modified.
	if strat.StopLossPct != 5 {
		t.Errorf("original StopLossPct was modified: %f", strat.StopLossPct)
	}
}

func TestApplyParamsRSI(t *testing.T) {
	strat := Strategy{
		Name: "test",
		EntryRules: []Condition{
			{Indicator: "rsi", Operator: "<", Value: 30},
			{Indicator: "macd_histogram", Operator: ">", Value: 0},
		},
		ExitRules: []Condition{
			{Indicator: "rsi", Operator: ">", Value: 70},
		},
	}

	params := map[string]float64{
		"rsi_entry": 25,
		"rsi_exit":  75,
	}

	modified := applyParams(strat, params)

	// Entry RSI should be updated.
	if modified.EntryRules[0].Value != 25 {
		t.Errorf("entry RSI value = %f, want 25", modified.EntryRules[0].Value)
	}
	// MACD entry should NOT be changed.
	if modified.EntryRules[1].Value != 0 {
		t.Errorf("entry MACD value was changed to %f, should be 0", modified.EntryRules[1].Value)
	}
	// Exit RSI should be updated.
	if modified.ExitRules[0].Value != 75 {
		t.Errorf("exit RSI value = %f, want 75", modified.ExitRules[0].Value)
	}

	// Original should not be modified.
	if strat.EntryRules[0].Value != 30 {
		t.Errorf("original entry RSI was modified: %f", strat.EntryRules[0].Value)
	}
}

func TestApplyParamsDoesNotMutateOriginal(t *testing.T) {
	strat := Strategy{
		Name:       "original",
		EntryRules: []Condition{{Indicator: "rsi", Operator: "<", Value: 30}},
		ExitRules:  []Condition{{Indicator: "rsi", Operator: ">", Value: 70}},
	}

	params := map[string]float64{"rsi_entry": 20, "rsi_exit": 80}
	applyParams(strat, params)

	// Original should be untouched.
	if strat.EntryRules[0].Value != 30 {
		t.Errorf("original entry rule was mutated: %f", strat.EntryRules[0].Value)
	}
	if strat.ExitRules[0].Value != 70 {
		t.Errorf("original exit rule was mutated: %f", strat.ExitRules[0].Value)
	}
}

func TestExtractMetric(t *testing.T) {
	result := &Result{
		SharpeRatio:  1.5,
		SortinoRatio: 2.0,
		TotalReturn:  25.0,
		CalmarRatio:  3.0,
		ProfitFactor: 1.8,
		WinRate:      65.0,
	}

	tests := []struct {
		metric   string
		expected float64
	}{
		{"sharpe", 1.5},
		{"sortino", 2.0},
		{"total_return", 25.0},
		{"calmar", 3.0},
		{"profit_factor", 1.8},
		{"win_rate", 65.0},
		{"unknown", 1.5}, // defaults to sharpe
	}

	for _, tt := range tests {
		got := extractMetric(result, tt.metric)
		if math.Abs(got-tt.expected) > 0.001 {
			t.Errorf("extractMetric(%q) = %f, want %f", tt.metric, got, tt.expected)
		}
	}
}

func TestRunGridSearchWithSyntheticData(t *testing.T) {
	candles := makeSyntheticCandles(200, 100.0)

	baseStrat := Strategy{
		Name:          "test-grid",
		Symbol:        "TESTUSDT",
		EntryRules:    []Condition{{Indicator: "rsi", Operator: "<", Value: 30}},
		ExitRules:     []Condition{{Indicator: "rsi", Operator: ">", Value: 70}},
		StopLossPct:   5,
		TakeProfitPct: 15,
		PositionSize:  1.0,
	}

	config := OptimizeConfig{
		Params: []ParamRange{
			{Name: "stop_loss_pct", Values: []float64{3, 5, 8}},
			{Name: "take_profit_pct", Values: []float64{10, 15}},
		},
		Metric:    "sharpe",
		MaxCombos: 100,
	}

	result, err := RunGridSearchWithCandles(baseStrat, candles, "1d", config)
	if err != nil {
		t.Fatalf("RunGridSearchWithCandles failed: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}

	// 3 * 2 = 6 combinations.
	if result.TotalCombos != 6 {
		t.Errorf("TotalCombos = %d, want 6", result.TotalCombos)
	}

	// Best params should be populated.
	if len(result.BestParams) == 0 {
		t.Error("BestParams is empty")
	}

	// Duration should be positive.
	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}

	// TopN should have entries.
	if len(result.TopN) == 0 {
		t.Error("TopN is empty")
	}

	// TopN should be sorted descending by metric.
	for i := 1; i < len(result.TopN); i++ {
		if result.TopN[i].Metric > result.TopN[i-1].Metric {
			t.Errorf("TopN not sorted: entry %d (%.4f) > entry %d (%.4f)",
				i, result.TopN[i].Metric, i-1, result.TopN[i-1].Metric)
		}
	}

	t.Logf("Best params: %v, metric: %.4f, duration: %v", result.BestParams, result.BestMetric, result.Duration)
}

func TestRunGridSearchMaxCombosLimit(t *testing.T) {
	candles := makeSyntheticCandles(200, 100.0)

	baseStrat := Strategy{
		Name:       "test-limit",
		Symbol:     "TESTUSDT",
		EntryRules: []Condition{{Indicator: "rsi", Operator: "<", Value: 30}},
		ExitRules:  []Condition{{Indicator: "rsi", Operator: ">", Value: 70}},
	}

	config := OptimizeConfig{
		Params: []ParamRange{
			{Name: "stop_loss_pct", Values: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
			{Name: "take_profit_pct", Values: []float64{5, 10, 15, 20, 25, 30}},
		},
		Metric:    "sharpe",
		MaxCombos: 10, // 10 * 6 = 60 combos, exceeds limit of 10
	}

	_, err := RunGridSearchWithCandles(baseStrat, candles, "1d", config)
	if err == nil {
		t.Error("expected error when exceeding MaxCombos limit")
	}
}

func TestRunGridSearchNoParams(t *testing.T) {
	candles := makeSyntheticCandles(200, 100.0)

	baseStrat := Strategy{
		Name:       "test-no-params",
		Symbol:     "TESTUSDT",
		EntryRules: []Condition{{Indicator: "rsi", Operator: "<", Value: 30}},
		ExitRules:  []Condition{{Indicator: "rsi", Operator: ">", Value: 70}},
	}

	config := OptimizeConfig{
		Params: nil,
		Metric: "sharpe",
	}

	_, err := RunGridSearchWithCandles(baseStrat, candles, "1d", config)
	if err == nil {
		t.Error("expected error for no parameters")
	}
}

func TestRunGridSearchWithRSIParams(t *testing.T) {
	candles := makeSyntheticCandles(200, 100.0)

	baseStrat := Strategy{
		Name:          "test-rsi-grid",
		Symbol:        "TESTUSDT",
		EntryRules:    []Condition{{Indicator: "rsi", Operator: "<", Value: 30}},
		ExitRules:     []Condition{{Indicator: "rsi", Operator: ">", Value: 70}},
		StopLossPct:   5,
		TakeProfitPct: 15,
		PositionSize:  1.0,
	}

	config := OptimizeConfig{
		Params: []ParamRange{
			{Name: "rsi_entry", Values: []float64{25, 30, 35}},
			{Name: "rsi_exit", Values: []float64{65, 70, 75}},
		},
		Metric:    "total_return",
		MaxCombos: 100,
	}

	result, err := RunGridSearchWithCandles(baseStrat, candles, "1d", config)
	if err != nil {
		t.Fatalf("RunGridSearchWithCandles failed: %v", err)
	}

	// 3 * 3 = 9 combinations.
	if result.TotalCombos != 9 {
		t.Errorf("TotalCombos = %d, want 9", result.TotalCombos)
	}

	// Verify best params contain both RSI entry and exit.
	if _, ok := result.BestParams["rsi_entry"]; !ok {
		t.Error("BestParams missing rsi_entry")
	}
	if _, ok := result.BestParams["rsi_exit"]; !ok {
		t.Error("BestParams missing rsi_exit")
	}

	t.Logf("Best RSI params: entry=%.0f, exit=%.0f, return=%.2f%%",
		result.BestParams["rsi_entry"], result.BestParams["rsi_exit"], result.BestMetric)
}

func TestOptimizeDefaults(t *testing.T) {
	cfg := OptimizeConfig{}
	cfg = optimizeDefaults(cfg)

	if cfg.MaxCombos != 1000 {
		t.Errorf("default MaxCombos = %d, want 1000", cfg.MaxCombos)
	}
	if cfg.Metric != "sharpe" {
		t.Errorf("default Metric = %q, want %q", cfg.Metric, "sharpe")
	}
}
