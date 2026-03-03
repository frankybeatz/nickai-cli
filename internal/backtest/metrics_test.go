package backtest

import (
	"math"
	"testing"
	"time"
)

func TestSortinoRatio(t *testing.T) {
	// Monotonically increasing curve → positive Sortino (no downside).
	upCurve := []float64{1.0, 1.01, 1.02, 1.03, 1.04, 1.05}
	got := sortinoRatio(upCurve, "1d")
	if !math.IsInf(got, 1) && got <= 0 {
		t.Errorf("uptrend: Sortino = %f, want > 0 or +Inf", got)
	}

	// Flat curve → zero Sortino.
	flatCurve := []float64{1.0, 1.0, 1.0, 1.0}
	got = sortinoRatio(flatCurve, "1d")
	if got != 0 {
		t.Errorf("flat curve: Sortino = %f, want 0", got)
	}

	// Curve with some drawdown → finite positive (mean is positive overall).
	mixedCurve := []float64{1.0, 1.05, 0.98, 1.03, 1.10, 1.08, 1.15}
	got = sortinoRatio(mixedCurve, "1d")
	if got <= 0 {
		t.Errorf("mixed curve: Sortino = %f, want > 0", got)
	}

	// Single point → zero.
	got = sortinoRatio([]float64{1.0}, "1d")
	if got != 0 {
		t.Errorf("single point: Sortino = %f, want 0", got)
	}
}

func TestCalmarRatio(t *testing.T) {
	// Known curve: start=1.0, end=1.20, peak=1.25 at index 2.
	// maxDD = (1.25-1.10)/1.25 * 100 = 12%.
	curve := []float64{1.0, 1.15, 1.25, 1.10, 1.20}
	got := calmarRatio(curve, "1d")
	if got <= 0 {
		t.Errorf("Calmar = %f, want > 0", got)
	}

	// No drawdown → should be Inf (or very large) for positive CAGR.
	noDDCurve := []float64{1.0, 1.05, 1.10, 1.15}
	got = calmarRatio(noDDCurve, "1d")
	if !math.IsInf(got, 1) {
		t.Errorf("no drawdown: Calmar = %f, want +Inf", got)
	}

	// Single point → zero.
	got = calmarRatio([]float64{1.0}, "1d")
	if got != 0 {
		t.Errorf("single point: Calmar = %f, want 0", got)
	}
}

func TestOmegaRatio(t *testing.T) {
	// All positive returns → Inf.
	allUpCurve := []float64{1.0, 1.01, 1.02, 1.03}
	got := omegaRatio(allUpCurve)
	if !math.IsInf(got, 1) {
		t.Errorf("all positive: Omega = %f, want +Inf", got)
	}

	// All negative returns → 0.
	allDownCurve := []float64{1.0, 0.99, 0.98, 0.97}
	got = omegaRatio(allDownCurve)
	if got != 0 {
		t.Errorf("all negative: Omega = %f, want 0", got)
	}

	// Mixed: gains > losses → Omega > 1.
	mixedCurve := []float64{1.0, 1.05, 0.98, 1.10}
	got = omegaRatio(mixedCurve)
	if got <= 0 {
		t.Errorf("mixed curve: Omega = %f, want > 0", got)
	}

	// Empty curve → 0.
	got = omegaRatio([]float64{1.0})
	if got != 0 {
		t.Errorf("single point: Omega = %f, want 0", got)
	}
}

func TestRecoveryFactor(t *testing.T) {
	// Simple division: 20% return / 10% drawdown = 2.0.
	got := recoveryFactor(20.0, 10.0)
	if math.Abs(got-2.0) > 0.001 {
		t.Errorf("recoveryFactor(20, 10) = %f, want 2.0", got)
	}

	// No drawdown with positive return → Inf.
	got = recoveryFactor(15.0, 0)
	if !math.IsInf(got, 1) {
		t.Errorf("recoveryFactor(15, 0) = %f, want +Inf", got)
	}

	// Negative return / positive drawdown → negative.
	got = recoveryFactor(-5.0, 10.0)
	if got >= 0 {
		t.Errorf("recoveryFactor(-5, 10) = %f, want < 0", got)
	}

	// Zero return / zero drawdown → 0.
	got = recoveryFactor(0, 0)
	if got != 0 {
		t.Errorf("recoveryFactor(0, 0) = %f, want 0", got)
	}
}

func TestExpectancy(t *testing.T) {
	now := time.Now()

	// Known: 2 wins (+10%, +20%), 1 loss (-5%).
	// winRate = 2/3, avgWin = 15, lossRate = 1/3, avgLoss = 5.
	// expectancy = (2/3 * 15) - (1/3 * 5) = 10 - 1.667 = 8.333.
	trades := []Trade{
		{PnLPct: 10, EntryTime: now, ExitTime: now.Add(time.Hour)},
		{PnLPct: 20, EntryTime: now, ExitTime: now.Add(time.Hour)},
		{PnLPct: -5, EntryTime: now, ExitTime: now.Add(time.Hour)},
	}
	got := expectancy(trades)
	expected := (2.0/3.0)*15.0 - (1.0/3.0)*5.0
	if math.Abs(got-expected) > 0.01 {
		t.Errorf("expectancy = %f, want ~%f", got, expected)
	}

	// No trades → 0.
	got = expectancy(nil)
	if got != 0 {
		t.Errorf("expectancy(nil) = %f, want 0", got)
	}

	// All wins → positive, no loss component.
	allWins := []Trade{
		{PnLPct: 5, EntryTime: now, ExitTime: now.Add(time.Hour)},
		{PnLPct: 10, EntryTime: now, ExitTime: now.Add(time.Hour)},
	}
	got = expectancy(allWins)
	if got <= 0 {
		t.Errorf("all wins: expectancy = %f, want > 0", got)
	}

	// All losses → negative.
	allLosses := []Trade{
		{PnLPct: -5, EntryTime: now, ExitTime: now.Add(time.Hour)},
		{PnLPct: -10, EntryTime: now, ExitTime: now.Add(time.Hour)},
	}
	got = expectancy(allLosses)
	if got >= 0 {
		t.Errorf("all losses: expectancy = %f, want < 0", got)
	}
}

func TestMaxDrawdownDuration(t *testing.T) {
	// Curve: peak at 1.10, then underwater for 3 bars.
	// 1.0 → 1.10 → 1.05 → 1.02 → 1.08 → 1.15
	// Underwater: indices 2,3,4 (3 bars below peak of 1.10).
	curve := []float64{1.0, 1.10, 1.05, 1.02, 1.08, 1.15}
	got := maxDrawdownDuration(curve)
	if got != 3 {
		t.Errorf("maxDrawdownDuration = %d, want 3", got)
	}

	// No drawdown → 0.
	noDDCurve := []float64{1.0, 1.05, 1.10, 1.15}
	got = maxDrawdownDuration(noDDCurve)
	if got != 0 {
		t.Errorf("no drawdown: maxDrawdownDuration = %d, want 0", got)
	}

	// Empty curve → 0.
	got = maxDrawdownDuration(nil)
	if got != 0 {
		t.Errorf("empty curve: maxDrawdownDuration = %d, want 0", got)
	}

	// Entire period underwater after first bar.
	longDD := []float64{1.0, 1.10, 1.05, 1.03, 1.01, 0.99}
	got = maxDrawdownDuration(longDD)
	if got != 4 {
		t.Errorf("long drawdown: maxDrawdownDuration = %d, want 4", got)
	}
}

func TestTimeInMarket(t *testing.T) {
	now := time.Now()

	// 1 trade, 24h holding, totalBars=10, daily interval.
	// Duration = 24h. Total = 10 * 24h = 240h. Pct = 10%.
	trades := []Trade{
		{
			EntryTime: now,
			ExitTime:  now.Add(24 * time.Hour),
		},
	}
	got := timeInMarket(trades, 10, "1d")
	if math.Abs(got-10.0) > 0.1 {
		t.Errorf("timeInMarket = %f, want ~10.0", got)
	}

	// No trades → 0.
	got = timeInMarket(nil, 100, "1d")
	if got != 0 {
		t.Errorf("no trades: timeInMarket = %f, want 0", got)
	}
}

func TestAvgTradeDurationBars(t *testing.T) {
	now := time.Now()

	// Two trades: 48h and 24h, daily bars.
	// 48h/24h = 2 bars, 24h/24h = 1 bar, avg = 1.5 bars.
	trades := []Trade{
		{EntryTime: now, ExitTime: now.Add(48 * time.Hour)},
		{EntryTime: now, ExitTime: now.Add(24 * time.Hour)},
	}
	got := avgTradeDurationBars(trades, "1d")
	if math.Abs(got-1.5) > 0.01 {
		t.Errorf("avgTradeDurationBars = %f, want 1.5", got)
	}

	// No trades → 0.
	got = avgTradeDurationBars(nil, "1d")
	if got != 0 {
		t.Errorf("no trades: avgTradeDurationBars = %f, want 0", got)
	}

	// 4h bars: 8h trade = 2 bars.
	trades4h := []Trade{
		{EntryTime: now, ExitTime: now.Add(8 * time.Hour)},
	}
	got = avgTradeDurationBars(trades4h, "4h")
	if math.Abs(got-2.0) > 0.01 {
		t.Errorf("4h: avgTradeDurationBars = %f, want 2.0", got)
	}
}

func TestTailRatio(t *testing.T) {
	// Build a known distribution of 100 returns.
	// Linearly spaced from -0.10 to +0.10.
	curve := make([]float64, 101) // 101 points → 100 returns
	curve[0] = 1.0
	for i := 1; i <= 100; i++ {
		// Return from -0.10 to +0.10.
		r := -0.10 + 0.20*float64(i-1)/99.0
		curve[i] = curve[i-1] * (1 + r)
	}

	got := tailRatio(curve)
	// For a symmetric distribution around 0, |P95| / |P5| should be near 1.
	// But our distribution is linearly spaced from -0.10 to +0.10, so
	// P95 ~= +0.09 and P5 ~= -0.09, giving tail ratio ~= 1.0.
	if got <= 0 {
		t.Errorf("tailRatio = %f, want > 0", got)
	}

	// Too few returns → 0.
	got = tailRatio([]float64{1.0, 1.01})
	if got != 0 {
		t.Errorf("few returns: tailRatio = %f, want 0", got)
	}
}

func TestHistoricalVaR(t *testing.T) {
	// Build a known curve: 100 returns, linearly from -10% to +10%.
	curve := make([]float64, 101)
	curve[0] = 1.0
	for i := 1; i <= 100; i++ {
		r := -0.10 + 0.20*float64(i-1)/99.0
		curve[i] = curve[i-1] * (1 + r)
	}

	got := historicalVaR(curve, 0.95)
	// VaR should be positive (representing a loss).
	// At 5th percentile of returns from -10% to +10%, we expect ~8-10% VaR.
	if got <= 0 {
		t.Errorf("VaR95 = %f, want > 0", got)
	}

	// Empty curve → 0.
	got = historicalVaR([]float64{1.0}, 0.95)
	if got != 0 {
		t.Errorf("empty: VaR95 = %f, want 0", got)
	}
}

func TestHistoricalCVaR(t *testing.T) {
	// Same curve as VaR test.
	curve := make([]float64, 101)
	curve[0] = 1.0
	for i := 1; i <= 100; i++ {
		r := -0.10 + 0.20*float64(i-1)/99.0
		curve[i] = curve[i-1] * (1 + r)
	}

	got := historicalCVaR(curve, 0.95)
	// CVaR should be >= VaR (expected shortfall is worse than VaR).
	varVal := historicalVaR(curve, 0.95)
	if got < varVal {
		t.Errorf("CVaR95 (%f) should be >= VaR95 (%f)", got, varVal)
	}
	if got <= 0 {
		t.Errorf("CVaR95 = %f, want > 0", got)
	}

	// Empty curve → 0.
	got = historicalCVaR([]float64{1.0}, 0.95)
	if got != 0 {
		t.Errorf("empty: CVaR95 = %f, want 0", got)
	}
}

func TestMonteCarloSmoke(t *testing.T) {
	now := time.Now()

	trades := []Trade{
		{PnLPct: 5, EntryTime: now, ExitTime: now.Add(time.Hour)},
		{PnLPct: -3, EntryTime: now.Add(2 * time.Hour), ExitTime: now.Add(3 * time.Hour)},
		{PnLPct: 8, EntryTime: now.Add(4 * time.Hour), ExitTime: now.Add(5 * time.Hour)},
		{PnLPct: -2, EntryTime: now.Add(6 * time.Hour), ExitTime: now.Add(7 * time.Hour)},
		{PnLPct: 6, EntryTime: now.Add(8 * time.Hour), ExitTime: now.Add(9 * time.Hour)},
		{PnLPct: -1, EntryTime: now.Add(10 * time.Hour), ExitTime: now.Add(11 * time.Hour)},
		{PnLPct: 4, EntryTime: now.Add(12 * time.Hour), ExitTime: now.Add(13 * time.Hour)},
		{PnLPct: -4, EntryTime: now.Add(14 * time.Hour), ExitTime: now.Add(15 * time.Hour)},
		{PnLPct: 7, EntryTime: now.Add(16 * time.Hour), ExitTime: now.Add(17 * time.Hour)},
		{PnLPct: -2, EntryTime: now.Add(18 * time.Hour), ExitTime: now.Add(19 * time.Hour)},
	}

	// Build equity curve from trades.
	equity := 1.0
	curve := []float64{1.0}
	for _, tr := range trades {
		equity *= (1 + tr.PnLPct/100)
		curve = append(curve, equity)
	}

	originalResult := &Result{
		SharpeRatio: sharpeRatio(curve, "1d"),
		MaxDrawdown: maxDrawdown(curve),
	}

	mc := RunMonteCarlo(trades, originalResult, 500, "1d")

	if mc == nil {
		t.Fatal("RunMonteCarlo returned nil")
	}
	if mc.Simulations != 500 {
		t.Errorf("Simulations = %d, want 500", mc.Simulations)
	}
	if mc.PValue < 0 || mc.PValue > 1 {
		t.Errorf("PValue = %f, want between 0 and 1", mc.PValue)
	}
	if mc.DD95 < 0 {
		t.Errorf("DD95 = %f, want >= 0", mc.DD95)
	}
	if mc.DD99 < mc.DD95 {
		t.Errorf("DD99 (%f) should be >= DD95 (%f)", mc.DD99, mc.DD95)
	}
	if mc.SharpeLower95 > mc.SharpeUpper95 {
		t.Errorf("SharpeLower95 (%f) should be <= SharpeUpper95 (%f)", mc.SharpeLower95, mc.SharpeUpper95)
	}
}

func TestMonteCarloNoTrades(t *testing.T) {
	result := &Result{
		SharpeRatio: 0,
		MaxDrawdown: 0,
	}
	mc := RunMonteCarlo(nil, result, 100, "1d")
	if mc == nil {
		t.Fatal("RunMonteCarlo returned nil for no trades")
	}
	if mc.Simulations != 100 {
		t.Errorf("Simulations = %d, want 100", mc.Simulations)
	}
}

func TestPeriodsPerYear(t *testing.T) {
	if got := periodsPerYear("1d"); got != 365 {
		t.Errorf("1d: periodsPerYear = %f, want 365", got)
	}
	if got := periodsPerYear("4h"); got != 365*6 {
		t.Errorf("4h: periodsPerYear = %f, want %f", got, 365.0*6)
	}
	if got := periodsPerYear("1h"); got != 365*24 {
		t.Errorf("1h: periodsPerYear = %f, want %f", got, 365.0*24)
	}
}

func TestEquityReturns(t *testing.T) {
	curve := []float64{1.0, 1.10, 1.05, 1.20}
	returns := equityReturns(curve)
	if len(returns) != 3 {
		t.Fatalf("len(returns) = %d, want 3", len(returns))
	}
	// 1.0 → 1.10: +10%
	if math.Abs(returns[0]-0.10) > 0.001 {
		t.Errorf("returns[0] = %f, want ~0.10", returns[0])
	}
	// 1.10 → 1.05: -4.545%
	expected := (1.05 - 1.10) / 1.10
	if math.Abs(returns[1]-expected) > 0.001 {
		t.Errorf("returns[1] = %f, want ~%f", returns[1], expected)
	}

	// Single point → nil.
	if got := equityReturns([]float64{1.0}); got != nil {
		t.Errorf("single point: returns = %v, want nil", got)
	}
}

func TestComputeMetricsAdvanced(t *testing.T) {
	// Verify that computeMetrics populates advanced metric fields.
	now := time.Now()
	trades := []Trade{
		{EntryPrice: 100, ExitPrice: 110, PnLPct: 10, EntryTime: now, ExitTime: now.Add(24 * time.Hour), Reason: "exit_signal"},
		{EntryPrice: 110, ExitPrice: 105, PnLPct: -4.55, EntryTime: now.Add(48 * time.Hour), ExitTime: now.Add(72 * time.Hour), Reason: "stop_loss"},
		{EntryPrice: 105, ExitPrice: 120, PnLPct: 14.29, EntryTime: now.Add(96 * time.Hour), ExitTime: now.Add(120 * time.Hour), Reason: "take_profit"},
	}

	// Build a realistic equity curve with enough points for tail ratio.
	curve := make([]float64, 0, 50)
	curve = append(curve, 1.0)
	for i := 1; i < 50; i++ {
		// Simulate small random-ish returns.
		r := 0.001 * float64(i%7-3)
		curve = append(curve, curve[len(curve)-1]*(1+r))
	}

	strat := Strategy{Name: "test-advanced", Symbol: "BTC", Period: "30d"}
	result := computeMetrics(strat, trades, curve, "1d")

	// SortinoRatio should be computed (could be any value).
	// Just verify it doesn't panic and is a number.
	if math.IsNaN(result.SortinoRatio) {
		t.Error("SortinoRatio is NaN")
	}
	if math.IsNaN(result.CalmarRatio) {
		t.Error("CalmarRatio is NaN")
	}
	if math.IsNaN(result.OmegaRatio) {
		t.Error("OmegaRatio is NaN")
	}
	if math.IsNaN(result.Expectancy) {
		t.Error("Expectancy is NaN")
	}

	// Expectancy should match manual calculation for these trades.
	expectedExp := (2.0/3.0)*((10.0+14.29)/2.0) - (1.0/3.0)*4.55
	if math.Abs(result.Expectancy-expectedExp) > 0.01 {
		t.Errorf("Expectancy = %f, want ~%f", result.Expectancy, expectedExp)
	}
}
