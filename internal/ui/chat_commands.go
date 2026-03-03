package ui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/commands"
)

// dispatchCommand routes a parsed command result to the appropriate handler.
// Called from updateInsertMode after the user presses Enter.
func (m Model) dispatchCommand(result commands.Result) (tea.Model, tea.Cmd) {
	switch result.Type {
	case commands.TypeQuit:
		m.cleanup()
		return m, tea.Quit
	case commands.TypeHelp:
		m.dialog = DialogState{Active: DialogHelp}
		return m, nil

	case commands.TypeClear:
		m.messages = nil
		m.viewport.SetContent(m.welcomeContent())
		m.viewport.GotoBottom()
		m.statusFlash = "Cleared"
		m.statusFlashExpiry = time.Now().Add(2 * time.Second)
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return statusFlashExpiredMsg{} })

	case commands.TypeChat:
		// Streaming AI call with loading spinner.
		if msg := m.requireAgent(); msg != "" {
			m.addBotMessage(msg)
			m.updateViewport()
			return m, nil
		}
		m.loading = true
		m.streaming = true
		m.loadingFrame = 0
		m.loadingText = "Thinking..."
		m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)
		m.updateViewport()
		tokenCh := make(chan string, 100)
		m.streamCh = tokenCh
		agent := m.agent
		userInput := result.Input
		confirmCh := m.toolRegistry.ConfirmCh
		return m, tea.Batch(
			tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
			func() tea.Msg {
				defer close(tokenCh)
				resp, err := agent.ChatStream(context.Background(), userInput, tokenCh)
				return aiStreamDoneMsg{finalContent: resp, err: err}
			},
			waitForStreamToken(tokenCh),
			waitForConfirmation(confirmCh),
		)

	case commands.TypeBuy, commands.TypeSell:
		// Trade confirmation flow (synchronous — no API call yet).
		side := "buy"
		if result.Type == commands.TypeSell {
			side = "sell"
		}
		output := m.handleTrade(side, result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		return m, nil

	case commands.TypeAlert:
		// Alert management (synchronous).
		output := m.handleAlert(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		// Start alert/trigger ticker if we just added the first alert.
		if !m.alertTicking && (len(m.alerts) > 0 || len(m.triggers) > 0) {
			m.alertTicking = true
			return m, tea.Tick(30*time.Second, func(t time.Time) tea.Msg { return alertCheckMsg{} })
		}
		return m, nil

	case commands.TypeTrigger:
		// Trigger management (synchronous).
		output := m.handleTrigger(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		// Start polling if we just added the first trigger.
		if !m.alertTicking && (len(m.alerts) > 0 || len(m.triggers) > 0) {
			m.alertTicking = true
			return m, tea.Tick(30*time.Second, func(t time.Time) tea.Msg { return alertCheckMsg{} })
		}
		return m, nil

	case commands.TypeRisk:
		output := m.handleRisk(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		return m, nil

	case commands.TypeStrategy:
		output, startTick := m.handleStrategy(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if startTick && !m.strategyTicking {
			m.strategyTicking = true
			return m, tea.Tick(60*time.Second, func(t time.Time) tea.Msg { return strategyTickMsg{} })
		}
		return m, nil

	case commands.TypeNotify:
		output := m.handleNotify(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		return m, nil

	case commands.TypeAuto:
		output, startTick := m.handleAuto(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if startTick && !m.autoTicking {
			m.autoTicking = true
			return m, tea.Tick(60*time.Second, func(t time.Time) tea.Msg { return autoTickMsg{} })
		}
		return m, nil

	case commands.TypeBacktest:
		output, cmd := m.handleBacktest(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypePolymarket:
		output, cmd := m.handlePolymarket(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypeGuide:
		output := m.handleGuide(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		return m, nil

	case commands.TypeMemory:
		output := m.handleMemory(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		return m, nil

	case commands.TypeConsensus:
		output, cmd := m.handleConsensus(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypeConnect:
		output := m.handleConnect(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		return m, nil

	case commands.TypeBalances:
		m.loading = true
		m.loadingFrame = 0
		m.loadingText = "Fetching balances..."
		m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)
		m.updateViewport()
		client := m.client
		return m, tea.Batch(
			tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
			func() tea.Msg {
				return apiResponseMsg{content: RenderBalances(client)}
			},
		)

	case commands.TypePositions:
		m.loading = true
		m.loadingFrame = 0
		m.loadingText = "Fetching positions..."
		m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)
		m.updateViewport()
		client := m.client
		return m, tea.Batch(
			tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
			func() tea.Msg {
				return apiResponseMsg{content: RenderPositions(client)}
			},
		)

	case commands.TypeMarkets:
		output, cmd := m.handleMarkets(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypeBet:
		output, cmd := m.handleBet(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypeWallet:
		output, cmd := m.handleWallet(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypeSwap:
		output, cmd := m.handleSwap(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypeGas:
		output, cmd := m.handleGas(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypeStock:
		output, cmd := m.handleStock(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypeScreen:
		output, cmd := m.handleScreen(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypeOdds:
		output, cmd := m.handleOdds(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypeLines:
		output, cmd := m.handleLines(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypeFunding:
		output, cmd := m.handleFunding(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypeWatch:
		return m.handleWatchCommand(result.Args)

	case commands.TypeDashboard:
		m.dashboardMode = !m.dashboardMode
		if m.dashboardMode {
			m.addBotMessage(BotMsgStyle.Render("nick: ") + "Dashboard mode. Press " + CommandStyle.Render("Esc") + " or " + CommandStyle.Render("/dashboard") + " to exit.")
		} else {
			m.addBotMessage(BotMsgStyle.Render("nick: ") + "Back to chat.")
		}
		m.updateViewport()
		return m, nil

	case commands.TypeVibe:
		output := m.handleVibe(result.Args)
		m.addBotMessage(output)
		m.updateViewport()
		return m, nil

	case commands.TypeExport:
		output := m.handleExport(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		return m, nil

	case commands.TypeAnalytics:
		// Async API call with loading spinner.
		m.loading = true
		m.loadingFrame = 0
		m.loadingText = "Computing analytics..."
		m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)
		m.updateViewport()
		rCopy := result
		return m, tea.Batch(
			tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
			func() tea.Msg {
				output := m.renderResult(rCopy)
				return apiResponseMsg{content: output}
			},
		)

	case commands.TypeAnalyze:
		output, cmd := m.handleAnalyze(result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	case commands.TypePrice, commands.TypeStatus, commands.TypeOrders,
		commands.TypeSnapshot, commands.TypeMarket, commands.TypePnl,
		commands.TypeHistory, commands.TypeChart:
		// Async API call with loading spinner.
		m.loading = true
		m.loadingFrame = 0
		m.loadingText = apiLoadingText(result.Type)
		m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)
		m.updateViewport()
		rCopy := result
		return m, tea.Batch(
			tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
			func() tea.Msg {
				output := m.renderResult(rCopy)
				return apiResponseMsg{content: output}
			},
		)

	case commands.TypeNode:
		output, cmd := m.handleNode(result.SubCommand, result.Args)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		if cmd != nil {
			return m, cmd
		}
		return m, nil

	default:
		output := m.renderResult(result)
		if output != "" {
			m.messages = append(m.messages, message{
				content: output,
				isUser:  false,
			})
		}
	}

	m.updateViewport()
	// Schedule status flash expiry if one was just set (e.g. /theme, /model).
	if m.statusFlash != "" {
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return statusFlashExpiredMsg{} })
	}
	return m, nil
}

// --- Command rendering ---

func (m *Model) renderResult(r commands.Result) string {
	switch r.Type {
	case commands.TypeHelp:
		return RenderHelp()

	case commands.TypeAgents:
		return RenderAgentListMock(m.width)

	case commands.TypeTemplates:
		return RenderTemplateList(m.width)

	case commands.TypeStatus:
		if m.client.IsConfigured() {
			return RenderStatusLive(m.client, m.mcpManager, m.width)
		}
		return RenderStatusMock(m.mcpManager)

	case commands.TypeOrders:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		return RenderOrderList(m.client, m.width)

	case commands.TypePrice:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		if len(r.Args) == 0 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/price BTC ETH SOL")
		}
		symbols := make([]string, len(r.Args))
		for i, s := range r.Args {
			symbols[i] = strings.ToUpper(s)
		}
		return RenderPrices(m.client, symbols, m.width)

	case commands.TypeBuy:
		return m.handleTrade("buy", r.Args)

	case commands.TypeSell:
		return m.handleTrade("sell", r.Args)

	case commands.TypeConfig:
		return m.handleConfig(r.Args)

	case commands.TypeCredential:
		return m.handleCredential(r.Args)

	case commands.TypeWorkflow:
		return m.handleWorkflow(r.Args)

	case commands.TypeLogs:
		return m.handleLogs(r.Args)

	case commands.TypeMan:
		if len(r.Args) > 0 {
			return RenderManPage(r.Args[0])
		}
		return RenderManIndex()

	case commands.TypePlugin:
		return m.handlePlugin(r.Args)

	case commands.TypeUnknown:
		hint := DimStyle.Render("Type /help for available commands.")
		if len(r.Args) > 0 && strings.HasPrefix(r.Args[0], "Did you mean") {
			hint = lipgloss.NewStyle().Foreground(ColorWarning).Render(r.Args[0])
		}
		return ErrorStyle.Render("Unknown command: ") + r.Input + "\n" + hint

	case commands.TypeChart:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		if len(r.Args) == 0 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/chart BTC")
		}
		return RenderChart(m.client, strings.ToUpper(r.Args[0]), m.width)

	case commands.TypeAlert:
		return m.handleAlert(r.Args)

	case commands.TypeTrigger:
		return m.handleTrigger(r.Args)

	case commands.TypeRisk:
		return m.handleRisk(r.Args)

	case commands.TypeStrategy:
		output, _ := m.handleStrategy(r.Args)
		return output

	case commands.TypeNotify:
		return m.handleNotify(r.Args)

	case commands.TypeAnalytics:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		return RenderAnalytics(m.client, m.width)

	case commands.TypeAnalyze:
		// Handled by handleAnalyze() now.
		return ""

	case commands.TypeAuto:
		output, _ := m.handleAuto(r.Args)
		return output

	case commands.TypeExport:
		return m.handleExport(r.Args)

	case commands.TypeTheme:
		return m.handleTheme(r.Args)

	case commands.TypeModel:
		return m.handleModel(r.Args)

	case commands.TypeMCP:
		return m.handleMCP(r.Args)

	case commands.TypeSnapshot:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		return RenderSnapshot(m.client, m.width)

	case commands.TypeMarket:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		return RenderMarket(m.client, m.width)

	case commands.TypePnl:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		return RenderPnl(m.client, m.width)

	case commands.TypeHistory:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		return RenderHistory(m.client, m.width)

	case commands.TypeChat:
		if msg := m.requireAgent(); msg != "" {
			return msg
		}
		resp, err := m.agent.Chat(context.Background(), r.Input)
		if err != nil {
			return ErrorStyle.Render("  AI error: ") + err.Error()
		}
		rendered := renderMarkdown(resp, m.width-8)
		return BotMsgStyle.Render("nick:") + "\n" + rendered

	default:
		return ""
	}
}

// streamToAI is a helper that sends a prompt to the AI agent with streaming.
// origin identifies the command source for post-stream next-step hints.
func (m *Model) streamToAI(prompt, loadingText, origin string) (string, tea.Cmd) {
	m.loading = true
	m.streaming = true
	m.loadingFrame = 0
	m.loadingText = loadingText
	m.streamOrigin = origin
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
