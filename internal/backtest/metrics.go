package backtest

import (
	"math"
	"sort"
)

// periodsPerYear returns the annualization factor for a given candle interval.
func periodsPerYear(candleInterval string) float64 {
	switch candleInterval {
	case "1h":
		return 365 * 24
	case "4h":
		return 365 * 6
	default:
		return 365 // daily
	}
}

// equityReturns computes per-period returns from an equity curve.
func equityReturns(equityCurve []float64) []float64 {
	if len(equityCurve) < 2 {
		return nil
	}
	returns := make([]float64, len(equityCurve)-1)
	for i := 1; i < len(equityCurve); i++ {
		if equityCurve[i-1] != 0 {
			returns[i-1] = (equityCurve[i] - equityCurve[i-1]) / equityCurve[i-1]
		}
	}
	return returns
}

// sortinoRatio computes the Sortino ratio (downside deviation only).
// Only penalizes returns below 0 (or a threshold).
func sortinoRatio(equityCurve []float64, candleInterval string) float64 {
	returns := equityReturns(equityCurve)
	if len(returns) == 0 {
		return 0
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	// Downside deviation: sqrt(mean(min(r, 0)^2))
	downsideSum := 0.0
	for _, r := range returns {
		if r < 0 {
			downsideSum += r * r
		}
	}
	downsideDev := math.Sqrt(downsideSum / float64(len(returns)))

	if downsideDev == 0 {
		if mean > 0 {
			return math.Inf(1)
		}
		return 0
	}

	return (mean / downsideDev) * math.Sqrt(periodsPerYear(candleInterval))
}

// calmarRatio computes CAGR / MaxDrawdown%.
// CAGR = (final/initial)^(periodsPerYear/n) - 1.
func calmarRatio(equityCurve []float64, candleInterval string) float64 {
	if len(equityCurve) < 2 {
		return 0
	}

	initial := equityCurve[0]
	final := equityCurve[len(equityCurve)-1]
	if initial == 0 || final <= 0 {
		return 0
	}

	n := float64(len(equityCurve) - 1) // number of periods
	ppy := periodsPerYear(candleInterval)

	cagr := math.Pow(final/initial, ppy/n) - 1

	maxDD := maxDrawdown(equityCurve) // already in percent
	if maxDD == 0 {
		if cagr > 0 {
			return math.Inf(1)
		}
		return 0
	}

	// Convert maxDD from percent to fraction for consistent ratio.
	return cagr / (maxDD / 100)
}

// omegaRatio computes sum of gains / sum of losses (at threshold 0).
func omegaRatio(equityCurve []float64) float64 {
	returns := equityReturns(equityCurve)
	if len(returns) == 0 {
		return 0
	}

	sumGains := 0.0
	sumLosses := 0.0
	for _, r := range returns {
		if r > 0 {
			sumGains += r
		} else if r < 0 {
			sumLosses += math.Abs(r)
		}
	}

	if sumLosses == 0 {
		if sumGains > 0 {
			return math.Inf(1)
		}
		return 0
	}

	return sumGains / sumLosses
}

// recoveryFactor computes total return / max drawdown.
func recoveryFactor(totalReturnPct, maxDrawdownPct float64) float64 {
	if maxDrawdownPct == 0 {
		if totalReturnPct > 0 {
			return math.Inf(1)
		}
		return 0
	}
	return totalReturnPct / maxDrawdownPct
}

// expectancy computes (winRate * avgWin) - (lossRate * avgLoss).
func expectancy(trades []Trade) float64 {
	if len(trades) == 0 {
		return 0
	}

	wins := 0
	losses := 0
	totalWinPct := 0.0
	totalLossPct := 0.0

	for _, t := range trades {
		if t.PnLPct > 0 {
			wins++
			totalWinPct += t.PnLPct
		} else {
			losses++
			totalLossPct += math.Abs(t.PnLPct)
		}
	}

	total := float64(len(trades))
	winRate := float64(wins) / total
	lossRate := float64(losses) / total

	avgWin := 0.0
	if wins > 0 {
		avgWin = totalWinPct / float64(wins)
	}
	avgLoss := 0.0
	if losses > 0 {
		avgLoss = totalLossPct / float64(losses)
	}

	return (winRate * avgWin) - (lossRate * avgLoss)
}

// maxDrawdownDuration computes the longest period (in bars) spent underwater.
func maxDrawdownDuration(equityCurve []float64) int {
	if len(equityCurve) == 0 {
		return 0
	}

	peak := equityCurve[0]
	currentDuration := 0
	maxDuration := 0

	for _, v := range equityCurve {
		if v >= peak {
			peak = v
			currentDuration = 0
		} else {
			currentDuration++
			if currentDuration > maxDuration {
				maxDuration = currentDuration
			}
		}
	}

	return maxDuration
}

// timeInMarket computes what % of total bars had an open position.
func timeInMarket(trades []Trade, totalBars int, candleInterval string) float64 {
	if len(trades) == 0 || totalBars == 0 {
		return 0
	}

	// Determine bar duration from candle interval.
	barDuration := barDurationHours(candleInterval)

	totalDurationHours := 0.0
	for _, t := range trades {
		dur := t.ExitTime.Sub(t.EntryTime).Hours()
		totalDurationHours += dur
	}

	totalBacktestHours := float64(totalBars) * barDuration
	if totalBacktestHours == 0 {
		return 0
	}

	pct := (totalDurationHours / totalBacktestHours) * 100
	if pct > 100 {
		pct = 100
	}
	return pct
}

// avgTradeDurationBars computes average holding period in bars.
func avgTradeDurationBars(trades []Trade, candleInterval string) float64 {
	if len(trades) == 0 {
		return 0
	}

	barDuration := barDurationHours(candleInterval)
	if barDuration == 0 {
		return 0
	}

	totalBars := 0.0
	for _, t := range trades {
		dur := t.ExitTime.Sub(t.EntryTime).Hours()
		totalBars += dur / barDuration
	}

	return totalBars / float64(len(trades))
}

// barDurationHours returns the number of hours per bar for a given interval.
func barDurationHours(candleInterval string) float64 {
	switch candleInterval {
	case "1h":
		return 1
	case "4h":
		return 4
	default:
		return 24 // daily
	}
}

// tailRatio computes |P95| / |P5| of the return distribution.
func tailRatio(equityCurve []float64) float64 {
	returns := equityReturns(equityCurve)
	if len(returns) < 20 {
		// Not enough data for meaningful percentiles.
		return 0
	}

	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)

	n := len(sorted)
	p95 := sorted[int(math.Floor(0.95*float64(n-1)))]
	p5 := sorted[int(math.Floor(0.05*float64(n-1)))]

	absP5 := math.Abs(p5)
	if absP5 == 0 {
		if math.Abs(p95) > 0 {
			return math.Inf(1)
		}
		return 0
	}

	return math.Abs(p95) / absP5
}

// historicalVaR computes the value at risk at a given confidence level.
// For confidence=0.95, it returns the loss at the 5th percentile.
func historicalVaR(equityCurve []float64, confidence float64) float64 {
	returns := equityReturns(equityCurve)
	if len(returns) == 0 {
		return 0
	}

	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)

	alpha := 1 - confidence // e.g. 0.05 for 95% confidence
	idx := int(math.Floor(alpha * float64(len(sorted))))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	// VaR is expressed as a positive number representing potential loss.
	return -sorted[idx] * 100 // convert to percentage
}

// historicalCVaR computes the expected shortfall beyond VaR.
// It is the average of all returns worse than the VaR threshold.
func historicalCVaR(equityCurve []float64, confidence float64) float64 {
	returns := equityReturns(equityCurve)
	if len(returns) == 0 {
		return 0
	}

	sorted := make([]float64, len(returns))
	copy(sorted, returns)
	sort.Float64s(sorted)

	alpha := 1 - confidence
	cutoff := int(math.Floor(alpha * float64(len(sorted))))
	if cutoff == 0 {
		cutoff = 1 // at least one observation
	}
	if cutoff > len(sorted) {
		cutoff = len(sorted)
	}

	sum := 0.0
	for i := 0; i < cutoff; i++ {
		sum += sorted[i]
	}

	// CVaR is expressed as a positive number representing expected loss.
	return -(sum / float64(cutoff)) * 100 // convert to percentage
}
