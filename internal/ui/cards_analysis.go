package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/ai"
	"github.com/nickai/cli/internal/analytics"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/backtest"
	"github.com/nickai/cli/internal/indicators"
	"github.com/nickai/cli/internal/journal"
	"github.com/nickai/cli/internal/market"
)

// --- Analytics rendering ---

// RenderAnalytics displays portfolio analytics with metrics and allocation chart.
func RenderAnalytics(client *api.PapernickClient, width int) string {
	cardWidth := min(width-4, 64)

	// Load journal entries.
	entries, _ := journal.All()

	// Load portfolio.
	portfolio, err := client.GetPortfolio()
	if err != nil {
		return ErrorStyle.Render("  Failed to load portfolio: ") + err.Error()
	}

	// Build price map.
	symbolSet := make(map[string]bool)
	for _, e := range entries {
		base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(e.Symbol, "USDT"), "USDC"), "USD")
		symbolSet[base] = true
	}
	for _, a := range portfolio.Assets {
		base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(a.Symbol, "USDT"), "USDC"), "USD")
		symbolSet[base] = true
	}
	symbols := make([]string, 0, len(symbolSet))
	for s := range symbolSet {
		symbols = append(symbols, s)
	}
	priceMap := make(map[string]float64)
	if len(symbols) > 0 {
		if prices, err := client.GetPrices(symbols); err == nil {
			for _, p := range prices {
				priceMap[p.Symbol] = p.Price
				base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(p.Symbol, "USDT"), "USDC"), "USD")
				priceMap[base] = p.Price
			}
		}
	}

	metrics := analytics.Calculate(entries, priceMap)
	allocs := analytics.CalcAllocation(portfolio)

	// Build metrics display.
	var lines []string
	lines = append(lines, "")

	// Key metrics row.
	sharpeColor := ColorPrimary
	if metrics.SharpeRatio < 0 {
		sharpeColor = ColorError
	}
	lines = append(lines,
		DimStyle.Render("Sharpe Ratio:   ")+lipgloss.NewStyle().Foreground(sharpeColor).Bold(true).Render(fmt.Sprintf("%.2f", metrics.SharpeRatio))+
			DimStyle.Render("        Win Rate:      ")+lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(fmt.Sprintf("%.1f%%", metrics.WinRate)))
	lines = append(lines,
		DimStyle.Render("Max Drawdown:   ")+ErrorStyle.Render(fmt.Sprintf("%.1f%%", metrics.MaxDrawdownPct))+
			DimStyle.Render("        Profit Factor: ")+BrandStyle.Render(fmt.Sprintf("%.2f", metrics.ProfitFactor)))
	lines = append(lines,
		DimStyle.Render("Total Trades:   ")+fmt.Sprintf("%d", metrics.TotalTrades)+
			DimStyle.Render("            W/L:           ")+fmt.Sprintf("%d/%d", metrics.WinCount, metrics.LossCount))

	pnlStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
	if metrics.TotalPnL < 0 {
		pnlStyle = lipgloss.NewStyle().Foreground(ColorError)
	}
	lines = append(lines,
		DimStyle.Render("Total P&L:      ")+pnlStyle.Bold(true).Render(fmt.Sprintf("$%.2f", metrics.TotalPnL)))

	if metrics.WinCount > 0 || metrics.LossCount > 0 {
		lines = append(lines,
			DimStyle.Render("Avg Win:        ")+BrandStyle.Render(fmt.Sprintf("$%.2f", metrics.AvgWin))+
				DimStyle.Render("       Avg Loss:       ")+ErrorStyle.Render(fmt.Sprintf("$%.2f", metrics.AvgLoss)))
		lines = append(lines,
			DimStyle.Render("Best Trade:     ")+BrandStyle.Render(fmt.Sprintf("$%.2f", metrics.BestTrade))+
				DimStyle.Render("     Worst Trade:    ")+ErrorStyle.Render(fmt.Sprintf("$%.2f", metrics.WorstTrade)))
	}

	// Allocation bar chart.
	if len(allocs) > 0 {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("Allocation"))
		barWidth := cardWidth - 24
		if barWidth < 10 {
			barWidth = 10
		}
		for _, a := range allocs {
			filled := int(a.Percent / 100 * float64(barWidth))
			if filled < 0 {
				filled = 0
			}
			if filled > barWidth {
				filled = barWidth
			}
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			lines = append(lines,
				"  "+padRight(a.Symbol, 6)+
					lipgloss.NewStyle().Foreground(ColorPrimary).Render(bar)+
					DimStyle.Render(fmt.Sprintf(" %5.1f%% $%.0f", a.Percent, a.Value)))
		}
	}

	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Width(cardWidth).
		Render(content)

	return SecondaryStyle.Render("  Portfolio Analytics") + "\n" + box
}

// --- Market Analysis rendering ---

// RenderAnalysis displays technical analysis for a symbol.
func RenderAnalysis(client *api.PapernickClient, symbol string, width int) string {
	cardWidth := min(width-4, 64)
	symbol = strings.ToUpper(symbol)

	// Fetch current price.
	prices, err := client.GetPrices([]string{symbol})
	if err != nil || len(prices) == 0 {
		return ErrorStyle.Render("  Failed to fetch price for ") + symbol
	}
	currentPrice := prices[0].Price

	// Fetch real price history from Binance, fallback to synthetic.
	var history []float64
	if candles, err := market.FetchKlines(symbol, "1d", 50); err == nil && len(candles) > 0 {
		history = market.ClosePrices(candles)
	} else {
		history = generateSparklineData(currentPrice, 50)
	}

	// Fear & Greed.
	fg, fgLabel, _ := indicators.FetchFearGreed()

	a := indicators.Analyze(symbol, currentPrice, history, fg, fgLabel)

	// Build display.
	var lines []string
	lines = append(lines, "")
	lines = append(lines,
		BrandStyle.Render(symbol)+"  "+
			lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(formatPrice(currentPrice)))
	lines = append(lines, "")

	// Indicator badges.
	badge := func(label, value, signal string) string {
		signalColor := ColorDim
		switch signal {
		case "overbought", "bearish", "above":
			signalColor = ColorError
		case "oversold", "bullish", "below":
			signalColor = ColorPrimary
		}
		return DimStyle.Render(padRight(label, 14)) +
			lipgloss.NewStyle().Foreground(ColorWhite).Render(value) +
			"  " + lipgloss.NewStyle().Foreground(signalColor).Bold(true).Render(signal)
	}

	lines = append(lines, badge("RSI (14)", fmt.Sprintf("%.1f", a.RSI), a.RSISignal))
	lines = append(lines, badge("MACD", fmt.Sprintf("%.2f", a.MACD), a.MACDTrend))
	lines = append(lines, badge("Bollinger", fmt.Sprintf("%.0f / %.0f", a.BollingerLower, a.BollingerUpper), a.BollingerPos))
	lines = append(lines, badge("SMA 20", formatPrice(a.SMA20), ""))
	if a.SMA50 > 0 {
		lines = append(lines, badge("SMA 50", formatPrice(a.SMA50), ""))
	}
	lines = append(lines, badge("Trend", a.Trend, a.Trend))

	// Fear & Greed.
	if a.FearGreedLabel != "" {
		fgColor := ColorDim
		switch {
		case a.FearGreed <= 25:
			fgColor = ColorError
		case a.FearGreed >= 75:
			fgColor = ColorPrimary
		case a.FearGreed >= 50:
			fgColor = ColorWarning
		}
		lines = append(lines, "")
		lines = append(lines, DimStyle.Render("Fear & Greed:   ")+
			lipgloss.NewStyle().Foreground(fgColor).Bold(true).Render(
				fmt.Sprintf("%d/100 (%s)", a.FearGreed, a.FearGreedLabel)))
	}

	// Sparkline.
	lines = append(lines, "")
	sparkWidth := cardWidth - 6
	if sparkWidth > 40 {
		sparkWidth = 40
	}
	lines = append(lines, "  "+renderSparkline(history, sparkWidth))

	// Summary.
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render(a.Summary))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Width(cardWidth).
		Render(content)

	return SecondaryStyle.Render("  Market Analysis") + "\n" + box
}

// --- Backtest rendering ---

// RenderBacktestCard renders backtest results as a styled card.
func RenderBacktestCard(result *backtest.Result) string {
	if result == nil {
		return ErrorStyle.Render("  No backtest results.")
	}

	cardWidth := 64

	var lines []string
	lines = append(lines, "")

	// Header.
	name := result.Strategy
	if name == "" {
		name = "Custom Strategy"
	}
	lines = append(lines, BrandStyle.Render(name)+"  "+
		lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(result.Symbol)+
		DimStyle.Render("  "+result.Period))
	lines = append(lines, "")

	// Metrics table.
	metricLine := func(label string, value string) string {
		return DimStyle.Render(padRight("  "+label, 22)) +
			lipgloss.NewStyle().Foreground(ColorWhite).Render(value)
	}

	returnColor := ColorPrimary
	if result.TotalReturn < 0 {
		returnColor = ColorError
	}

	lines = append(lines, metricLine("Total Trades", fmt.Sprintf("%d", result.TotalTrades)))
	lines = append(lines, metricLine("Win Rate", fmt.Sprintf("%.1f%%", result.WinRate)))
	lines = append(lines, DimStyle.Render(padRight("  Total Return", 22))+
		lipgloss.NewStyle().Foreground(returnColor).Bold(true).Render(fmt.Sprintf("%.2f%%", result.TotalReturn)))
	lines = append(lines, metricLine("Sharpe Ratio", fmt.Sprintf("%.2f", result.SharpeRatio)))
	lines = append(lines, metricLine("Sortino Ratio", fmt.Sprintf("%.2f", result.SortinoRatio)))
	lines = append(lines, metricLine("Max Drawdown", fmt.Sprintf("%.2f%%", result.MaxDrawdown)))
	if !math.IsInf(result.ProfitFactor, 0) {
		lines = append(lines, metricLine("Profit Factor", fmt.Sprintf("%.2f", result.ProfitFactor)))
	}
	if !math.IsInf(result.CalmarRatio, 0) && result.CalmarRatio != 0 {
		lines = append(lines, metricLine("Calmar Ratio", fmt.Sprintf("%.2f", result.CalmarRatio)))
	}
	if result.Expectancy != 0 {
		lines = append(lines, metricLine("Expectancy", fmt.Sprintf("%.2f%%", result.Expectancy)))
	}
	if result.VaR95 != 0 {
		lines = append(lines, metricLine("VaR (95%)", fmt.Sprintf("%.2f%%", result.VaR95)))
	}
	if result.TimeInMarketPct != 0 {
		lines = append(lines, metricLine("Time in Market", fmt.Sprintf("%.1f%%", result.TimeInMarketPct)))
	}

	// Benchmarks.
	if result.HODLReturn != 0 || result.DCAReturn != 0 {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("  Benchmarks"))
		hodlColor := ColorPrimary
		if result.HODLReturn < 0 {
			hodlColor = ColorError
		}
		lines = append(lines, DimStyle.Render(padRight("  HODL", 22))+
			lipgloss.NewStyle().Foreground(hodlColor).Render(fmt.Sprintf("%.2f%%", result.HODLReturn)))
		if result.DCAReturn != 0 {
			dcaColor := ColorPrimary
			if result.DCAReturn < 0 {
				dcaColor = ColorError
			}
			lines = append(lines, DimStyle.Render(padRight("  DCA (weekly)", 22))+
				lipgloss.NewStyle().Foreground(dcaColor).Render(fmt.Sprintf("%.2f%%", result.DCAReturn)))
		}
	}

	// Best/Worst trade.
	if result.TotalTrades > 0 {
		lines = append(lines, "")
		bestColor := ColorPrimary
		if result.BestTrade < 0 {
			bestColor = ColorError
		}
		worstColor := ColorError
		if result.WorstTrade > 0 {
			worstColor = ColorPrimary
		}
		lines = append(lines, DimStyle.Render(padRight("  Best Trade", 22))+
			lipgloss.NewStyle().Foreground(bestColor).Render(fmt.Sprintf("%+.2f%%", result.BestTrade)))
		lines = append(lines, DimStyle.Render(padRight("  Worst Trade", 22))+
			lipgloss.NewStyle().Foreground(worstColor).Render(fmt.Sprintf("%+.2f%%", result.WorstTrade)))
	}

	// Monte Carlo confidence intervals.
	if result.MonteCarlo != nil {
		mc := result.MonteCarlo
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("  Monte Carlo (1000 sims)"))
		pvalColor := ColorPrimary
		if mc.PValue > 0.5 {
			pvalColor = ColorWarning
		}
		if mc.PValue > 0.8 {
			pvalColor = ColorError
		}
		lines = append(lines, DimStyle.Render(padRight("  p-value", 22))+
			lipgloss.NewStyle().Foreground(pvalColor).Render(fmt.Sprintf("%.3f", mc.PValue)))
		lines = append(lines, metricLine("Sharpe 95% CI", fmt.Sprintf("[%.2f, %.2f]", mc.SharpeLower95, mc.SharpeUpper95)))
		lines = append(lines, metricLine("Median Sharpe", fmt.Sprintf("%.2f", mc.MedianSharpe)))
		lines = append(lines, metricLine("Max DD 95th pctl", fmt.Sprintf("%.2f%%", mc.DD95)))
		lines = append(lines, metricLine("Max DD 99th pctl", fmt.Sprintf("%.2f%%", mc.DD99)))
	}

	// Equity curve sparkline.
	if len(result.EquityCurve) > 2 {
		lines = append(lines, "")
		sparkWidth := cardWidth - 6
		if sparkWidth > 40 {
			sparkWidth = 40
		}
		lines = append(lines, "  "+renderSparkline(result.EquityCurve, sparkWidth))
		lines = append(lines, DimStyle.Render("  Equity curve"))
	}

	// Trade list (last 10).
	if len(result.Trades) > 0 {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("  Recent Trades"))
		showCount := len(result.Trades)
		startIdx := 0
		if showCount > 10 {
			startIdx = showCount - 10
			lines = append(lines, DimStyle.Render(fmt.Sprintf("  (showing last 10 of %d)", showCount)))
		}
		for i := startIdx; i < showCount; i++ {
			t := result.Trades[i]
			pnlColor := ColorPrimary
			if t.PnLPct < 0 {
				pnlColor = ColorError
			}
			entry := t.EntryTime.Format("Jan 02")
			exit := t.ExitTime.Format("Jan 02")
			pnl := lipgloss.NewStyle().Foreground(pnlColor).Render(fmt.Sprintf("%+.2f%%", t.PnLPct))
			reason := DimStyle.Render(t.Reason)
			lines = append(lines, fmt.Sprintf("  %s → %s  %s  %s", entry, exit, pnl, reason))
		}
	}

	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Width(cardWidth).
		Render(content)

	return SecondaryStyle.Render("  Backtest Results") + "\n" + box +
		NextSteps("/backtest activate", "/auto list")
}

// RenderBacktestHelp shows /backtest usage.
func RenderBacktestHelp() string {
	header := SectionHeader("/backtest — strategy backtesting")
	lines := []string{
		header,
		"  " + CommandStyle.Render("/backtest presets") + DimStyle.Render("                — list preset strategies"),
		"  " + CommandStyle.Render("/backtest run <preset> <symbol>") + DimStyle.Render(" — run a preset"),
		"  " + CommandStyle.Render("/backtest run momentum ETH 90d") + DimStyle.Render(" — preset with custom period"),
		"",
		DimStyle.Render("  Or describe a custom strategy:"),
		"  " + CommandStyle.Render("/backtest BTC RSI below 30 with 5% stop loss"),
		"  " + CommandStyle.Render("\"backtest buying ETH when RSI drops below 30\""),
		"",
		DimStyle.Render("  Long: rsi-reversal, macd-crossover, bollinger-bounce,"),
		DimStyle.Render("  golden-cross, momentum, fear-and-greed, dip-buyer"),
		DimStyle.Render("  Short: rsi-short, macd-short"),
		DimStyle.Render("  Research: and-tre-mom-dir, and-tre-mom, calm-trend"),
	}
	return strings.Join(lines, "\n")
}

// RenderBacktestPresets shows all available preset strategies.
func RenderBacktestPresets() string {
	presets := backtest.GetPresets()

	var lines []string
	lines = append(lines, SectionHeader("Backtest Presets"))

	for _, p := range presets {
		name := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(padRight(p.Strategy.Name, 20))

		// Truncate description to fit typical terminal width.
		desc := p.Description
		if len(desc) > 45 {
			desc = desc[:42] + "..."
		}
		lines = append(lines, "  "+name+DimStyle.Render(desc))

		// Details.
		details := "    "
		if p.Strategy.StopLossPct > 0 {
			details += DimStyle.Render(fmt.Sprintf("SL: %.0f%%", p.Strategy.StopLossPct))
		}
		if p.Strategy.TakeProfitPct > 0 {
			details += DimStyle.Render(fmt.Sprintf("  TP: %.0f%%", p.Strategy.TakeProfitPct))
		}
		lines = append(lines, details)
	}

	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("  Run: ")+
		CommandStyle.Render("/backtest run <name> <symbol> [period]"))
	lines = append(lines, DimStyle.Render("  Example: ")+
		CommandStyle.Render("/backtest run rsi-reversal BTC 90d"))

	return strings.Join(lines, "\n")
}

// RenderConsensusCard renders the multi-LLM consensus result.
func RenderConsensusCard(result *ai.ConsensusResult) string {
	header := lipgloss.NewStyle().
		Foreground(ColorPrimary).Bold(true).
		Render(fmt.Sprintf("  Multi-Model Consensus: %s @ $%.2f", result.Symbol, result.Price))
	divider := "  " + Divider(60)

	var rows []string
	rows = append(rows, "", header, divider, "")

	// Table header.
	thModel := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(fmt.Sprintf("  %-38s", "Model"))
	thVerdict := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(fmt.Sprintf("%-8s", "Verdict"))
	thConf := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(fmt.Sprintf("%-10s", "Confidence"))
	thTime := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("Time")
	rows = append(rows, thModel+thVerdict+thConf+thTime)
	rows = append(rows, "  "+Divider(70))

	for _, v := range result.Verdicts {
		model := fmt.Sprintf("  %-38s", v.Model)
		var verdictStyled string
		switch v.Verdict {
		case "BUY":
			verdictStyled = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(fmt.Sprintf("%-8s", "BUY"))
		case "SELL":
			verdictStyled = lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render(fmt.Sprintf("%-8s", "SELL"))
		case "HOLD":
			verdictStyled = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render(fmt.Sprintf("%-8s", "HOLD"))
		default:
			if v.Error != "" {
				verdictStyled = ErrorStyle.Render(fmt.Sprintf("%-8s", "ERROR"))
			} else {
				verdictStyled = DimStyle.Render(fmt.Sprintf("%-8s", "—"))
			}
		}
		conf := fmt.Sprintf("%-10s", v.Confidence)
		dur := fmt.Sprintf("%.1fs", v.Duration.Seconds())
		rows = append(rows, model+verdictStyled+conf+dur)
		if v.Reasoning != "" {
			rows = append(rows, "    "+DimStyle.Render(v.Reasoning))
		}
		if v.Error != "" {
			rows = append(rows, "    "+ErrorStyle.Render(v.Error))
		}
	}

	rows = append(rows, "  "+Divider(70))

	// Consensus summary.
	var consensusStyled string
	switch result.Consensus {
	case "BUY":
		consensusStyled = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("BUY")
	case "SELL":
		consensusStyled = lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render("SELL")
	case "HOLD":
		consensusStyled = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render("HOLD")
	default:
		consensusStyled = DimStyle.Render("NO CONSENSUS")
	}

	rows = append(rows, fmt.Sprintf("  Consensus: %s  (%s agree)", consensusStyled, result.Agreement))
	rows = append(rows, "")

	return strings.Join(rows, "\n")
}

// RenderAnalysisPresets renders available analysis presets.
func RenderAnalysisPresets() string {
	presets := backtest.GetAnalysisPresets()

	header := lipgloss.NewStyle().
		Foreground(ColorPrimary).Bold(true).
		Render("  Analysis Presets")
	divider := "  " + Divider(50)

	var rows []string
	rows = append(rows, "", header, divider, "")

	for _, p := range presets {
		name := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(fmt.Sprintf("  %-20s", p.Name))
		desc := lipgloss.NewStyle().Foreground(ColorWhite).Render(p.Description)
		rows = append(rows, name+desc)
		if len(p.MCPTools) > 0 {
			rows = append(rows, "    "+DimStyle.Render("requires: "+strings.Join(p.MCPTools, ", ")))
		}
	}

	rows = append(rows, "")
	rows = append(rows, DimStyle.Render("  Run: /analyze run <preset> [args]"))
	rows = append(rows, DimStyle.Render("  Shortcuts: /analyze sentiment <symbol>  •  /analyze whale <symbol>  •  /analyze defi"))
	rows = append(rows, "")

	return strings.Join(rows, "\n") + NextSteps("/analyze run <preset>")
}

// RenderAnalyzeHelp renders the /analyze command help.
func RenderAnalyzeHelp() string {
	header := lipgloss.NewStyle().
		Foreground(ColorPrimary).Bold(true).
		Render("  Market Analysis")
	divider := "  " + Divider(50)

	return strings.Join([]string{
		"", header, divider, "",
		"  " + CommandStyle.Render("/analyze <symbol>") + DimStyle.Render("         Technical analysis"),
		"  " + CommandStyle.Render("/analyze presets") + DimStyle.Render("          List analysis presets"),
		"  " + CommandStyle.Render("/analyze run <preset> [args]") + DimStyle.Render("  Run a preset"),
		"  " + CommandStyle.Render("/analyze sentiment <symbol>") + DimStyle.Render("  News/social sentiment"),
		"  " + CommandStyle.Render("/analyze whale <symbol>") + DimStyle.Render("      On-chain whale activity"),
		"  " + CommandStyle.Render("/analyze defi") + DimStyle.Render("              Top DeFi yields"),
		"",
		DimStyle.Render("  Example: /analyze run sentiment-check BTC"),
		"",
	}, "\n")
}

// RenderConsensusHelp renders the /consensus command help.
func RenderConsensusHelp() string {
	header := lipgloss.NewStyle().
		Foreground(ColorPrimary).Bold(true).
		Render("  Multi-LLM Consensus")
	divider := "  " + Divider(50)

	return strings.Join([]string{
		"", header, divider, "",
		"  " + CommandStyle.Render("/consensus <symbol>") + DimStyle.Render("       Tier 1 frontier panel (4 models)"),
		"  " + CommandStyle.Render("/consensus all <symbol>") + DimStyle.Render("   All tiers (10 models)"),
		"  " + CommandStyle.Render("/consensus budget <symbol>") + DimStyle.Render("  Tier 3 free models only"),
		"  " + CommandStyle.Render("/consensus models") + DimStyle.Render("         Show model tiers"),
		"",
		DimStyle.Render("  Requires OpenRouter API key:"),
		"  " + CommandStyle.Render("/config set openrouter_key <key>"),
		"",
	}, "\n")
}
