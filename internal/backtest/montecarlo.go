package backtest

import (
	"math"
	"math/rand/v2"
	"sort"
	"sync"
)

// MonteCarloResult holds the output of a Monte Carlo simulation.
type MonteCarloResult struct {
	Simulations    int     `json:"simulations"`
	OriginalSharpe float64 `json:"original_sharpe"`
	OriginalMaxDD  float64 `json:"original_max_dd"`
	PValue         float64 `json:"p_value"`         // fraction of sims with Sharpe >= original
	MedianSharpe   float64 `json:"median_sharpe"`
	MedianMaxDD    float64 `json:"median_max_dd"`
	DD95           float64 `json:"dd_95"`            // 95th percentile max drawdown
	DD99           float64 `json:"dd_99"`            // 99th percentile max drawdown
	SharpeLower95  float64 `json:"sharpe_lower_95"`  // 2.5th percentile Sharpe
	SharpeUpper95  float64 `json:"sharpe_upper_95"`  // 97.5th percentile Sharpe
}

// simResult holds the output of a single Monte Carlo simulation run.
type simResult struct {
	sharpe float64
	maxDD  float64
}

// RunMonteCarlo performs trade-resampling Monte Carlo simulation.
// It shuffles the trade PnL list N times, replays each shuffled sequence,
// and computes the distribution of outcomes.
func RunMonteCarlo(trades []Trade, originalResult *Result, nSims int, candleInterval string) *MonteCarloResult {
	if nSims <= 0 {
		nSims = 1000
	}
	if len(trades) == 0 {
		return &MonteCarloResult{
			Simulations:    nSims,
			OriginalSharpe: originalResult.SharpeRatio,
			OriginalMaxDD:  originalResult.MaxDrawdown,
		}
	}

	// Extract PnL percentages from trades.
	pnls := make([]float64, len(trades))
	for i, t := range trades {
		pnls[i] = t.PnLPct
	}

	// Run simulations using worker goroutines.
	numWorkers := 4
	results := make([]simResult, nSims)

	var wg sync.WaitGroup
	simCh := make(chan int, nSims)

	// Fill the work channel.
	for i := 0; i < nSims; i++ {
		simCh <- i
	}
	close(simCh)

	// Launch workers.
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(seed uint64) {
			defer wg.Done()
			// Each worker gets its own RNG to avoid contention.
			rng := rand.New(rand.NewPCG(seed, seed*31+7))
			localPnls := make([]float64, len(pnls))

			for simIdx := range simCh {
				// Copy and shuffle PnLs (Fisher-Yates).
				copy(localPnls, pnls)
				rng.Shuffle(len(localPnls), func(i, j int) {
					localPnls[i], localPnls[j] = localPnls[j], localPnls[i]
				})

				// Build synthetic equity curve.
				equity := 1.0
				curve := make([]float64, len(localPnls)+1)
				curve[0] = 1.0
				for k, pnl := range localPnls {
					equity *= (1 + pnl/100)
					curve[k+1] = equity
				}

				// Compute metrics on synthetic curve.
				results[simIdx] = simResult{
					sharpe: sharpeRatio(curve, candleInterval),
					maxDD:  maxDrawdown(curve),
				}
			}
		}(uint64(w) * 12345)
	}
	wg.Wait()

	// Collect sharpe and maxDD arrays.
	sharpes := make([]float64, nSims)
	maxDDs := make([]float64, nSims)
	for i, r := range results {
		sharpes[i] = r.sharpe
		maxDDs[i] = r.maxDD
	}

	// Sort for percentile calculations.
	sort.Float64s(sharpes)
	sort.Float64s(maxDDs)

	// Compute p-value: fraction of simulations with Sharpe >= original.
	pValueCount := 0
	for _, s := range sharpes {
		if s >= originalResult.SharpeRatio {
			pValueCount++
		}
	}

	return &MonteCarloResult{
		Simulations:    nSims,
		OriginalSharpe: originalResult.SharpeRatio,
		OriginalMaxDD:  originalResult.MaxDrawdown,
		PValue:         float64(pValueCount) / float64(nSims),
		MedianSharpe:   percentile(sharpes, 0.50),
		MedianMaxDD:    percentile(maxDDs, 0.50),
		DD95:           percentile(maxDDs, 0.95),
		DD99:           percentile(maxDDs, 0.99),
		SharpeLower95:  percentile(sharpes, 0.025),
		SharpeUpper95:  percentile(sharpes, 0.975),
	}
}

// percentile returns the value at the given percentile from a sorted slice.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := p * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper || upper >= len(sorted) {
		return sorted[lower]
	}
	// Linear interpolation.
	frac := idx - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}
