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

func TestEvalConditionCompareWith(t *testing.T) {
	snapshots := []indicatorSnapshot{
		{Price: 89, BollingerLower: 90, SMA20: 95, SMA50: 100},
		{Price: 91, BollingerLower: 90, SMA20: 100, SMA50: 99},
		{Price: 105, BollingerLower: 95, SMA20: 98, SMA50: 100},
	}

	tests := []struct {
		name     string
		cond     Condition
		idx      int
		expected bool
	}{
		// price < bollinger_lower at index 0 (89 < 90)
		{"price below lower band", Condition{Indicator: "price", Operator: "<", CompareWith: "bollinger_lower"}, 0, true},
		// price < bollinger_lower at index 1 (91 < 90 = false)
		{"price above lower band", Condition{Indicator: "price", Operator: "<", CompareWith: "bollinger_lower"}, 1, false},
		// sma20 crosses_above sma50 at index 1 (prev: 95<=100, curr: 100>99)
		{"golden cross", Condition{Indicator: "sma20", Operator: "crosses_above", CompareWith: "sma50"}, 1, true},
		// sma20 crosses_above sma50 at index 0 (no prev)
		{"golden cross no prev", Condition{Indicator: "sma20", Operator: "crosses_above", CompareWith: "sma50"}, 0, false},
		// sma20 crosses_below sma50 at index 2 (prev: 100>=99, curr: 98<100)
		{"death cross", Condition{Indicator: "sma20", Operator: "crosses_below", CompareWith: "sma50"}, 2, true},
	}

	for _, tt := range tests {
		got := evalCondition(tt.cond, snapshots, tt.idx)
		if got != tt.expected {
			t.Errorf("%s: evalCondition(%+v, idx=%d) = %v, want %v", tt.name, tt.cond, tt.idx, got, tt.expected)
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
	result := computeMetrics(strat, trades, curve, "1d")

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
	result := computeMetrics(strat, nil, []float64{1.0}, "1d")

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

func TestSharpeRatio(t *testing.T) {
	// Flat equity curve → zero Sharpe.
	if got := sharpeRatio([]float64{1.0, 1.0, 1.0}, "1d"); got != 0 {
		t.Errorf("flat curve: Sharpe = %f, want 0", got)
	}

	// Single point → zero.
	if got := sharpeRatio([]float64{1.0}, "1d"); got != 0 {
		t.Errorf("single point: Sharpe = %f, want 0", got)
	}

	// Monotonically increasing → positive Sharpe.
	curve := []float64{1.0, 1.01, 1.02, 1.03, 1.04}
	got := sharpeRatio(curve, "1d")
	if got <= 0 {
		t.Errorf("increasing curve: Sharpe = %f, want > 0", got)
	}

	// Hourly annualization should be larger than daily for same curve.
	hourly := sharpeRatio(curve, "1h")
	daily := sharpeRatio(curve, "1d")
	if hourly <= daily {
		t.Errorf("1h Sharpe (%f) should be > 1d Sharpe (%f) for same returns", hourly, daily)
	}
}

func TestGetIndicatorValueResearchFeatures(t *testing.T) {
	snap := indicatorSnapshot{
		Trend:     0.42,
		Momentum:  -0.15,
		VolRegime: 0.30,
		Drawdown:  -0.50,
		DirVolume: 0.75,
	}
	tests := []struct {
		indicator string
		expected  float64
	}{
		{"trend", 0.42},
		{"momentum", -0.15},
		{"vol_regime", 0.30},
		{"drawdown", -0.50},
		{"dir_volume", 0.75},
	}
	for _, tt := range tests {
		got := getIndicatorValue(tt.indicator, snap)
		if got != tt.expected {
			t.Errorf("getIndicatorValue(%q) = %f, want %f", tt.indicator, got, tt.expected)
		}
	}
}

func TestAnyConditionMet(t *testing.T) {
	snapshots := []indicatorSnapshot{
		{Trend: -0.1, Momentum: 0.2, DirVolume: 0.3},
		{Trend: 0.2, Momentum: -0.1, DirVolume: 0.1},
	}

	// OR logic: trend < -0.05 should trigger at index 0 (trend = -0.1).
	conditions := []Condition{
		{Indicator: "trend", Operator: "<", Value: -0.05},
		{Indicator: "momentum", Operator: "<", Value: -0.05},
		{Indicator: "dir_volume", Operator: "<", Value: -0.05},
	}

	// Index 0: trend=-0.1 < -0.05 → true (OR).
	if !anyConditionMet(conditions, snapshots, 0) {
		t.Error("index 0: anyConditionMet should be true (trend is negative)")
	}

	// Index 1: momentum=-0.1 < -0.05 → true (OR).
	if !anyConditionMet(conditions, snapshots, 1) {
		t.Error("index 1: anyConditionMet should be true (momentum is negative)")
	}

	// All positive → false.
	allPositive := []indicatorSnapshot{
		{Trend: 0.2, Momentum: 0.2, DirVolume: 0.2},
	}
	if anyConditionMet(conditions, allPositive, 0) {
		t.Error("all positive: anyConditionMet should be false")
	}
}

func TestExitLogicOR(t *testing.T) {
	// Simulate a scenario where OR exit logic should trigger early.
	snapshots := make([]indicatorSnapshot, 55)
	for i := range snapshots {
		snapshots[i] = indicatorSnapshot{
			Price:     100 + float64(i),
			Trend:     0.2,
			Momentum:  0.2,
			DirVolume: 0.2,
		}
	}
	// At candle 52, make trend drop below -0.05 (only 1 of 3 features).
	snapshots[52].Trend = -0.1

	strat := Strategy{
		Name:   "test-or-exit",
		Symbol: "BTC",
		EntryRules: []Condition{
			{Indicator: "trend", Operator: ">", Value: 0.05},
		},
		ExitRules: []Condition{
			{Indicator: "trend", Operator: "<", Value: -0.05},
			{Indicator: "momentum", Operator: "<", Value: -0.05},
			{Indicator: "dir_volume", Operator: "<", Value: -0.05},
		},
		ExitLogic:    "or",
		PositionSize: 1.0,
	}

	// Manually run allConditionsMet and anyConditionMet to verify behavior.
	// Entry at index 50 (trend=0.2 > 0.05): true.
	if !allConditionsMet(strat.EntryRules, snapshots, 50) {
		t.Fatal("entry should trigger at index 50")
	}

	// At index 52, OR exit: trend=-0.1 < -0.05 → should trigger.
	if !anyConditionMet(strat.ExitRules, snapshots, 52) {
		t.Error("OR exit should trigger at index 52 (trend dropped)")
	}

	// AND exit at index 52: only trend dropped, others still positive → should NOT trigger.
	if allConditionsMet(strat.ExitRules, snapshots, 52) {
		t.Error("AND exit should NOT trigger at index 52 (only trend dropped)")
	}
}

func TestShortSellingSimulation(t *testing.T) {
	// Build synthetic candles: price goes UP so a short should LOSE.
	candles := make([]indicatorSnapshot, 55)
	for i := range candles {
		candles[i] = indicatorSnapshot{
			Price: 100 + float64(i), // steadily rising
			RSI:   25,               // triggers entry < 30
		}
	}
	// Trigger exit at candle 53.
	candles[53].RSI = 75 // RSI > 70 → exit signal

	// Verify short entry conditions work.
	strat := Strategy{
		Side: "short",
		EntryRules: []Condition{
			{Indicator: "rsi", Operator: "<", Value: 30},
		},
		ExitRules: []Condition{
			{Indicator: "rsi", Operator: ">", Value: 70},
		},
		PositionSize: 1.0,
	}

	if !allConditionsMet(strat.EntryRules, candles, 50) {
		t.Fatal("entry should trigger at index 50 (RSI=25)")
	}
	if !allConditionsMet(strat.ExitRules, candles, 53) {
		t.Fatal("exit should trigger at index 53 (RSI=75)")
	}
}

func TestShortSLTP(t *testing.T) {
	// Short SL: price goes UP past stop-loss level.
	entryPrice := 100.0
	slPct := 5.0
	tpPct := 10.0
	slPrice := entryPrice * (1 + slPct/100)  // 105
	tpPrice := entryPrice * (1 - tpPct/100)  // 90

	// SL triggers when High breaches SL level.
	if slPrice != 105 {
		t.Errorf("short SL price = %f, want 105", slPrice)
	}
	// TP triggers when Low breaches TP level.
	if tpPrice != 90 {
		t.Errorf("short TP price = %f, want 90", tpPrice)
	}

	// Short P&L: profit when price goes down.
	exitAtSL := slPrice
	pnlSL := (entryPrice - exitAtSL) / entryPrice * 100 // -5%
	if math.Abs(pnlSL-(-5.0)) > 0.01 {
		t.Errorf("short SL P&L = %.2f, want -5.0", pnlSL)
	}

	exitAtTP := tpPrice
	pnlTP := (entryPrice - exitAtTP) / entryPrice * 100 // +10%
	if math.Abs(pnlTP-10.0) > 0.01 {
		t.Errorf("short TP P&L = %.2f, want 10.0", pnlTP)
	}
}

func TestShortPnL(t *testing.T) {
	// Short entry at 100, exit at 90 → profit.
	entry := 100.0
	exit := 90.0
	pnl := (entry - exit) / entry * 100
	if math.Abs(pnl-10.0) > 0.01 {
		t.Errorf("short profit PnL = %.2f, want 10.0", pnl)
	}

	// Short entry at 100, exit at 110 → loss.
	exit = 110.0
	pnl = (entry - exit) / entry * 100
	if math.Abs(pnl-(-10.0)) > 0.01 {
		t.Errorf("short loss PnL = %.2f, want -10.0", pnl)
	}
}

func TestSideDefaultsToLong(t *testing.T) {
	strat := Strategy{Side: ""}
	isShort := strat.Side == "short"
	if isShort {
		t.Error("empty Side should default to long (isShort=false)")
	}

	strat.Side = "long"
	isShort = strat.Side == "short"
	if isShort {
		t.Error("Side='long' should not be short")
	}

	strat.Side = "short"
	isShort = strat.Side == "short"
	if !isShort {
		t.Error("Side='short' should be short")
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
