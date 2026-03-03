package backtest

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nickai/cli/internal/market"
)

// ParamRange defines a range of values for a parameter.
type ParamRange struct {
	Name   string    // "stop_loss_pct", "take_profit_pct", "rsi_entry", "rsi_exit", etc.
	Values []float64 // explicit list of values to try
}

// OptimizeConfig configures grid search optimization.
type OptimizeConfig struct {
	Params    []ParamRange
	Metric    string // "sharpe", "sortino", "total_return", "calmar" — what to maximize
	MaxCombos int    // safety limit (default 1000)
}

// OptimizeResult holds grid search results.
type OptimizeResult struct {
	BestParams  map[string]float64
	BestMetric  float64
	BestResult  *Result
	TotalCombos int
	TopN        []OptimizeEntry // top 10 results
	Duration    time.Duration
}

// OptimizeEntry is a single parameter combination result.
type OptimizeEntry struct {
	Params map[string]float64
	Metric float64
	Result *Result
}

// optimizeDefaults fills in zero-value config fields with defaults.
func optimizeDefaults(cfg OptimizeConfig) OptimizeConfig {
	if cfg.MaxCombos <= 0 {
		cfg.MaxCombos = 1000
	}
	if cfg.Metric == "" {
		cfg.Metric = "sharpe"
	}
	return cfg
}

// cartesianProduct generates all combinations of parameter values.
// Returns a slice of maps, where each map is one parameter combination.
func cartesianProduct(params []ParamRange) []map[string]float64 {
	if len(params) == 0 {
		return nil
	}

	// Calculate total combinations.
	total := 1
	for _, p := range params {
		if len(p.Values) == 0 {
			return nil
		}
		total *= len(p.Values)
	}

	combos := make([]map[string]float64, 0, total)
	indices := make([]int, len(params))

	for {
		// Build current combination.
		combo := make(map[string]float64, len(params))
		for i, p := range params {
			combo[p.Name] = p.Values[indices[i]]
		}
		combos = append(combos, combo)

		// Increment indices (odometer-style).
		carry := true
		for i := len(indices) - 1; i >= 0 && carry; i-- {
			indices[i]++
			if indices[i] < len(params[i].Values) {
				carry = false
			} else {
				indices[i] = 0
			}
		}
		if carry {
			break // all combinations generated
		}
	}

	return combos
}

// applyParams modifies a strategy based on a parameter map.
// Known parameter names:
//   - "stop_loss_pct"    → strat.StopLossPct
//   - "take_profit_pct"  → strat.TakeProfitPct
//   - "position_size"    → strat.PositionSize
//   - "slippage_bps"     → strat.SlippageBps
//   - "commission_bps"   → strat.CommissionBps
//   - "rsi_entry"        → modifies Condition.Value in EntryRules where Indicator == "rsi"
//   - "rsi_exit"         → modifies Condition.Value in ExitRules where Indicator == "rsi"
//   - "macd_entry"       → modifies Condition.Value in EntryRules where Indicator == "macd_histogram"
//   - "macd_exit"        → modifies Condition.Value in ExitRules where Indicator == "macd_histogram"
func applyParams(strat Strategy, params map[string]float64) Strategy {
	// Deep copy entry and exit rules to avoid mutating the original.
	if len(strat.EntryRules) > 0 {
		rules := make([]Condition, len(strat.EntryRules))
		copy(rules, strat.EntryRules)
		strat.EntryRules = rules
	}
	if len(strat.ExitRules) > 0 {
		rules := make([]Condition, len(strat.ExitRules))
		copy(rules, strat.ExitRules)
		strat.ExitRules = rules
	}

	for name, value := range params {
		switch strings.ToLower(name) {
		case "stop_loss_pct":
			strat.StopLossPct = value
		case "take_profit_pct":
			strat.TakeProfitPct = value
		case "position_size":
			strat.PositionSize = value
		case "slippage_bps":
			strat.SlippageBps = value
		case "commission_bps":
			strat.CommissionBps = value
		case "rsi_entry":
			for j := range strat.EntryRules {
				if strings.ToLower(strat.EntryRules[j].Indicator) == "rsi" {
					strat.EntryRules[j].Value = value
				}
			}
		case "rsi_exit":
			for j := range strat.ExitRules {
				if strings.ToLower(strat.ExitRules[j].Indicator) == "rsi" {
					strat.ExitRules[j].Value = value
				}
			}
		case "macd_entry":
			for j := range strat.EntryRules {
				ind := strings.ToLower(strat.EntryRules[j].Indicator)
				if ind == "macd_histogram" || ind == "macd" {
					strat.EntryRules[j].Value = value
				}
			}
		case "macd_exit":
			for j := range strat.ExitRules {
				ind := strings.ToLower(strat.ExitRules[j].Indicator)
				if ind == "macd_histogram" || ind == "macd" {
					strat.ExitRules[j].Value = value
				}
			}
		}
	}

	return strat
}

// extractMetric returns the value of the specified metric from a Result.
func extractMetric(result *Result, metric string) float64 {
	switch strings.ToLower(metric) {
	case "sharpe":
		return result.SharpeRatio
	case "sortino":
		return result.SortinoRatio
	case "total_return":
		return result.TotalReturn
	case "calmar":
		return result.CalmarRatio
	case "profit_factor":
		return result.ProfitFactor
	case "win_rate":
		return result.WinRate
	default:
		return result.SharpeRatio
	}
}

// RunGridSearch performs exhaustive grid search over parameter combinations.
// It fetches candles once, then runs the strategy with each parameter combination
// using a worker pool for parallelism.
func RunGridSearch(baseStrat Strategy, config OptimizeConfig) (*OptimizeResult, error) {
	if baseStrat.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	config = optimizeDefaults(config)

	// Fetch candles once.
	days, err := ParsePeriod(baseStrat.Period)
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

	candles, err := market.FetchKlinesPaginated(baseStrat.Symbol, interval, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market data: %w", err)
	}

	return RunGridSearchWithCandles(baseStrat, candles, interval, config)
}

// RunGridSearchWithCandles performs grid search on pre-fetched candle data.
func RunGridSearchWithCandles(baseStrat Strategy, candles []market.Candle, interval string, config OptimizeConfig) (*OptimizeResult, error) {
	config = optimizeDefaults(config)

	if len(config.Params) == 0 {
		return nil, fmt.Errorf("no parameters specified for optimization")
	}

	// Generate all parameter combinations.
	combos := cartesianProduct(config.Params)
	if len(combos) == 0 {
		return nil, fmt.Errorf("no parameter combinations generated (check that all ParamRange.Values are non-empty)")
	}
	if len(combos) > config.MaxCombos {
		return nil, fmt.Errorf("too many parameter combinations: %d exceeds limit of %d", len(combos), config.MaxCombos)
	}

	start := time.Now()

	// Fan out using worker pool.
	numWorkers := runtime.NumCPU()
	if numWorkers > len(combos) {
		numWorkers = len(combos)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	type workItem struct {
		idx    int
		params map[string]float64
	}

	workCh := make(chan workItem, len(combos))
	for i, combo := range combos {
		workCh <- workItem{idx: i, params: combo}
	}
	close(workCh)

	results := make([]OptimizeEntry, len(combos))
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range workCh {
				strat := applyParams(baseStrat, item.params)
				result, err := RunWithCandles(strat, candles, interval)

				entry := OptimizeEntry{
					Params: item.params,
				}

				if err == nil && result != nil {
					metricValue := extractMetric(result, config.Metric)
					// Filter out NaN and -Inf values.
					if math.IsNaN(metricValue) || math.IsInf(metricValue, -1) {
						metricValue = math.Inf(-1)
					}
					entry.Metric = metricValue
					entry.Result = result
				} else {
					entry.Metric = math.Inf(-1)
				}

				results[item.idx] = entry
			}
		}()
	}
	wg.Wait()

	duration := time.Since(start)

	// Sort by metric (descending). Handle +Inf correctly.
	sort.Slice(results, func(i, j int) bool {
		mi, mj := results[i].Metric, results[j].Metric
		// Push -Inf to the end.
		if math.IsInf(mi, -1) {
			return false
		}
		if math.IsInf(mj, -1) {
			return true
		}
		return mi > mj
	})

	// Collect top N (up to 10).
	topN := 10
	if topN > len(results) {
		topN = len(results)
	}
	top := make([]OptimizeEntry, 0, topN)
	for i := 0; i < topN; i++ {
		if !math.IsInf(results[i].Metric, -1) {
			top = append(top, results[i])
		}
	}

	bestParams := make(map[string]float64)
	var bestMetric float64
	var bestResult *Result
	if len(top) > 0 {
		bestParams = top[0].Params
		bestMetric = top[0].Metric
		bestResult = top[0].Result
	}

	return &OptimizeResult{
		BestParams:  bestParams,
		BestMetric:  bestMetric,
		BestResult:  bestResult,
		TotalCombos: len(combos),
		TopN:        top,
		Duration:    duration,
	}, nil
}
