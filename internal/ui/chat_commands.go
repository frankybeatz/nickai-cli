package ui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/ai"
	"github.com/nickai/cli/internal/alert"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/automation"
	"github.com/nickai/cli/internal/backtest"
	"github.com/nickai/cli/internal/commands"
	"github.com/nickai/cli/internal/credential"
	"github.com/nickai/cli/internal/indicators"
	"github.com/nickai/cli/internal/market"
	"github.com/nickai/cli/internal/mcp"
	"github.com/nickai/cli/internal/notify"
	"github.com/nickai/cli/internal/personality"
	"github.com/nickai/cli/internal/risk"
	"github.com/nickai/cli/internal/strategy"
	"github.com/nickai/cli/internal/trigger"
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

	case commands.TypeWatch:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		if len(r.Args) == 0 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/watch BTC ETH SOL")
		}
		symbols := make([]string, len(r.Args))
		for i, s := range r.Args {
			symbols[i] = strings.ToUpper(s)
		}
		return RenderWatch(m.client, symbols, m.width)

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

// --- Alert handling ---

func (m *Model) handleAlert(args []string) string {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/alert BTC > 100000") + "\n" +
			DimStyle.Render("  Subcommands: /alert list, /alert clear")
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "list":
		if len(m.alerts) == 0 {
			return DimStyle.Render("  No active alerts. Create one with ") +
				CommandStyle.Render("/alert BTC > 100000")
		}
		var lines []string
		lines = append(lines, SecondaryStyle.Render("  Active Alerts\n"))
		for _, a := range m.alerts {
			lines = append(lines, "  "+StatusIndicator("running")+
				BrandStyle.Render(a.Symbol)+" "+a.Operator+" "+formatPrice(a.Target))
		}
		return strings.Join(lines, "\n")

	case "clear":
		count := len(m.alerts)
		m.alerts = nil
		_ = alert.Clear()
		return BotMsgStyle.Render("nick: ") +
			fmt.Sprintf("Cleared %d alert(s).", count)
	}

	// Parse: /alert BTC > 100000
	if len(args) < 3 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/alert BTC > 100000")
	}
	symbol := strings.ToUpper(args[0])
	op := args[1]
	if op != ">" && op != "<" {
		return ErrorStyle.Render("  Operator must be ") +
			CommandStyle.Render(">") + " or " + CommandStyle.Render("<")
	}
	target, err := strconv.ParseFloat(args[2], 64)
	if err != nil || target <= 0 {
		return ErrorStyle.Render("  Invalid target price: ") + args[2]
	}

	a := alert.Alert{
		Symbol:   symbol,
		Operator: op,
		Target:   target,
		Created:  time.Now(),
	}
	m.alerts = append(m.alerts, a)
	_ = alert.Add(a)

	return BotMsgStyle.Render("nick: ") + "Alert set: " +
		BrandStyle.Render(symbol) + " " + op + " " + formatPrice(target) +
		DimStyle.Render("  (checking every 30s, persists across restarts)")
}

// --- Trigger handling ---

func (m *Model) handleTrigger(args []string) string {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/trigger add BTC < 60000 sell 0.5") + "\n" +
			DimStyle.Render("  Subcommands: /trigger list, /trigger add, /trigger remove <id>, /trigger clear")
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "list":
		if len(m.triggers) == 0 {
			return DimStyle.Render("  No active triggers. Create one with ") +
				CommandStyle.Render("/trigger add BTC < 60000 sell 0.5")
		}
		return RenderTriggerList(m.triggers)

	case "clear":
		count := len(m.triggers)
		m.triggers = nil
		_ = trigger.Clear()
		return BotMsgStyle.Render("nick: ") +
			fmt.Sprintf("Cleared %d trigger(s).", count)

	case "remove", "rm", "delete":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/trigger remove <id>")
		}
		idPrefix := args[1]
		found := false
		for i, t := range m.triggers {
			if len(t.ID) >= len(idPrefix) && t.ID[:len(idPrefix)] == idPrefix {
				m.triggers = append(m.triggers[:i], m.triggers[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			return ErrorStyle.Render("  No trigger found with ID prefix: ") + idPrefix
		}
		_ = trigger.Remove(idPrefix)
		return BotMsgStyle.Render("nick: ") + "Trigger removed."

	case "add":
		// /trigger add BTC < 60000 sell 0.5 [market]
		if len(args) < 6 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/trigger add BTC < 60000 sell 0.5") + "\n" +
				DimStyle.Render("  Format: /trigger add <symbol> <> or <> <price> <buy|sell> <qty> [market|limit]")
		}
		symbol := strings.ToUpper(args[1])
		op := args[2]
		if op != ">" && op != "<" {
			return ErrorStyle.Render("  Operator must be ") +
				CommandStyle.Render(">") + " or " + CommandStyle.Render("<")
		}
		target, err := strconv.ParseFloat(args[3], 64)
		if err != nil || target <= 0 {
			return ErrorStyle.Render("  Invalid target price: ") + args[3]
		}
		side := strings.ToLower(args[4])
		if side != "buy" && side != "sell" {
			return ErrorStyle.Render("  Side must be ") +
				CommandStyle.Render("buy") + " or " + CommandStyle.Render("sell")
		}
		qty, err := strconv.ParseFloat(args[5], 64)
		if err != nil || qty <= 0 {
			return ErrorStyle.Render("  Invalid quantity: ") + args[5]
		}
		orderType := "market"
		if len(args) > 6 {
			orderType = strings.ToLower(args[6])
		}

		t := trigger.Trigger{
			ID:        randomID(8),
			Symbol:    symbol,
			Operator:  op,
			Target:    target,
			Action:    trigger.Action{Side: side, Quantity: qty, Type: orderType},
			CreatedAt: time.Now(),
		}
		m.triggers = append(m.triggers, t)
		_ = trigger.Add(t)

		return BotMsgStyle.Render("nick: ") + "Trigger set: " +
			DimStyle.Render("if ") + BrandStyle.Render(symbol) + " " + op + " " + formatPrice(target) +
			DimStyle.Render(" → ") +
			lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(
				strings.ToUpper(side)+" "+strconv.FormatFloat(qty, 'f', -1, 64)+" "+symbol) +
			DimStyle.Render("  (ID: "+t.ID+")")
	}

	return ErrorStyle.Render("  Unknown subcommand: ") + sub + "\n" +
		DimStyle.Render("  Try: list, add, remove, clear")
}

// --- Notify handling ---

func (m *Model) handleNotify(args []string) string {
	if len(args) == 0 {
		return RenderNotifyHelp()
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "show":
		return RenderNotifyConfig(m.notifyConfig)

	case "set":
		if len(args) < 3 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/notify set desktop on|off")
		}
		key := strings.ToLower(args[1])
		value := strings.ToLower(args[2])

		switch key {
		case "desktop":
			m.notifyConfig.Desktop = (value == "on" || value == "true" || value == "1")
		case "sound":
			m.notifyConfig.Sound = (value == "on" || value == "true" || value == "1")
		case "webhook":
			m.notifyConfig.WebhookURL = args[2] // preserve case for URL
		default:
			return ErrorStyle.Render("  Unknown setting: ") + key +
				"\n" + DimStyle.Render("  Valid keys: desktop, sound, webhook")
		}

		_ = notify.Save(m.notifyConfig)
		return BotMsgStyle.Render("nick: ") + "Notification setting updated." +
			"\n" + RenderNotifyConfig(m.notifyConfig)

	case "clear":
		m.notifyConfig = &notify.Config{}
		_ = notify.Save(m.notifyConfig)
		return BotMsgStyle.Render("nick: ") + "Notification settings cleared."

	case "test":
		if m.notifyConfig.IsEmpty() {
			return ErrorStyle.Render("  No notification channels configured.") + "\n" +
				DimStyle.Render("  Set one first with ") + CommandStyle.Render("/notify set desktop on")
		}
		notify.Send(m.notifyConfig, "NickAI Test", "Notifications are working!")
		return BotMsgStyle.Render("nick: ") + "Test notification sent."
	}

	return ErrorStyle.Render("  Unknown subcommand: ") + sub + "\n" +
		DimStyle.Render("  Try: show, set, clear, test")
}

// --- Automation handling ---

func (m *Model) handleAuto(args []string) (string, bool) {
	if len(args) == 0 {
		return RenderAutoHelp(), false
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "list":
		m.automations, _ = automation.Load()
		return RenderAutoList(m.automations), false

	case "pause":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/auto pause <id>"), false
		}
		if err := automation.Pause(args[1]); err != nil {
			return ErrorStyle.Render("  " + err.Error()), false
		}
		m.automations, _ = automation.Load()
		return BotMsgStyle.Render("nick: ") + "Automation rule paused.", false

	case "resume":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/auto resume <id>"), false
		}
		if err := automation.Resume(args[1]); err != nil {
			return ErrorStyle.Render("  " + err.Error()), false
		}
		m.automations, _ = automation.Load()
		// May need to start tick.
		needTick := false
		for _, r := range m.automations {
			if r.Status == "active" {
				needTick = true
				break
			}
		}
		return BotMsgStyle.Render("nick: ") + "Automation rule resumed.", needTick

	case "remove", "rm", "delete":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/auto remove <id>"), false
		}
		if err := automation.Remove(args[1]); err != nil {
			return ErrorStyle.Render("  " + err.Error()), false
		}
		m.automations, _ = automation.Load()
		return BotMsgStyle.Render("nick: ") + "Automation rule removed.", false
	}

	return ErrorStyle.Render("  Unknown subcommand: ") + sub + "\n" +
		DimStyle.Render("  Try: list, pause, resume, remove"), false
}

// --- /backtest handler ---

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
				if err != nil {
					return apiResponseMsg{content: ErrorStyle.Render("  Backtest failed: ") + err.Error()}
				}
				return apiResponseMsg{content: RenderBacktestCard(result)}
			},
		)

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

// --- /polymarket handler ---

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

// --- /guide handler ---

func (m *Model) handleGuide(args []string) string {
	section := "start"
	if len(args) > 0 {
		section = strings.ToLower(args[0])
	}
	return RenderGuideCard(section)
}

// --- Memory command handler ---

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

// --- Consensus command handler ---

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

// --- Analyze command handler ---

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

// --- Multi-vertical command handlers ---

// exchangeMap maps common exchange names to MCP server names.
var exchangeMap = map[string]string{
	"binance":     "binance",
	"coinbase":    "ccxt",
	"hyperliquid": "ccxt",
	"kraken":      "ccxt",
	"bybit":       "ccxt",
	"alpaca":      "alpaca",
}

func (m *Model) handleConnect(args []string) string {
	if len(args) == 0 {
		return RenderConnectHelp()
	}

	sub := strings.ToLower(args[0])
	if sub == "list" {
		if m.mcpManager == nil || m.mcpManager.ConnectionCount() == 0 {
			return BotMsgStyle.Render("nick: ") + "No exchanges connected." + "\n" +
				DimStyle.Render("  Run /connect to see available exchanges.")
		}
		var rows []string
		rows = append(rows, BotMsgStyle.Render("nick: ")+"Connected exchanges:")
		for _, c := range m.mcpManager.Connections() {
			rows = append(rows, "  "+lipgloss.NewStyle().Foreground(ColorPrimary).Render("●")+" "+c.Name+
				DimStyle.Render(fmt.Sprintf(" (%d tools)", len(c.Tools))))
		}
		return strings.Join(rows, "\n")
	}

	// Map exchange name to MCP server.
	serverName, ok := exchangeMap[sub]
	if !ok {
		return ErrorStyle.Render("  Unknown exchange: ") + sub + "\n" +
			DimStyle.Render("  Available: binance, coinbase, hyperliquid, kraken, bybit, alpaca")
	}
	return BotMsgStyle.Render("nick: ") + "To connect " + sub + ", run:\n" +
		"  " + CommandStyle.Render("/mcp add "+serverName) + "\n" +
		DimStyle.Render("  This installs the MCP server for "+sub+".")
}

func (m *Model) handleMarkets(args []string) (string, tea.Cmd) {
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}

	prompt := "Show trending prediction markets with highest volume. Use available polymarket or prediction market tools."
	if len(args) > 0 {
		prompt = "Search prediction markets for: " + strings.Join(args, " ")
	}

	return m.streamToAI(prompt, "Searching markets...", "markets")
}

func (m *Model) handleBet(args []string) (string, tea.Cmd) {
	if len(args) < 3 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/bet <market> <yes|no> <amount>") + "\n" +
			DimStyle.Render("  Example: /bet \"Trump wins\" yes 50"), nil
	}
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	prompt := fmt.Sprintf("Place a prediction market bet: market=%s, side=%s, amount=$%s. Use the polymarket tools to execute.",
		args[0], args[1], args[2])
	return m.streamToAI(prompt, "Placing bet...", "bet")
}

func (m *Model) handleWallet(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return RenderWalletHelp(), nil
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "balance", "bal":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/wallet balance <address>"), nil
		}
		if msg := m.requireAgent(); msg != "" {
			return msg, nil
		}
		prompt := "Check the wallet balance for address: " + args[1] + ". Use onchain/web3 MCP tools if available."
		return m.streamToAI(prompt, "Checking wallet...", "wallet")

	default:
		if msg := m.requireAgent(); msg != "" {
			return msg, nil
		}
		prompt := "Wallet command: " + strings.Join(args, " ")
		return m.streamToAI(prompt, "Processing wallet request...", "wallet")
	}
}

func (m *Model) handleSwap(args []string) (string, tea.Cmd) {
	if len(args) < 3 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/swap <from> <to> <amount>") + "\n" +
			DimStyle.Render("  Example: /swap SOL USDC 10"), nil
	}
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}

	from := strings.ToUpper(args[0])
	to := strings.ToUpper(args[1])
	amount := args[2]
	prompt := fmt.Sprintf("Swap %s %s to %s using Jupiter (Solana) or LiFi (cross-chain) MCP servers. Confirm before executing.", amount, from, to)
	return m.streamToAI(prompt, fmt.Sprintf("Swapping %s %s → %s...", amount, from, to), "swap")
}

func (m *Model) handleGas(args []string) (string, tea.Cmd) {
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	chain := "ethereum"
	if len(args) > 0 {
		chain = strings.ToLower(args[0])
	}
	prompt := fmt.Sprintf("Fetch current gas prices for %s. Show fast, standard, and slow estimates. Use onchain MCP tools if available.", chain)
	return m.streamToAI(prompt, "Fetching gas prices...", "gas")
}

func (m *Model) handleStock(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/stock <ticker>") + "\n" +
			DimStyle.Render("  Example: /stock AAPL"), nil
	}
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	ticker := strings.ToUpper(args[0])
	prompt := fmt.Sprintf("Analyze stock %s — current price, key fundamentals (P/E, market cap, revenue), and recent news. Use Alpaca MCP if connected, otherwise use your knowledge.", ticker)
	return m.streamToAI(prompt, "Analyzing "+ticker+"...", "stock")
}

func (m *Model) handleScreen(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/screen <filters>") + "\n" +
			DimStyle.Render("  Example: /screen high dividend tech stocks under $50"), nil
	}
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	prompt := "Screen stocks matching these criteria: " + strings.Join(args, " ") + ". List top 10 matches with ticker, price, and why they match."
	return m.streamToAI(prompt, "Screening stocks...", "stock")
}

func (m *Model) handleOdds(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/odds <event>") + "\n" +
			DimStyle.Render("  Example: /odds Lakers vs Celtics"), nil
	}
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	prompt := "Find current betting odds for: " + strings.Join(args, " ") + ". Show moneyline, spread, and over/under from major sportsbooks. Use brave-search MCP or web tools if available."
	return m.streamToAI(prompt, "Finding odds...", "bet")
}

func (m *Model) handleLines(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/lines <event>") + "\n" +
			DimStyle.Render("  Example: /lines Super Bowl"), nil
	}
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	prompt := "Show line movement and betting line history for: " + strings.Join(args, " ") + ". Highlight any significant shifts. Use brave-search MCP or web tools if available."
	return m.streamToAI(prompt, "Checking line movement...", "bet")
}

func (m *Model) handleFunding(args []string) (string, tea.Cmd) {
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	prompt := "Show current funding rates for major perpetual contracts (BTC, ETH, SOL, and any other notable rates). Include annualized rates and direction. Use exchange MCP tools if available."
	if len(args) > 0 {
		prompt = "Show funding rates for: " + strings.Join(args, " ")
	}
	return m.streamToAI(prompt, "Fetching funding rates...", "funding")
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

// handleTrade processes /buy and /sell commands.
func (m *Model) handleTrade(side string, args []string) string {
	if !m.client.IsConfigured() {
		return connectPrompt()
	}

	if len(args) < 2 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render(fmt.Sprintf("/%s BTC 0.1", side)) +
			DimStyle.Render("  or  ") +
			CommandStyle.Render(fmt.Sprintf("/%s BTC 0.1 limit 65000", side))
	}

	symbol := NormalizeSymbol(args[0])

	qty, err := strconv.ParseFloat(args[1], 64)
	if err != nil || qty <= 0 {
		return ErrorStyle.Render("  Invalid quantity: ") + args[1] +
			"\n" + DimStyle.Render("  Must be a positive number, e.g. 0.1")
	}

	orderType := "market"
	var limitPrice float64

	if len(args) >= 4 && strings.ToLower(args[2]) == "limit" {
		orderType = "limit"
		limitPrice, err = strconv.ParseFloat(args[3], 64)
		if err != nil || limitPrice <= 0 {
			return ErrorStyle.Render("  Invalid limit price: ") + args[3] +
				"\n" + DimStyle.Render("  Must be a positive number, e.g. 65000")
		}
	}

	req := api.PlaceOrderRequest{
		Symbol:   symbol,
		Quantity: qty,
		Side:     side,
		Type:     orderType,
		Price:    limitPrice,
	}

	// Store pending trade and enter confirmation mode.
	m.pendingTrade = &req
	m.vimMode = ModeConfirm
	m.textInput.Blur()

	return RenderTradeConfirmCard(&req, m.width)
}

// handleRisk processes /risk subcommands.
func (m *Model) handleRisk(args []string) string {
	if len(args) == 0 {
		return RenderRiskHelp()
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "show":
		return RenderRiskLimits(m.riskLimits)

	case "clear":
		m.riskLimits = &risk.RiskLimits{}
		_ = risk.Save(m.riskLimits)
		if m.agent != nil {
			m.agent.SetRiskInfo("")
		}
		return BotMsgStyle.Render("nick: ") + "All risk limits cleared."

	case "set":
		if len(args) < 3 {
			return RenderRiskHelp()
		}
		key := strings.ToLower(args[1])
		valueStr := strings.TrimPrefix(args[2], "$")
		valueStr = strings.TrimSuffix(valueStr, "%")
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil || value <= 0 {
			return ErrorStyle.Render("  Invalid value: ") + args[2]
		}

		if m.riskLimits == nil {
			m.riskLimits = &risk.RiskLimits{}
		}

		switch key {
		case "max-order", "maxorder":
			m.riskLimits.MaxOrderValue = value
		case "max-position", "maxposition":
			m.riskLimits.MaxPositionPct = value
		case "daily-loss", "dailyloss":
			m.riskLimits.DailyLossPct = value
		default:
			return ErrorStyle.Render("  Unknown limit: ") + key + "\n" +
				DimStyle.Render("  Valid: max-order, max-position, daily-loss")
		}

		_ = risk.Save(m.riskLimits)
		if m.agent != nil {
			m.agent.SetRiskInfo(riskPromptFromLimits(m.riskLimits))
		}
		return BotMsgStyle.Render("nick: ") + "Risk limit set.\n" + RenderRiskLimits(m.riskLimits)

	default:
		return RenderRiskHelp()
	}
}

// handleStrategy processes /strategy subcommands.
func (m *Model) handleStrategy(args []string) (string, bool) {
	if len(args) == 0 {
		return RenderStrategyHelp(), false
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "list", "ls":
		// Reload from disk to get latest state.
		all, _ := strategy.Load()
		m.strategies = all
		return RenderStrategyList(all), false

	case "cancel":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/strategy cancel <id>"), false
		}
		if err := strategy.Cancel(args[1]); err != nil {
			return ErrorStyle.Render("  " + err.Error()), false
		}
		// Reload.
		all, _ := strategy.Load()
		m.strategies = all
		return BotMsgStyle.Render("nick: ") + "Strategy cancelled.", false

	case "twap":
		// /strategy twap ETH buy $2000 4h
		if len(args) < 5 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/strategy twap <SYMBOL> <buy|sell> $<VALUE> <DURATION>"), false
		}
		symbol := strings.ToUpper(args[1])
		side := strings.ToLower(args[2])
		if side != "buy" && side != "sell" {
			return ErrorStyle.Render("  Side must be buy or sell"), false
		}
		valueStr := strings.TrimPrefix(args[3], "$")
		totalValue, err := strconv.ParseFloat(valueStr, 64)
		if err != nil || totalValue <= 0 {
			return ErrorStyle.Render("  Invalid value: ") + args[3], false
		}
		dur, err := strategy.ParseDuration(args[4])
		if err != nil {
			return ErrorStyle.Render("  " + err.Error()), false
		}

		sliceCount, intervalSec := strategy.CalcSlices(dur)
		sliceValue := totalValue / float64(sliceCount)

		s := strategy.TWAPStrategy{
			ID:          randomID(8),
			Symbol:      symbol,
			Side:        side,
			TotalValue:  totalValue,
			Duration:    args[4],
			IntervalSec: intervalSec,
			SliceCount:  sliceCount,
			SliceValue:  sliceValue,
			Executed:    0,
			Status:      "active",
			CreatedAt:   time.Now(),
			NextSliceAt: time.Now().Add(time.Duration(intervalSec) * time.Second),
		}

		if err := strategy.Add(s); err != nil {
			return ErrorStyle.Render("  Failed to save: ") + err.Error(), false
		}

		// Reload.
		all, _ := strategy.Load()
		m.strategies = all

		return BotMsgStyle.Render("nick: ") + "TWAP strategy created.\n" +
			"  " + DimStyle.Render("Symbol: ") + BrandStyle.Render(symbol) +
			DimStyle.Render("  Side: ") + strings.ToUpper(side) +
			DimStyle.Render(fmt.Sprintf("  Value: $%.0f  Duration: %s", totalValue, args[4])) + "\n" +
			"  " + DimStyle.Render(fmt.Sprintf("%d slices × $%.2f every %dm", sliceCount, sliceValue, intervalSec/60)) + "\n" +
			"  " + DimStyle.Render("ID: ") + s.ID, true

	default:
		return RenderStrategyHelp(), false
	}
}

// handleConfig processes /config subcommands.
func (m *Model) handleConfig(args []string) string {
	if len(args) == 0 {
		return RenderConfigHelp()
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "init":
		// Auto-provision: create anonymous account and store API key.
		// Allow re-provisioning with "init force" if user set wrong key.
		if m.cfg.HasAPIKey() && !(len(args) > 1 && args[1] == "force") {
			return BotMsgStyle.Render("nick: ") + "API key already configured. " +
				DimStyle.Render("Use ") + CommandStyle.Render("/config show") +
				DimStyle.Render(" to view, or ") + CommandStyle.Render("/config init force") +
				DimStyle.Render(" to re-provision.")
		}
		name := fmt.Sprintf("nickai-%s", randomID(8))
		baseURL := m.cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://paper.getnick.ai/api/v1"
		}
		result, err := api.CreateAccount(baseURL, name)
		if err != nil {
			return ErrorStyle.Render("  Account creation failed: ") + err.Error()
		}
		m.cfg.SetSecureKey("api_key", result.User.APIKey)
		if err := m.cfg.Save(); err != nil {
			return ErrorStyle.Render("  Failed to save config: ") + err.Error()
		}
		m.client.UpdateConfig(m.cfg)
		return RenderConfigInit(result.User.APIKey, result.User.Name)

	case "show":
		return RenderConfigShow(m.cfg)

	case "test":
		if !m.client.IsConfigured() {
			return ErrorStyle.Render("  No API key configured. ") +
				"Set one first with " + CommandStyle.Render("/config set api_key <key>")
		}
		return RenderConfigTest(m.client)

	case "set":
		if len(args) < 3 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/config set <key> <value>") + "\n" +
				DimStyle.Render("  Keys: api_key, url, anthropic_key, minimax_key, openrouter_key")
		}
		key := strings.ToLower(args[1])
		value := args[2]

		switch key {
		case "api_key", "anthropic_key", "minimax_key", "openrouter_key":
			m.cfg.SetSecureKey(key, value)
		case "url":
			m.cfg.BaseURL = value
		default:
			return ErrorStyle.Render("  Unknown config key: ") + key +
				"\n" + DimStyle.Render("  Valid keys: api_key, url, anthropic_key, minimax_key, openrouter_key")
		}

		if err := m.cfg.Save(); err != nil {
			return ErrorStyle.Render("  Failed to save config: ") + err.Error()
		}
		m.client.UpdateConfig(m.cfg)

		if akKey := m.cfg.AnthropicKeyOrEnv(); akKey != "" {
			if m.agent == nil {
				m.agent = ai.NewAgent(m.client, akKey, m.toolRegistry, m.cfg.Vibe)
			}
		}
		if mmKey := m.cfg.MinimaxKeyOrEnv(); mmKey != "" {
			if m.agent == nil {
				m.agent = ai.NewAgent(m.client, "", m.toolRegistry, m.cfg.Vibe)
			}
			m.agent.SetMinimaxKey(mmKey)
		}
		if orKey := m.cfg.DataKeyOrEnv("openrouter"); orKey != "" {
			if m.agent == nil {
				m.agent = ai.NewAgent(m.client, "", m.toolRegistry, m.cfg.Vibe)
			}
			m.agent.SetOpenRouterKey(orKey)
		}
		m.updatePlaceholder()

		return RenderConfigSet(key, value)

	case "reset":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/config reset api_key") + "\n" +
				DimStyle.Render("  Valid keys: api_key, anthropic_key, minimax_key")
		}
		key := strings.ToLower(args[1])
		switch key {
		case "api_key", "anthropic_key", "minimax_key", "openrouter_key":
			m.cfg.DeleteSecureKey(key)
		default:
			return ErrorStyle.Render("  Unknown config key: ") + key +
				"\n" + DimStyle.Render("  Valid keys: api_key, anthropic_key, minimax_key, openrouter_key")
		}
		if err := m.cfg.Save(); err != nil {
			return ErrorStyle.Render("  Failed to save config: ") + err.Error()
		}
		m.client.UpdateConfig(m.cfg)
		return BotMsgStyle.Render("nick: ") + "Cleared " + CommandStyle.Render(key) + "."

	default:
		return RenderConfigHelp()
	}
}

// handleMCP processes /mcp subcommands.
func (m *Model) handleMCP(args []string) string {
	if len(args) == 0 {
		return RenderMCPHelp()
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "list", "ls":
		return m.renderMCPList()

	case "search":
		query := ""
		if len(args) > 1 {
			query = strings.Join(args[1:], " ")
		}
		results := mcp.SearchRegistry(query)
		if len(results) == 0 {
			return BotMsgStyle.Render("nick: ") + "No servers found for " +
				CommandStyle.Render(query) + "." +
				DimStyle.Render("\n  Try: /mcp search trading, /mcp search defi, /mcp search blockchain")
		}
		return RenderMCPSearchResults(results)

	case "info":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/mcp info <name>")
		}
		entry := mcp.GetEntry(args[1])
		if entry == nil {
			return ErrorStyle.Render("  Unknown server: ") + args[1] +
				DimStyle.Render("\n  Use ") + CommandStyle.Render("/mcp search") +
				DimStyle.Render(" to browse available servers.")
		}
		return RenderMCPInfo(entry)

	case "add":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/mcp add <name> [KEY=value ...]")
		}
		entry := mcp.GetEntry(args[1])
		if entry == nil {
			return ErrorStyle.Render("  Unknown server: ") + args[1] +
				DimStyle.Render("\n  Use ") + CommandStyle.Render("/mcp search") +
				DimStyle.Render(" to browse available servers.")
		}
		// Parse inline KEY=VALUE pairs from remaining args.
		inlineEnv := map[string]string{}
		for _, a := range args[2:] {
			if idx := strings.Index(a, "="); idx > 0 {
				inlineEnv[a[:idx]] = a[idx+1:]
			}
		}
		// Check if required env vars are provided (inline, env, or already in config).
		var missing []string
		for _, key := range entry.EnvKeys {
			if _, ok := inlineEnv[key]; ok {
				continue
			}
			if os.Getenv(key) != "" {
				continue
			}
			missing = append(missing, key)
		}
		if len(missing) > 0 {
			lines := []string{
				BotMsgStyle.Render("nick: ") + "To add " + BrandStyle.Render(entry.DisplayName) + ", provide the required keys:",
				"",
			}
			example := "/mcp add " + entry.Name
			for _, k := range missing {
				hint := "<your-value>"
				if entry.EnvHints != nil {
					if h, ok := entry.EnvHints[k]; ok {
						hint = h
					}
				}
				example += " " + k + "=" + hint
			}
			lines = append(lines, "  "+CommandStyle.Render(example))
			return strings.Join(lines, "\n")
		}
		// Write to mcp.json config.
		err := mcp.AddServerToConfig(entry, inlineEnv)
		if err != nil {
			return ErrorStyle.Render("  Failed to save MCP config: ") + err.Error()
		}
		return BotMsgStyle.Render("nick: ") + "Added " + BrandStyle.Render(entry.DisplayName) + " to " +
			DimStyle.Render("~/.nickai/mcp.json") + "." +
			DimStyle.Render("\n  Restart nickai to activate, or it will load on next launch.")

	case "remove", "rm":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/mcp remove <name>") +
				DimStyle.Render("  or  ") + CommandStyle.Render("/mcp remove all")
		}
		if strings.ToLower(args[1]) == "all" {
			mcpCfg, err := mcp.LoadMCPConfig()
			if err != nil || len(mcpCfg.MCPServers) == 0 {
				return BotMsgStyle.Render("nick: ") + "No MCP servers configured."
			}
			count := len(mcpCfg.MCPServers)
			for name := range mcpCfg.MCPServers {
				_ = mcp.RemoveServerFromConfig(name)
			}
			return BotMsgStyle.Render("nick: ") + fmt.Sprintf("Removed all %d MCP servers.", count) +
				DimStyle.Render("\n  Restart nickai to apply changes.")
		}
		err := mcp.RemoveServerFromConfig(args[1])
		if err != nil {
			return ErrorStyle.Render("  " + err.Error())
		}
		return BotMsgStyle.Render("nick: ") + "Removed " + CommandStyle.Render(args[1]) + " from config." +
			DimStyle.Render("\n  Restart nickai to apply changes.")

	case "quick":
		// Add all servers that need no API keys.
		var added []string
		for _, entry := range mcp.CuratedRegistry {
			if len(entry.EnvKeys) == 0 {
				e := entry
				if err := mcp.AddServerToConfig(&e, nil); err == nil {
					added = append(added, entry.DisplayName)
				}
			}
		}
		if len(added) == 0 {
			return BotMsgStyle.Render("nick: ") + "All free servers already configured."
		}
		lines := []string{
			BotMsgStyle.Render("nick: ") + fmt.Sprintf("Added %d servers (no API keys needed):", len(added)),
			"",
		}
		for _, name := range added {
			lines = append(lines, "  "+StatusIndicator("running")+BrandStyle.Render(name))
		}
		lines = append(lines, "", DimStyle.Render("  Restart nickai to connect them all."))
		return strings.Join(lines, "\n")

	default:
		return RenderMCPHelp()
	}
}

// renderMCPList shows connected MCP servers and their tools.
func (m *Model) renderMCPList() string {
	lines := []string{SecondaryStyle.Render("  MCP Servers\n")}

	// Show connected servers.
	if m.mcpManager != nil && m.mcpManager.ConnectionCount() > 0 {
		for _, conn := range m.mcpManager.Connections() {
			lines = append(lines, "  "+StatusIndicator("running")+BrandStyle.Render(conn.Name)+
				DimStyle.Render(fmt.Sprintf("  (%d tools)", len(conn.Tools))))
			for _, t := range conn.Tools {
				// Truncate long descriptions to keep the list readable.
				desc := t.Description
				if idx := strings.IndexAny(desc, ".\n"); idx > 0 {
					desc = desc[:idx]
				}
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				lines = append(lines, "    "+CommandStyle.Render(t.Name)+
					DimStyle.Render("  "+desc))
			}
		}
	} else {
		lines = append(lines, DimStyle.Render("  No MCP servers connected."))
		lines = append(lines, "")
		lines = append(lines, DimStyle.Render("  Get started:")+
			"\n  "+CommandStyle.Render("/mcp search")+DimStyle.Render("        — browse available servers")+
			"\n  "+CommandStyle.Render("/mcp add <name>")+DimStyle.Render("   — install a server"))
	}

	// Show failed connections.
	if m.mcpManager != nil && len(m.mcpManager.Failed()) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+ErrorStyle.Render("Failed to connect:"))
		for _, f := range m.mcpManager.Failed() {
			lines = append(lines, "  "+StatusIndicator("stopped")+
				WarningStyle.Render(f.Name)+DimStyle.Render("  "+f.Error))
		}
	}

	// Show built-in tool count.
	if m.toolRegistry != nil {
		builtinCount := 0
		for _, entry := range m.toolRegistry.All() {
			if entry.Source == "builtin" {
				builtinCount++
			}
		}
		lines = append(lines, "")
		lines = append(lines, DimStyle.Render(fmt.Sprintf("  + %d built-in tools (get_prices, get_portfolio, get_orders, place_order)", builtinCount)))
	}

	return strings.Join(lines, "\n")
}

// handleCredential processes /credential subcommands.
func (m *Model) handleCredential(args []string) string {
	if len(args) == 0 {
		return RenderCredentialList(m.credStore)
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "list":
		return RenderCredentialList(m.credStore)

	case "add":
		if len(args) < 5 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/credential add <name> <exchange> <api_key> <api_secret>") +
				"\n" + DimStyle.Render("  Exchanges: "+strings.Join(credential.SupportedExchanges(), ", "))
		}
		name := args[1]
		exchange := strings.ToLower(args[2])
		apiKey := args[3]
		apiSecret := args[4]

		if !credential.IsSupportedExchange(exchange) {
			return ErrorStyle.Render("  Unsupported exchange: ") + exchange +
				"\n" + DimStyle.Render("  Supported: "+strings.Join(credential.SupportedExchanges(), ", "))
		}

		m.credStore.Add(credential.Credential{
			Name:      name,
			Exchange:  exchange,
			APIKey:    apiKey,
			APISecret: apiSecret,
		})
		if err := m.credStore.Save(); err != nil {
			return ErrorStyle.Render("  Failed to save credential: ") + err.Error()
		}
		return RenderCredentialAdded(name, exchange)

	case "remove":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/credential remove <name>")
		}
		name := args[1]
		if !m.credStore.Remove(name) {
			return ErrorStyle.Render("  Credential not found: ") + name
		}
		if err := m.credStore.Save(); err != nil {
			return ErrorStyle.Render("  Failed to save: ") + err.Error()
		}
		return RenderCredentialRemoved(name)

	default:
		return ErrorStyle.Render("  Unknown subcommand: ") + sub +
			"\n" + DimStyle.Render("  Usage: /credential <list|add|remove>")
	}
}

// handleWorkflow processes /workflow subcommands.
func (m *Model) handleWorkflow(args []string) string {
	if len(args) == 0 {
		return RenderWorkflowList(m.wfStore)
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "list":
		return RenderWorkflowList(m.wfStore)

	case "create":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/workflow create <path.json>")
		}
		w, err := m.wfStore.CreateFromFile(args[1])
		if err != nil {
			return ErrorStyle.Render("  Failed to create workflow: ") + err.Error()
		}
		if saveErr := m.wfStore.Save(); saveErr != nil {
			return ErrorStyle.Render("  Failed to save: ") + saveErr.Error()
		}
		return RenderWorkflowCreated(w)

	case "run":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/workflow run <name>")
		}
		logs, err := m.wfStore.Run(args[1])
		if err != nil {
			return ErrorStyle.Render("  " + err.Error())
		}
		if saveErr := m.wfStore.Save(); saveErr != nil {
			return ErrorStyle.Render("  Failed to save: ") + saveErr.Error()
		}
		return RenderWorkflowRunning(args[1], logs)

	case "stop":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/workflow stop <name>")
		}
		if err := m.wfStore.Stop(args[1]); err != nil {
			return ErrorStyle.Render("  " + err.Error())
		}
		if saveErr := m.wfStore.Save(); saveErr != nil {
			return ErrorStyle.Render("  Failed to save: ") + saveErr.Error()
		}
		return RenderWorkflowStopped(args[1])

	case "show":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/workflow show <name>")
		}
		w := m.wfStore.Get(args[1])
		if w == nil {
			return ErrorStyle.Render("  Workflow not found: ") + args[1]
		}
		return RenderWorkflowShow(w, m.width)

	case "remove":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/workflow remove <name>")
		}
		name := args[1]
		if !m.wfStore.Remove(name) {
			return ErrorStyle.Render("  Workflow not found: ") + name
		}
		if saveErr := m.wfStore.Save(); saveErr != nil {
			return ErrorStyle.Render("  Failed to save: ") + saveErr.Error()
		}
		return RenderWorkflowRemoved(name)

	case "edit":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/workflow edit <name>")
		}
		return DimStyle.Render("  Tip: use ") + CommandStyle.Render(":e ~/.nickai/workflows.json") +
			DimStyle.Render(" in COMMAND mode (press Esc then :)")

	default:
		return ErrorStyle.Render("  Unknown subcommand: ") + sub +
			"\n" + DimStyle.Render("  Usage: /workflow <list|create|run|stop|show|remove|edit>")
	}
}

// handleLogs processes /logs command.
func (m *Model) handleLogs(args []string) string {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/logs <workflow-name>")
	}
	w := m.wfStore.Get(args[0])
	if w == nil {
		return ErrorStyle.Render("  Workflow not found: ") + args[0]
	}
	return RenderLogs(w)
}

// handleTheme processes /theme command.
func (m *Model) handleTheme(args []string) string {
	if len(args) == 0 {
		// Show available themes.
		var lines []string
		lines = append(lines, SecondaryStyle.Render("  Available Themes\n"))
		current := m.cfg.Theme
		if current == "" {
			current = "default"
		}
		for name := range Themes {
			indicator := "  "
			if name == current {
				indicator = BrandStyle.Render("● ")
			}
			lines = append(lines, "  "+indicator+CommandStyle.Render(name))
		}
		lines = append(lines, "")
		lines = append(lines, DimStyle.Render("  Usage: ")+CommandStyle.Render("/theme <name>"))
		return strings.Join(lines, "\n")
	}

	name := strings.ToLower(args[0])
	t, ok := Themes[name]
	if !ok {
		var names []string
		for n := range Themes {
			names = append(names, n)
		}
		return ErrorStyle.Render("  Unknown theme: ") + name + "\n" +
			DimStyle.Render("  Available: "+strings.Join(names, ", "))
	}

	ApplyTheme(t)
	m.refreshInputStyles()
	m.cfg.Theme = name
	_ = m.cfg.Save()

	m.statusFlash = "Theme: " + name
	m.statusFlashExpiry = time.Now().Add(2 * time.Second)

	return BotMsgStyle.Render("nick: ") + "Theme set to " + BrandStyle.Render(name) + "."
}

// handleModel processes /model command.
func (m *Model) handleModel(args []string) string {
	if len(args) == 0 {
		// Show available models.
		var lines []string
		lines = append(lines, SecondaryStyle.Render("  Available Models\n"))
		currentModel := "claude-sonnet"
		if m.agent != nil {
			currentModel = m.agent.ModelID()
		}
		for _, opt := range ai.AvailableModels {
			indicator := "  "
			if opt.ID == currentModel {
				indicator = BrandStyle.Render("● ")
			}
			freeTag := ""
			if opt.Free {
				freeTag = lipgloss.NewStyle().Foreground(ColorPrimary).Render(" [FREE]")
			}
			lines = append(lines, "  "+indicator+CommandStyle.Render(padRight(opt.ID, 18))+
				DimStyle.Render(opt.Name)+freeTag)
		}
		lines = append(lines, "")
		lines = append(lines, DimStyle.Render("  Usage: ")+CommandStyle.Render("/model <id>"))
		lines = append(lines, DimStyle.Render("  Custom: ")+CommandStyle.Render("/model <openrouter-slug>")+DimStyle.Render("  (e.g. openai/gpt-4o-mini)"))
		return strings.Join(lines, "\n")
	}

	modelID := strings.ToLower(args[0])

	if m.agent == nil {
		// Create agent if we have any key.
		anthKey := m.cfg.AnthropicKeyOrEnv()
		mmKey := m.cfg.MinimaxKeyOrEnv()
		orKey := m.cfg.DataKeyOrEnv("openrouter")
		if anthKey == "" && mmKey == "" && orKey == "" {
			return ErrorStyle.Render("  No API keys configured.") + "\n" +
				DimStyle.Render("  Set one with ") +
				CommandStyle.Render("/config set anthropic_key <key>") +
				DimStyle.Render(" or ") +
				CommandStyle.Render("/config set openrouter_key <key>")
		}
		if anthKey != "" {
			m.agent = ai.NewAgent(m.client, anthKey, m.toolRegistry, m.cfg.Vibe)
		} else {
			m.agent = ai.NewAgent(m.client, "", m.toolRegistry, m.cfg.Vibe)
		}
		if mmKey != "" {
			m.agent.SetMinimaxKey(mmKey)
		}
		if orKey != "" {
			m.agent.SetOpenRouterKey(orKey)
		}
		m.updatePlaceholder()
	}

	if err := m.agent.SetModel(modelID); err != nil {
		return ErrorStyle.Render("  " + err.Error())
	}

	m.cfg.Model = modelID
	_ = m.cfg.Save()

	// Find model name for display.
	name := modelID
	for _, opt := range ai.AvailableModels {
		if opt.ID == modelID {
			name = opt.Name
			break
		}
	}

	m.statusFlash = "Model: " + name
	m.statusFlashExpiry = time.Now().Add(2 * time.Second)

	result := BotMsgStyle.Render("nick: ") + "Switched to " + BrandStyle.Render(name) + "."

	// Warn if non-Anthropic model (no tool use).
	if m.agent.Provider() != ai.ProviderAnthropic {
		result += "\n" + WarningStyle.Render("  ⚠ Tools are unavailable with this model.") +
			DimStyle.Render(" Trading, portfolio, and MCP tools require an Anthropic model.")
	}

	return result
}

// handleVibe processes /vibe commands (list, set).
func (m *Model) handleVibe(args []string) string {
	allVibes := personality.AllVibes()

	// Determine current vibe.
	currentID := personality.DefaultVibeID
	if m.cfg.Vibe != "" {
		currentID = m.cfg.Vibe
	}

	// No args or "list" → show all vibes.
	if len(args) == 0 || strings.ToLower(args[0]) == "list" {
		var sb strings.Builder
		sb.WriteString(BotMsgStyle.Render("nick: ") + "Pick your vibe:\n\n")
		for _, v := range allVibes {
			marker := "  "
			if v.ID == currentID {
				marker = "▸ "
			}
			line := fmt.Sprintf("%s%s %s — \"%s\"", marker, v.Emoji, v.Name, v.Tagline)
			if v.ID == currentID {
				line = lipgloss.NewStyle().Bold(true).Render(line)
			}
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n" + DimStyle.Render("Usage: ") + CommandStyle.Render("/vibe set <id>") +
			DimStyle.Render("  IDs: degen, quant, zen, hype, sensei, degen-bets"))
		return sb.String()
	}

	// "set <id>"
	if strings.ToLower(args[0]) == "set" && len(args) >= 2 {
		id := strings.ToLower(args[1])
		vibe := personality.GetVibe(id)
		if vibe.ID != id {
			return BotMsgStyle.Render("nick: ") + "Unknown vibe " + CommandStyle.Render(id) +
				". Try: degen, quant, zen, hype, sensei, degen-bets"
		}
		m.cfg.Vibe = id
		_ = m.cfg.Save()
		if m.agent != nil {
			m.agent.SetVibe(id)
		}
		m.welcomeDirty = true
		return BotMsgStyle.Render("nick: ") + vibe.Emoji + " " + lipgloss.NewStyle().Bold(true).Render(vibe.Name) +
			" activated. " + DimStyle.Render("\""+vibe.Tagline+"\"")
	}

	return BotMsgStyle.Render("nick: ") + "Usage: " + CommandStyle.Render("/vibe") +
		" or " + CommandStyle.Render("/vibe set <id>")
}
