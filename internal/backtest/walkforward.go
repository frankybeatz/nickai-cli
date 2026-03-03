package backtest

import (
	"fmt"
	"math"

	"github.com/nickai/cli/internal/market"
)

// WFAConfig configures walk-forward analysis.
type WFAConfig struct {
	Windows  int     // number of IS/OOS windows (default 5)
	OOSRatio float64 // fraction of each window for OOS (default 0.3)
	Anchored bool    // if true, IS window grows from start (anchored); if false, rolling
}

// WFAResult holds the output of walk-forward analysis.
type WFAResult struct {
	Windows     []WFAWindow
	CombinedOOS *Result // concatenated OOS equity curves
	Efficiency  float64 // OOS_Sharpe / IS_Sharpe (>0.5 = generalizes)
	Robust      bool    // true if Efficiency > 0.5
}

// WFAWindow holds results for a single IS/OOS window.
type WFAWindow struct {
	WindowNum int
	ISStart   int // candle index
	ISEnd     int
	OOSStart  int
	OOSEnd    int
	ISResult  *Result
	OOSResult *Result
}

// wfaDefaults fills in zero-value config fields with defaults.
func wfaDefaults(cfg WFAConfig) WFAConfig {
	if cfg.Windows <= 0 {
		cfg.Windows = 5
	}
	if cfg.OOSRatio <= 0 || cfg.OOSRatio >= 1 {
		cfg.OOSRatio = 0.3
	}
	return cfg
}

// splitWindows computes IS/OOS index ranges for walk-forward analysis.
// Returns a slice of WFAWindow with index ranges filled in (results are nil).
func splitWindows(totalCandles int, cfg WFAConfig) ([]WFAWindow, error) {
	cfg = wfaDefaults(cfg)

	if totalCandles < 100 {
		return nil, fmt.Errorf("insufficient candles for WFA: got %d, need at least 100", totalCandles)
	}

	n := cfg.Windows
	if n <= 0 {
		return nil, fmt.Errorf("windows must be > 0, got %d", n)
	}

	// Each window covers totalCandles/n candles.
	windowSize := totalCandles / n
	if windowSize < 60 {
		return nil, fmt.Errorf("window size too small: %d candles per window (need at least 60 for warmup + trading)", windowSize)
	}

	oosSize := int(float64(windowSize) * cfg.OOSRatio)
	if oosSize < 1 {
		oosSize = 1
	}

	windows := make([]WFAWindow, n)
	for w := 0; w < n; w++ {
		windowEnd := (w + 1) * windowSize
		if w == n-1 {
			// Last window gets any remaining candles.
			windowEnd = totalCandles
		}

		oosEnd := windowEnd
		oosStart := oosEnd - oosSize

		var isStart int
		if cfg.Anchored {
			// Anchored: IS always starts from the beginning.
			isStart = 0
		} else {
			// Rolling: IS starts at the beginning of this window.
			isStart = w * windowSize
		}
		isEnd := oosStart

		// Ensure IS has enough candles for warmup (50) + at least some trading.
		if isEnd-isStart < 55 {
			return nil, fmt.Errorf("window %d: IS segment too small (%d candles), need at least 55", w+1, isEnd-isStart)
		}
		// Ensure OOS has some candles to trade.
		if oosEnd-oosStart < 1 {
			return nil, fmt.Errorf("window %d: OOS segment is empty", w+1)
		}

		windows[w] = WFAWindow{
			WindowNum: w + 1,
			ISStart:   isStart,
			ISEnd:     isEnd,
			OOSStart:  oosStart,
			OOSEnd:    oosEnd,
		}
	}

	return windows, nil
}

// RunWFA performs walk-forward analysis on a strategy.
// It splits the data into N windows, each with an in-sample (IS) and out-of-sample (OOS) portion.
// The strategy is evaluated on each segment. Efficiency = avg(OOS_Sharpe) / avg(IS_Sharpe).
func RunWFA(strat Strategy, config WFAConfig) (*WFAResult, error) {
	if strat.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	config = wfaDefaults(config)

	// Fetch all candles for the full period once.
	days, err := ParsePeriod(strat.Period)
	if err != nil {
		return nil, err
	}

	interval := market.IntervalForPeriod(days)
	limit := days
	if interval == "4h" {
		limit = days * 6
	} else if interval == "1h" {
		limit = days * 24
	}

	candles, err := market.FetchKlinesPaginated(strat.Symbol, interval, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market data: %w", err)
	}

	return RunWFAWithCandles(strat, candles, interval, config)
}

// RunWFAWithCandles performs walk-forward analysis on pre-fetched candle data.
// This allows WFA to be used without re-fetching from Binance.
func RunWFAWithCandles(strat Strategy, candles []market.Candle, interval string, config WFAConfig) (*WFAResult, error) {
	config = wfaDefaults(config)

	windows, err := splitWindows(len(candles), config)
	if err != nil {
		return nil, fmt.Errorf("WFA window split failed: %w", err)
	}

	// Run strategy on each IS and OOS segment.
	for i := range windows {
		w := &windows[i]

		// In-sample run.
		isCandles := candles[w.ISStart:w.ISEnd]
		isResult, err := RunWithCandles(strat, isCandles, interval)
		if err != nil {
			return nil, fmt.Errorf("window %d IS failed: %w", w.WindowNum, err)
		}
		w.ISResult = isResult

		// Out-of-sample run.
		oosCandles := candles[w.OOSStart:w.OOSEnd]
		// OOS segment might be too small for warmup. If so, extend backwards
		// to include warmup candles but only count trades in the OOS range.
		// For simplicity, we prepend warmup candles from before OOSStart.
		warmup := 50
		oosWithWarmupStart := w.OOSStart - warmup
		if oosWithWarmupStart < 0 {
			oosWithWarmupStart = 0
		}
		oosWithWarmup := candles[oosWithWarmupStart:w.OOSEnd]

		if len(oosWithWarmup) < 51 {
			// Not enough data even with warmup extension; use what we have.
			// This might produce no trades, which is valid.
			oosResult, oosErr := RunWithCandles(strat, oosCandles, interval)
			if oosErr != nil {
				// If insufficient data, create an empty result.
				oosResult = &Result{
					Strategy:    strat.Name,
					Symbol:      strat.Symbol,
					EquityCurve: []float64{1.0},
				}
			}
			w.OOSResult = oosResult
		} else {
			oosResult, oosErr := RunWithCandles(strat, oosWithWarmup, interval)
			if oosErr != nil {
				return nil, fmt.Errorf("window %d OOS failed: %w", w.WindowNum, oosErr)
			}
			w.OOSResult = oosResult
		}
	}

	// Compute WFA efficiency: avg(OOS Sharpe) / avg(IS Sharpe).
	var isSharpeSum, oosSharpeSum float64
	var isSharpeCount, oosSharpeCount int

	for _, w := range windows {
		if w.ISResult != nil && !math.IsNaN(w.ISResult.SharpeRatio) && !math.IsInf(w.ISResult.SharpeRatio, 0) {
			isSharpeSum += w.ISResult.SharpeRatio
			isSharpeCount++
		}
		if w.OOSResult != nil && !math.IsNaN(w.OOSResult.SharpeRatio) && !math.IsInf(w.OOSResult.SharpeRatio, 0) {
			oosSharpeSum += w.OOSResult.SharpeRatio
			oosSharpeCount++
		}
	}

	efficiency := 0.0
	if isSharpeCount > 0 && oosSharpeCount > 0 {
		avgIS := isSharpeSum / float64(isSharpeCount)
		avgOOS := oosSharpeSum / float64(oosSharpeCount)
		if avgIS != 0 {
			efficiency = avgOOS / avgIS
		}
	}

	// Build combined OOS equity curve by concatenating OOS results.
	combinedOOS := combineOOSResults(windows, strat, interval)

	return &WFAResult{
		Windows:     windows,
		CombinedOOS: combinedOOS,
		Efficiency:  efficiency,
		Robust:      efficiency > 0.5,
	}, nil
}

// combineOOSResults concatenates OOS equity curves into a single Result.
func combineOOSResults(windows []WFAWindow, strat Strategy, interval string) *Result {
	var allTrades []Trade
	combinedCurve := []float64{1.0}

	lastEquity := 1.0
	for _, w := range windows {
		if w.OOSResult == nil || len(w.OOSResult.EquityCurve) < 2 {
			continue
		}

		allTrades = append(allTrades, w.OOSResult.Trades...)

		// Scale OOS equity curve to continue from where the last window ended.
		oosCurve := w.OOSResult.EquityCurve
		startValue := oosCurve[0]
		if startValue == 0 {
			startValue = 1.0
		}

		for j := 1; j < len(oosCurve); j++ {
			scaledValue := lastEquity * (oosCurve[j] / startValue)
			combinedCurve = append(combinedCurve, scaledValue)
		}
		lastEquity = combinedCurve[len(combinedCurve)-1]
	}

	return computeMetrics(strat, allTrades, combinedCurve, interval)
}
