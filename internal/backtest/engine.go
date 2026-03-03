package backtest

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/nickai/cli/internal/indicators"
	"github.com/nickai/cli/internal/logging"
	"github.com/nickai/cli/internal/market"
)

// Strategy describes a backtesting strategy.
type Strategy struct {
	Name          string      `json:"name,omitempty"`
	Symbol        string      `json:"symbol"`
	EntryRules    []Condition `json:"entry_conditions"`
	ExitRules     []Condition `json:"exit_conditions"`
	StopLossPct   float64     `json:"stop_loss_pct,omitempty"`
	TakeProfitPct float64     `json:"take_profit_pct,omitempty"`
	PositionSize  float64     `json:"position_size,omitempty"`  // fraction, default 1.0
	Period        string      `json:"period,omitempty"`         // e.g. "180d", "1y"
	SlippageBps   float64     `json:"slippage_bps,omitempty"`   // slippage in basis points (e.g. 10 = 0.1%)
	CommissionBps float64     `json:"commission_bps,omitempty"` // commission in basis points per trade
}

// Condition represents a single rule for entry/exit.
type Condition struct {
	Indicator   string  `json:"indicator"`              // rsi, macd, macd_histogram, macd_signal, bollinger_upper, bollinger_lower, sma20, sma50, ema12, ema26, price, fear_greed
	Operator    string  `json:"operator"`               // <, >, crosses_above, crosses_below
	Value       float64 `json:"value"`                  // static threshold (used when CompareWith is empty)
	CompareWith string  `json:"compare_with,omitempty"` // compare against another indicator instead of Value
}

// Result holds the output of a backtest run.
type Result struct {
	Strategy     string    `json:"strategy"`
	Symbol       string    `json:"symbol"`
	Period       string    `json:"period"`
	Trades       []Trade   `json:"trades"`
	TotalTrades  int       `json:"total_trades"`
	WinRate      float64   `json:"win_rate"`
	TotalReturn  float64   `json:"total_return"`
	SharpeRatio  float64   `json:"sharpe_ratio"`
	MaxDrawdown  float64   `json:"max_drawdown"`
	ProfitFactor float64   `json:"profit_factor"`
	BestTrade    float64   `json:"best_trade"`
	WorstTrade   float64   `json:"worst_trade"`
	EquityCurve  []float64 `json:"equity_curve"`
}

// Trade records a single entry/exit pair.
type Trade struct {
	EntryTime  time.Time `json:"entry_time"`
	ExitTime   time.Time `json:"exit_time"`
	EntryPrice float64   `json:"entry_price"`
	ExitPrice  float64   `json:"exit_price"`
	PnLPct     float64   `json:"pnl_pct"`
	Reason     string    `json:"reason"`
}

// indicatorSnapshot holds pre-computed indicator values at a given candle.
type indicatorSnapshot struct {
	RSI            float64
	MACD           float64
	MACDSignal     float64
	MACDHistogram  float64
	BollingerUpper float64
	BollingerLower float64
	SMA20          float64
	SMA50          float64
	EMA12          float64
	EMA26          float64
	Price          float64
	FearGreed      float64
}

// ParsePeriod converts a period string like "180d", "1y", "6m" to days.
func ParsePeriod(s string) (int, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 180, nil // default
	}

	if strings.HasSuffix(s, "y") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "y"))
		if err != nil {
			return 0, fmt.Errorf("invalid period: %s", s)
		}
		return n * 365, nil
	}
	if strings.HasSuffix(s, "m") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "m"))
		if err != nil {
			return 0, fmt.Errorf("invalid period: %s", s)
		}
		return n * 30, nil
	}
	if strings.HasSuffix(s, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid period: %s", s)
		}
		return n, nil
	}

	// Try bare number as days.
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid period: %s (use e.g. 180d, 6m, 1y)", s)
	}
	return n, nil
}

// Run executes a backtest strategy against historical data.
func Run(strat Strategy) (*Result, error) {
	if strat.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

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
	if limit > 1000 {
		limit = 1000
	}

	candles, err := market.FetchKlines(strat.Symbol, interval, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market data: %w", err)
	}
	if len(candles) < 50 {
		return nil, fmt.Errorf("insufficient data: got %d candles, need at least 50 for indicator warmup", len(candles))
	}

	closePrices := market.ClosePrices(candles)

	// Fetch historical Fear & Greed data if any condition uses it.
	var fgData []market.FearGreedDay
	if needsFearGreed(strat) {
		fgData, _ = market.FetchHistoricalFearGreed(days + 10)
	}

	// Pre-compute indicator snapshots.
	snapshots := make([]indicatorSnapshot, len(candles))
	for i := range candles {
		slice := closePrices[:i+1]
		snap := indicatorSnapshot{
			Price: closePrices[i],
		}
		if len(slice) > 14 {
			snap.RSI = indicators.RSI(slice, 14)
		} else {
			snap.RSI = 50
		}
		if len(slice) >= 26 {
			snap.MACD, snap.MACDSignal, snap.MACDHistogram = indicators.MACDCalc(slice)
		}
		if len(slice) >= 20 {
			snap.BollingerUpper, _, snap.BollingerLower = indicators.BollingerBands(slice, 20)
			snap.SMA20 = indicators.SMA(slice, 20)
		}
		if len(slice) >= 50 {
			snap.SMA50 = indicators.SMA(slice, 50)
		}
		if len(slice) >= 12 {
			snap.EMA12 = indicators.EMA(slice, 12)
		}
		if len(slice) >= 26 {
			snap.EMA26 = indicators.EMA(slice, 26)
		}

		// Map Fear & Greed to candle timestamp.
		if len(fgData) > 0 {
			snap.FearGreed = findFearGreed(fgData, candles[i].OpenTime)
		}

		snapshots[i] = snap
	}

	if strat.PositionSize <= 0 || strat.PositionSize > 1 {
		strat.PositionSize = 1.0
	}

	// Slippage and commission as multipliers (basis points → fraction).
	slippage := strat.SlippageBps / 10000.0
	commission := strat.CommissionBps / 10000.0

	// Walk candles and simulate.
	var trades []Trade
	inPosition := false
	var entryPrice float64
	var entryTime time.Time
	equity := 1.0
	equityCurve := []float64{1.0}

	// Start from candle 50 (warmup period).
	for i := 50; i < len(candles); i++ {
		price := closePrices[i]

		if !inPosition {
			// Check entry conditions.
			if allConditionsMet(strat.EntryRules, snapshots, i) {
				inPosition = true
				// Simulate slippage: buying at a slightly higher price.
				entryPrice = price * (1 + slippage)
				entryTime = candles[i].OpenTime
				// Commission on entry reduces equity.
				equity *= (1 - commission*strat.PositionSize)
			}
		} else {
			// Check exit conditions.
			// Simulate slippage: selling at a slightly lower price.
			exitPrice := price * (1 - slippage)
			pnlPct := (exitPrice - entryPrice) / entryPrice * 100

			reason := ""
			shouldExit := false

			// Stop loss.
			if strat.StopLossPct > 0 && pnlPct <= -strat.StopLossPct {
				shouldExit = true
				reason = "stop_loss"
			}
			// Take profit.
			if !shouldExit && strat.TakeProfitPct > 0 && pnlPct >= strat.TakeProfitPct {
				shouldExit = true
				reason = "take_profit"
			}
			// Exit rules.
			if !shouldExit && len(strat.ExitRules) > 0 && allConditionsMet(strat.ExitRules, snapshots, i) {
				shouldExit = true
				reason = "exit_signal"
			}

			if shouldExit {
				actualPnl := pnlPct * strat.PositionSize / 100
				equity *= (1 + actualPnl)
				// Commission on exit.
				equity *= (1 - commission*strat.PositionSize)
				trades = append(trades, Trade{
					EntryTime:  entryTime,
					ExitTime:   candles[i].OpenTime,
					EntryPrice: entryPrice,
					ExitPrice:  exitPrice,
					PnLPct:     pnlPct,
					Reason:     reason,
				})
				inPosition = false
			}
		}
		equityCurve = append(equityCurve, equity)
	}

	// Close open position at end of period.
	if inPosition && len(candles) > 0 {
		lastPrice := closePrices[len(closePrices)-1]
		exitPrice := lastPrice * (1 - slippage)
		pnlPct := (exitPrice - entryPrice) / entryPrice * 100
		actualPnl := pnlPct * strat.PositionSize / 100
		equity *= (1 + actualPnl)
		equity *= (1 - commission*strat.PositionSize)
		trades = append(trades, Trade{
			EntryTime:  entryTime,
			ExitTime:   candles[len(candles)-1].OpenTime,
			EntryPrice: entryPrice,
			ExitPrice:  exitPrice,
			PnLPct:     pnlPct,
			Reason:     "period_end",
		})
		equityCurve = append(equityCurve, equity)
	}

	result := computeMetrics(strat, trades, equityCurve, interval)
	return result, nil
}

// allConditionsMet checks if all conditions are met at candle index i.
func allConditionsMet(conditions []Condition, snapshots []indicatorSnapshot, i int) bool {
	for _, cond := range conditions {
		if !evalCondition(cond, snapshots, i) {
			return false
		}
	}
	return true
}

// evalCondition evaluates a single condition at candle index i.
// When CompareWith is set, the condition compares two indicators instead of
// comparing an indicator against a static Value.
func evalCondition(cond Condition, snapshots []indicatorSnapshot, i int) bool {
	current := getIndicatorValue(cond.Indicator, snapshots[i])

	// Resolve the comparison target: another indicator or a static value.
	target := cond.Value
	if cond.CompareWith != "" {
		target = getIndicatorValue(cond.CompareWith, snapshots[i])
	}

	switch cond.Operator {
	case "<":
		return current < target
	case ">":
		return current > target
	case "crosses_above":
		if i == 0 {
			return false
		}
		prev := getIndicatorValue(cond.Indicator, snapshots[i-1])
		prevTarget := target
		if cond.CompareWith != "" {
			prevTarget = getIndicatorValue(cond.CompareWith, snapshots[i-1])
		}
		return prev <= prevTarget && current > target
	case "crosses_below":
		if i == 0 {
			return false
		}
		prev := getIndicatorValue(cond.Indicator, snapshots[i-1])
		prevTarget := target
		if cond.CompareWith != "" {
			prevTarget = getIndicatorValue(cond.CompareWith, snapshots[i-1])
		}
		return prev >= prevTarget && current < target
	default:
		logging.Debug("backtest unknown operator", "operator", cond.Operator, "indicator", cond.Indicator)
		return false
	}
}

// getIndicatorValue returns the value of an indicator from a snapshot.
func getIndicatorValue(indicator string, snap indicatorSnapshot) float64 {
	switch strings.ToLower(indicator) {
	case "rsi":
		return snap.RSI
	case "macd":
		return snap.MACD
	case "macd_histogram":
		return snap.MACDHistogram
	case "macd_signal":
		return snap.MACDSignal
	case "bollinger_upper":
		return snap.BollingerUpper
	case "bollinger_lower":
		return snap.BollingerLower
	case "sma20":
		return snap.SMA20
	case "sma50":
		return snap.SMA50
	case "ema12":
		return snap.EMA12
	case "ema26":
		return snap.EMA26
	case "price":
		return snap.Price
	case "fear_greed":
		return snap.FearGreed
	default:
		logging.Debug("backtest unknown indicator", "indicator", indicator)
		return 0
	}
}

// computeMetrics calculates backtest result metrics.
func computeMetrics(strat Strategy, trades []Trade, equityCurve []float64, interval string) *Result {
	result := &Result{
		Strategy:    strat.Name,
		Symbol:      strat.Symbol,
		Period:      strat.Period,
		Trades:      trades,
		TotalTrades: len(trades),
		EquityCurve: equityCurve,
	}

	if len(trades) == 0 {
		return result
	}

	wins := 0
	totalProfit := 0.0
	totalLoss := 0.0
	best := trades[0].PnLPct
	worst := trades[0].PnLPct

	for _, t := range trades {
		if t.PnLPct > 0 {
			wins++
			totalProfit += t.PnLPct
		} else {
			totalLoss += math.Abs(t.PnLPct)
		}
		if t.PnLPct > best {
			best = t.PnLPct
		}
		if t.PnLPct < worst {
			worst = t.PnLPct
		}
	}

	result.WinRate = float64(wins) / float64(len(trades)) * 100
	result.BestTrade = best
	result.WorstTrade = worst
	result.TotalReturn = (equityCurve[len(equityCurve)-1] - 1) * 100

	// Profit factor.
	if totalLoss > 0 {
		result.ProfitFactor = totalProfit / totalLoss
	} else if totalProfit > 0 {
		result.ProfitFactor = math.Inf(1)
	}

	// Max drawdown from equity curve.
	result.MaxDrawdown = maxDrawdown(equityCurve)

	// Sharpe ratio (annualized based on candle interval).
	result.SharpeRatio = sharpeRatio(equityCurve, interval)

	return result
}

// maxDrawdown calculates the maximum peak-to-trough decline in the equity curve.
func maxDrawdown(curve []float64) float64 {
	if len(curve) == 0 {
		return 0
	}
	peak := curve[0]
	maxDD := 0.0
	for _, v := range curve {
		if v > peak {
			peak = v
		}
		dd := (peak - v) / peak * 100
		if dd > maxDD {
			maxDD = dd
		}
	}
	return maxDD
}

// sharpeRatio calculates the annualized Sharpe ratio from an equity curve.
// candleInterval is the candle size ("1d", "4h", "1h") for correct annualization.
func sharpeRatio(curve []float64, candleInterval string) float64 {
	if len(curve) < 2 {
		return 0
	}

	returns := make([]float64, len(curve)-1)
	for i := 1; i < len(curve); i++ {
		if curve[i-1] != 0 {
			returns[i-1] = (curve[i] - curve[i-1]) / curve[i-1]
		}
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	// Use sample variance (N-1) for unbiased estimate.
	if len(returns) > 1 {
		variance /= float64(len(returns) - 1)
	}
	stddev := math.Sqrt(variance)

	if stddev == 0 {
		return 0
	}

	// Candles per year depends on interval (crypto trades 365 days).
	periodsPerYear := 365.0 // daily
	switch candleInterval {
	case "1h":
		periodsPerYear = 365 * 24
	case "4h":
		periodsPerYear = 365 * 6
	}

	return (mean / stddev) * math.Sqrt(periodsPerYear)
}

// needsFearGreed checks if any condition references the fear_greed indicator.
func needsFearGreed(strat Strategy) bool {
	for _, c := range strat.EntryRules {
		if strings.ToLower(c.Indicator) == "fear_greed" || strings.ToLower(c.CompareWith) == "fear_greed" {
			return true
		}
	}
	for _, c := range strat.ExitRules {
		if strings.ToLower(c.Indicator) == "fear_greed" || strings.ToLower(c.CompareWith) == "fear_greed" {
			return true
		}
	}
	return false
}

// findFearGreed maps a candle timestamp to the nearest Fear & Greed day value.
func findFearGreed(fgData []market.FearGreedDay, candleTime time.Time) float64 {
	// Data is sorted newest-first. Find the closest day.
	candleDay := candleTime.Truncate(24 * time.Hour)
	for _, fg := range fgData {
		fgDay := fg.Timestamp.Truncate(24 * time.Hour)
		if !fgDay.After(candleDay) {
			return float64(fg.Value)
		}
	}
	if len(fgData) > 0 {
		return float64(fgData[len(fgData)-1].Value)
	}
	return 50 // neutral default
}
