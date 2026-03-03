package backtest

import (
	"fmt"
	"math"
	"strings"
)

// analysisSystemPrompt is prepended to the formatted result when sending to the AI agent.
const AnalysisSystemPrompt = `You are a quantitative trading analyst reviewing backtest results. Analyze the following strategy results critically. Be specific about statistical significance, overfitting risks, and actionable improvements. Do not be overly optimistic — flag real concerns.

Focus on:
1. Statistical significance — is the Sharpe/Sortino meaningful given the trade count?
2. Overfitting risk — are there too few trades? Would results hold out-of-sample?
3. Benchmark comparison — does this beat a simple buy-and-hold (HODL) or DCA approach?
4. Risk assessment — interpret max drawdown, VaR, tail ratio, and worst-case scenarios
5. Specific improvements — parameter adjustments, additional filters, position sizing changes
6. Regime risk — when and why would this strategy likely fail?

`

// FormatResultForAnalysis converts a Result to a structured prompt for Claude analysis.
// The output is human-readable text (not raw JSON) designed for LLM comprehension.
func FormatResultForAnalysis(result *Result) string {
	if result == nil {
		return "No backtest result available."
	}

	var b strings.Builder

	// --- Header ---
	b.WriteString("=== BACKTEST RESULTS FOR ANALYSIS ===\n\n")

	// --- Strategy Overview ---
	b.WriteString("## Strategy Overview\n")
	if result.Strategy != "" {
		b.WriteString(fmt.Sprintf("Strategy: %s\n", result.Strategy))
	}
	b.WriteString(fmt.Sprintf("Symbol: %s\n", result.Symbol))
	if result.Period != "" {
		b.WriteString(fmt.Sprintf("Period: %s\n", result.Period))
	}
	b.WriteString("\n")

	// --- Performance Metrics ---
	b.WriteString("## Performance Metrics\n")
	b.WriteString(fmt.Sprintf("Total Return: %.2f%%\n", result.TotalReturn))
	b.WriteString(fmt.Sprintf("Sharpe Ratio: %.2f\n", result.SharpeRatio))
	b.WriteString(fmt.Sprintf("Max Drawdown: %.2f%%\n", result.MaxDrawdown))
	b.WriteString(fmt.Sprintf("Profit Factor: %s\n", formatProfitFactor(result.ProfitFactor)))
	b.WriteString("\n")

	// --- Trade Statistics ---
	b.WriteString("## Trade Statistics\n")
	b.WriteString(fmt.Sprintf("Total Trades: %d\n", result.TotalTrades))
	b.WriteString(fmt.Sprintf("Win Rate: %.1f%%\n", result.WinRate))
	b.WriteString(fmt.Sprintf("Best Trade: %+.2f%%\n", result.BestTrade))
	b.WriteString(fmt.Sprintf("Worst Trade: %+.2f%%\n", result.WorstTrade))

	if result.TotalTrades > 0 {
		avgPnl := averagePnL(result.Trades)
		b.WriteString(fmt.Sprintf("Average Trade: %+.2f%%\n", avgPnl))

		wins, losses := countWinsLosses(result.Trades)
		b.WriteString(fmt.Sprintf("Winning Trades: %d\n", wins))
		b.WriteString(fmt.Sprintf("Losing Trades: %d\n", losses))
	}
	b.WriteString("\n")

	// --- Exit Reason Breakdown ---
	if result.TotalTrades > 0 {
		b.WriteString("## Exit Reason Breakdown\n")
		reasons := exitReasonBreakdown(result.Trades)
		for reason, count := range reasons {
			b.WriteString(fmt.Sprintf("  %s: %d (%.0f%%)\n", reason, count,
				float64(count)/float64(result.TotalTrades)*100))
		}
		b.WriteString("\n")
	}

	// --- Equity Curve Summary ---
	if len(result.EquityCurve) > 1 {
		b.WriteString("## Equity Curve Summary\n")
		b.WriteString(fmt.Sprintf("Starting Equity: %.4f\n", result.EquityCurve[0]))
		b.WriteString(fmt.Sprintf("Ending Equity: %.4f\n", result.EquityCurve[len(result.EquityCurve)-1]))
		b.WriteString(fmt.Sprintf("Data Points: %d\n", len(result.EquityCurve)))

		// Peak and trough.
		peak, trough := equityPeakTrough(result.EquityCurve)
		b.WriteString(fmt.Sprintf("Peak Equity: %.4f\n", peak))
		b.WriteString(fmt.Sprintf("Trough Equity: %.4f\n", trough))
		b.WriteString("\n")
	}

	// --- Recent Trades (last 10) ---
	if len(result.Trades) > 0 {
		b.WriteString("## Recent Trades (last 10)\n")
		start := 0
		if len(result.Trades) > 10 {
			start = len(result.Trades) - 10
		}
		for i := start; i < len(result.Trades); i++ {
			t := result.Trades[i]
			b.WriteString(fmt.Sprintf("  #%d: Entry $%.2f (%s) -> Exit $%.2f (%s) | PnL: %+.2f%% | Reason: %s\n",
				i+1,
				t.EntryPrice, t.EntryTime.Format("2006-01-02"),
				t.ExitPrice, t.ExitTime.Format("2006-01-02"),
				t.PnLPct,
				t.Reason,
			))
		}
		b.WriteString("\n")
	}

	// --- Analysis Questions ---
	b.WriteString("## Questions for Analysis\n")
	b.WriteString("Please address each of the following:\n\n")
	b.WriteString(fmt.Sprintf(
		"1. **Statistical Significance**: With %d trades and a Sharpe ratio of %.2f, "+
			"is this result statistically meaningful? What is the minimum number of trades "+
			"needed for confidence in these metrics?\n\n", result.TotalTrades, result.SharpeRatio))
	b.WriteString(fmt.Sprintf(
		"2. **Overfitting Risk**: Given the %s period and %d trades, how likely is this "+
			"strategy to be overfit to historical data? What would you expect out-of-sample?\n\n",
		result.Period, result.TotalTrades))
	b.WriteString(fmt.Sprintf(
		"3. **Benchmark Comparison**: The strategy returned %.2f%% over this period. "+
			"How does this likely compare to a simple buy-and-hold or DCA approach for %s?\n\n",
		result.TotalReturn, result.Symbol))
	b.WriteString(fmt.Sprintf(
		"4. **Risk Assessment**: Max drawdown was %.2f%% and worst single trade was %+.2f%%. "+
			"Evaluate whether this risk level is acceptable and what it implies about tail risk.\n\n",
		result.MaxDrawdown, result.WorstTrade))
	b.WriteString("5. **Improvement Suggestions**: What specific parameter adjustments, additional " +
		"filters, or position sizing changes would you recommend?\n\n")
	b.WriteString(fmt.Sprintf(
		"6. **Regime Risk**: Under what market conditions (trending, ranging, high volatility, "+
			"crash) would this %s strategy likely fail? What regime detection could be added?\n",
		result.Strategy))

	return b.String()
}

// --- helpers ---

func formatProfitFactor(pf float64) string {
	if math.IsInf(pf, 1) {
		return "Inf (no losing trades)"
	}
	return fmt.Sprintf("%.2f", pf)
}

func averagePnL(trades []Trade) float64 {
	if len(trades) == 0 {
		return 0
	}
	sum := 0.0
	for _, t := range trades {
		sum += t.PnLPct
	}
	return sum / float64(len(trades))
}

func countWinsLosses(trades []Trade) (int, int) {
	wins, losses := 0, 0
	for _, t := range trades {
		if t.PnLPct > 0 {
			wins++
		} else {
			losses++
		}
	}
	return wins, losses
}

func exitReasonBreakdown(trades []Trade) map[string]int {
	reasons := make(map[string]int)
	for _, t := range trades {
		r := t.Reason
		if r == "" {
			r = "unknown"
		}
		reasons[r]++
	}
	return reasons
}

func equityPeakTrough(curve []float64) (float64, float64) {
	if len(curve) == 0 {
		return 0, 0
	}
	peak := curve[0]
	trough := curve[0]
	for _, v := range curve {
		if v > peak {
			peak = v
		}
		if v < trough {
			trough = v
		}
	}
	return peak, trough
}
