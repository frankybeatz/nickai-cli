package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/ai"
	"github.com/nickai/cli/internal/automation"
	"github.com/nickai/cli/internal/backtest"
	"github.com/nickai/cli/internal/indicators"
	"github.com/nickai/cli/internal/market"
)

// handleBacktest processes /backtest subcommands.
func (m *Model) handleBacktest(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return RenderBacktestHelp(), nil
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "presets", "list":
		return RenderBacktestPresets(), nil

	case "run":
		if len(args) < 3 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/backtest run <preset> <symbol> [period]"), nil
		}
		presetName := strings.ToLower(args[1])
		symbol := strings.ToUpper(args[2])
		period := "180d"
		if len(args) >= 4 {
			period = args[3]
		}

		preset := backtest.GetPreset(presetName)
		if preset == nil {
			return ErrorStyle.Render("  Unknown preset: ") + presetName + "\n" +
				DimStyle.Render("  Run /backtest presets to see available strategies"), nil
		}

		// Run backtest asynchronously with loading spinner.
		m.loading = true
		m.loadingFrame = 0
		m.loadingText = "Running backtest..."
		m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)

		strat := preset.Strategy
		strat.Symbol = symbol
		strat.Period = period

		return "", tea.Batch(
			tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
			func() tea.Msg {
				result, err := backtest.Run(strat)
				return backtestDoneMsg{result: result, err: err}
			},
		)

	case "analyze":
		if msg := m.requireAgent(); msg != "" {
			return msg, nil
		}
		if m.lastBacktestResult == nil {
			return ErrorStyle.Render("  No backtest results to analyze.") + "\n" +
				DimStyle.Render("  Run a backtest first, then use /backtest analyze"), nil
		}
		prompt := backtest.AnalysisSystemPrompt + "\n\n" + backtest.FormatResultForAnalysis(m.lastBacktestResult)
		return m.streamToAI(prompt, "Analyzing backtest results...", "backtest-analyze")

	case "activate":
		// /backtest activate <preset> <symbol> [value]
		if msg := m.requireAgent(); msg != "" {
			return msg, nil
		}
		if len(args) < 3 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/backtest activate <preset> <symbol> [value]"), nil
		}
		presetName := strings.ToLower(args[1])
		symbol := strings.ToUpper(args[2])
		value := ""
		if len(args) >= 4 {
			value = args[3]
		}
		prompt := fmt.Sprintf("Activate the %s strategy for %s as a live monitoring rule using the activate_strategy tool.", presetName, symbol)
		if value != "" {
			prompt += fmt.Sprintf(" Trade size: $%s.", value)
		}

		m.loading = true
		m.streaming = true
		m.loadingFrame = 0
		m.loadingText = "Activating strategy..."
		m.streamOrigin = "backtest"
		m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)

		tokenCh := make(chan string, 100)
		m.streamCh = tokenCh
		agent := m.agent
		confirmCh := m.toolRegistry.ConfirmCh
		return "", tea.Batch(
			tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
			func() tea.Msg {
				defer close(tokenCh)
				resp, err := agent.ChatStream(context.Background(), prompt, tokenCh)
				return aiStreamDoneMsg{finalContent: resp, err: err}
			},
			waitForStreamToken(tokenCh),
			waitForConfirmation(confirmCh),
		)

	default:
		// Pass to AI as natural language backtest request.
		if msg := m.requireAgent(); msg != "" {
			return msg, nil
		}

		prompt := "Backtest the following strategy using the backtest_strategy tool: " + strings.Join(args, " ")
		m.loading = true
		m.streaming = true
		m.loadingFrame = 0
		m.loadingText = "Building backtest..."
		m.streamOrigin = "backtest"
		m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)

		tokenCh := make(chan string, 100)
		m.streamCh = tokenCh
		agent := m.agent
		confirmCh := m.toolRegistry.ConfirmCh
		return "", tea.Batch(
			tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
			func() tea.Msg {
				defer close(tokenCh)
				resp, err := agent.ChatStream(context.Background(), prompt, tokenCh)
				return aiStreamDoneMsg{finalContent: resp, err: err}
			},
			waitForStreamToken(tokenCh),
			waitForConfirmation(confirmCh),
		)
	}
}

// handlePolymarket processes /polymarket subcommands.
func (m *Model) handlePolymarket(args []string) (string, tea.Cmd) {
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}

	sub := "scan"
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}

	var prompt string
	switch sub {
	case "scan":
		preset := backtest.GetAnalysisPreset("polymarket-scan")
		if preset != nil {
			prompt = preset.Prompt
		} else {
			prompt = "Scan top Polymarket events and find mispriced contracts using available MCP tools."
		}
	case "analyze":
		event := strings.Join(args[1:], " ")
		if event == "" {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/polymarket analyze <event>"), nil
		}
		preset := backtest.GetAnalysisPreset("polymarket-deep")
		if preset != nil {
			prompt = fmt.Sprintf(preset.Prompt, event)
		} else {
			prompt = "Do a deep analysis of this Polymarket event: " + event
		}
	case "hot":
		prompt = "Show trending Polymarket events with the biggest volume and price moves using available tools."
	default:
		prompt = "Analyze the following using Polymarket tools: " + strings.Join(args, " ")
	}

	m.loading = true
	m.streaming = true
	m.loadingFrame = 0
	m.loadingText = "Analyzing prediction markets..."
	m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)

	tokenCh := make(chan string, 100)
	m.streamCh = tokenCh
	agent := m.agent
	confirmCh := m.toolRegistry.ConfirmCh
	return "", tea.Batch(
		tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
		func() tea.Msg {
			defer close(tokenCh)
			resp, err := agent.ChatStream(context.Background(), prompt, tokenCh)
			return aiStreamDoneMsg{finalContent: resp, err: err}
		},
		waitForStreamToken(tokenCh),
		waitForConfirmation(confirmCh),
	)
}

// handleGuide processes /guide subcommands.
func (m *Model) handleGuide(args []string) string {
	section := "start"
	if len(args) > 0 {
		section = strings.ToLower(args[0])
	}
	return RenderGuideCard(section)
}

// handleMemory processes /memory subcommands.
func (m *Model) handleMemory(args []string) string {
	if m.memoryStore == nil {
		return ErrorStyle.Render("  Memory store unavailable.")
	}

	if len(args) == 0 {
		return RenderMemoryList(m.memoryStore.Entries)
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "clear":
		m.memoryStore.Entries = nil
		_ = m.memoryStore.Save()
		if m.agent != nil {
			m.agent.SetMemoryInfo("")
		}
		return BotMsgStyle.Render("nick: ") + "All memories cleared."

	case "remove", "rm":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/memory remove <id-prefix>")
		}
		m.memoryStore.Remove(args[1])
		_ = m.memoryStore.Save()
		return BotMsgStyle.Render("nick: ") + "Memory removed."

	default:
		return RenderMemoryList(m.memoryStore.Entries)
	}
}

// handleConsensus processes /consensus subcommands.
func (m *Model) handleConsensus(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return RenderConsensusHelp(), nil
	}

	sub := strings.ToLower(args[0])

	// /consensus models — show model tiers.
	if sub == "models" {
		var rows []string
		header := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("  Consensus Model Tiers")
		divider := "  " + Divider(50)
		rows = append(rows, "", header, divider, "")

		tierLabel := func(name string) string {
			return lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("  " + name)
		}

		rows = append(rows, tierLabel("Tier 1 — Frontier (default):"))
		for _, m := range ai.Tier1Models {
			rows = append(rows, "    "+lipgloss.NewStyle().Foreground(ColorWhite).Render(m))
		}
		rows = append(rows, "")
		rows = append(rows, tierLabel("Tier 2 — Diversity:"))
		for _, m := range ai.Tier2Models {
			rows = append(rows, "    "+lipgloss.NewStyle().Foreground(ColorWhite).Render(m))
		}
		rows = append(rows, "")
		rows = append(rows, tierLabel("Tier 3 — Budget (free):"))
		for _, m := range ai.Tier3Models {
			rows = append(rows, "    "+lipgloss.NewStyle().Foreground(ColorWhite).Render(m))
		}
		rows = append(rows, "")
		return strings.Join(rows, "\n"), nil
	}

	// Determine models and symbol.
	var models []string
	var symbol string
	switch sub {
	case "all":
		models = ai.AllConsensusModels
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/consensus all <symbol>"), nil
		}
		symbol = strings.ToUpper(args[1])
	case "budget":
		models = ai.Tier3Models
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/consensus budget <symbol>"), nil
		}
		symbol = strings.ToUpper(args[1])
	default:
		models = ai.DefaultConsensusModels
		symbol = strings.ToUpper(sub)
	}

	// Check for OpenRouter key.
	orKey := m.cfg.DataKeyOrEnv("openrouter")
	if orKey == "" {
		return ErrorStyle.Render("  OpenRouter API key required.") + "\n" +
			DimStyle.Render("  Set via: ") + CommandStyle.Render("/config set openrouter_key <key>"), nil
	}

	// Fetch current price.
	prices, err := m.client.GetPrices([]string{symbol})
	if err != nil || len(prices) == 0 {
		return ErrorStyle.Render("  Failed to fetch price for " + symbol), nil
	}
	price := prices[0].Price

	// Build market context.
	var marketContext string
	if candles, err := market.FetchKlines(symbol, "1d", 30); err == nil && len(candles) > 0 {
		closes := market.ClosePrices(candles)
		rsi := indicators.RSI(closes, 14)
		macdLine, macdSignal, _ := indicators.MACDCalc(closes)
		sma20 := indicators.SMA(closes, 20)
		trend := indicators.TrendDirection(closes)
		marketContext = fmt.Sprintf("RSI(14): %.1f | MACD: %.2f (signal: %.2f) | SMA20: %.2f | Trend: %s",
			rsi, macdLine, macdSignal, sma20, trend)
	}

	// Run async with loading spinner.
	m.loading = true
	m.loadingFrame = 0
	m.loadingText = fmt.Sprintf("Querying %d models for %s...", len(models), symbol)
	m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)

	orClient := ai.NewOpenRouterClient(orKey)
	cfg := ai.ConsensusConfig{Models: models, Threshold: 0.67}
	return "", tea.Batch(
		tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
		func() tea.Msg {
			result := ai.RunConsensus(orClient, cfg, symbol, price, marketContext)
			return consensusDoneMsg{result: result}
		},
	)
}

// handleAnalyze processes /analyze subcommands.
func (m *Model) handleAnalyze(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return RenderAnalyzeHelp(), nil
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "presets":
		return RenderAnalysisPresets(), nil

	case "run":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/analyze run <preset> [args]"), nil
		}
		presetName := strings.ToLower(args[1])
		preset := backtest.GetAnalysisPreset(presetName)
		if preset == nil {
			return ErrorStyle.Render("  Unknown preset: ") + presetName + "\n" +
				DimStyle.Render("  Run /analyze presets to see available presets"), nil
		}
		extraArgs := ""
		if len(args) > 2 {
			extraArgs = strings.Join(args[2:], " ")
		}
		return m.runAnalysisPreset(preset, extraArgs)

	case "sentiment":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/analyze sentiment <symbol>"), nil
		}
		preset := backtest.GetAnalysisPreset("sentiment-check")
		if preset == nil {
			return ErrorStyle.Render("  Preset 'sentiment-check' not found."), nil
		}
		return m.runAnalysisPreset(preset, strings.ToUpper(args[1]))

	case "whale":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/analyze whale <symbol>"), nil
		}
		preset := backtest.GetAnalysisPreset("whale-watch")
		if preset == nil {
			return ErrorStyle.Render("  Preset 'whale-watch' not found."), nil
		}
		return m.runAnalysisPreset(preset, strings.ToUpper(args[1]))

	case "defi":
		preset := backtest.GetAnalysisPreset("defi-yield")
		if preset == nil {
			return ErrorStyle.Render("  Preset 'defi-yield' not found."), nil
		}
		return m.runAnalysisPreset(preset, "")

	default:
		// Backward-compatible: /analyze BTC → technical analysis.
		if !m.client.IsConfigured() {
			return BotMsgStyle.Render("nick: ") +
				"Connect a paper trading account first with " +
				CommandStyle.Render("/config init"), nil
		}
		symbol := strings.ToUpper(sub)
		m.loading = true
		m.loadingFrame = 0
		m.loadingText = "Analyzing market..."
		m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)
		client := m.client
		width := m.width
		return "", tea.Batch(
			tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
			func() tea.Msg {
				return apiResponseMsg{content: RenderAnalysis(client, symbol, width)}
			},
		)
	}
}

// runAnalysisPreset sends a preset's AI prompt through the agent.
func (m *Model) runAnalysisPreset(preset *backtest.AnalysisPreset, extraArgs string) (string, tea.Cmd) {
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}

	// Check MCP tool requirements.
	if missing := m.checkMCPTools(preset.MCPTools); len(missing) > 0 {
		return ErrorStyle.Render("  Missing MCP servers: ") +
			strings.Join(missing, ", ") + "\n" +
			DimStyle.Render("  Install via: ") + CommandStyle.Render("/mcp add <server>"), nil
	}

	prompt := preset.Prompt
	if extraArgs != "" {
		prompt = strings.ReplaceAll(prompt, "{symbol}", extraArgs)
		prompt = strings.ReplaceAll(prompt, "{args}", extraArgs)
		if !strings.Contains(preset.Prompt, "{") {
			prompt = prompt + " " + extraArgs
		}
	}

	m.loading = true
	m.streaming = true
	m.loadingFrame = 0
	m.loadingText = "Running " + preset.Name + "..."
	m.streamOrigin = "analyze"
	m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)

	tokenCh := make(chan string, 100)
	m.streamCh = tokenCh
	agent := m.agent
	confirmCh := m.toolRegistry.ConfirmCh
	return "", tea.Batch(
		tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
		func() tea.Msg {
			defer close(tokenCh)
			resp, err := agent.ChatStream(context.Background(), prompt, tokenCh)
			return aiStreamDoneMsg{finalContent: resp, err: err}
		},
		waitForStreamToken(tokenCh),
		waitForConfirmation(confirmCh),
	)
}

// checkMCPTools returns a list of required MCP server names that are not connected.
func (m *Model) checkMCPTools(required []string) []string {
	if len(required) == 0 || m.mcpManager == nil {
		return nil
	}
	connSet := map[string]bool{}
	for _, c := range m.mcpManager.Connections() {
		connSet[c.Name] = true
	}
	var missing []string
	for _, r := range required {
		if !connSet[r] {
			missing = append(missing, r)
		}
	}
	return missing
}

// computeIndicatorSnapshot builds an indicator snapshot from close prices.
func computeIndicatorSnapshot(closePrices []float64) automation.IndicatorSnapshot {
	snap := automation.IndicatorSnapshot{
		Price: closePrices[len(closePrices)-1],
	}
	if len(closePrices) >= 14 {
		snap.RSI = indicators.RSI(closePrices, 14)
	}
	if len(closePrices) >= 26 {
		snap.MACD, snap.MACDSignal, snap.MACDHistogram = indicators.MACDCalc(closePrices)
	}
	if len(closePrices) >= 20 {
		snap.SMA20 = indicators.SMA(closePrices, 20)
		snap.BollingerUpper, _, snap.BollingerLower = indicators.BollingerBands(closePrices, 20)
	}
	if len(closePrices) >= 50 {
		snap.SMA50 = indicators.SMA(closePrices, 50)
	}
	if len(closePrices) >= 12 {
		snap.EMA12 = indicators.EMA(closePrices, 12)
	}
	if len(closePrices) >= 26 {
		snap.EMA26 = indicators.EMA(closePrices, 26)
	}
	return snap
}
