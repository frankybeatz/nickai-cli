package ui

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/guidance"
	"github.com/nickai/cli/internal/journal"
	"github.com/nickai/cli/internal/risk"
	"github.com/nickai/cli/internal/safefile"
	"github.com/nickai/cli/internal/tools"
)

// buildPortfolioSummary creates rich portfolio context for the AI agent.
func buildPortfolioSummary(portfolio *api.Portfolio) string {
	if portfolio == nil {
		return ""
	}

	startingCapital := 100000.0
	pnl := portfolio.TotalValue - startingCapital
	pnlPct := (pnl / startingCapital) * 100
	cashPct := 0.0
	if portfolio.TotalValue > 0 {
		cashPct = (portfolio.AvailableCash / portfolio.TotalValue) * 100
	}

	var sb strings.Builder
	sb.WriteString("CURRENT PORTFOLIO STATE:\n")
	sb.WriteString(fmt.Sprintf("Total value: $%.2f | Cash: $%.2f (%.0f%%) | P&L: %+.2f (%+.1f%%)\n",
		portfolio.TotalValue, portfolio.AvailableCash, cashPct, pnl, pnlPct))

	if len(portfolio.Assets) > 0 {
		sb.WriteString("Positions: ")
		for i, pos := range portfolio.Assets {
			if pos.Quantity <= 0 {
				continue
			}
			sym := strings.TrimSuffix(pos.Symbol, "USDT")
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s %.4f ($%.0f)", sym, pos.Quantity, pos.Value))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("No open positions.\n")
	}

	// Proactive hints for the AI based on portfolio state.
	if cashPct > 50 {
		sb.WriteString("Note: User has significant idle cash — consider suggesting entries when discussing markets.\n")
	}
	if pnlPct < -5 {
		sb.WriteString("Note: User is down significantly — be mindful of risk management.\n")
	}
	if pnlPct > 10 {
		sb.WriteString("Note: User is performing well — consider suggesting profit-taking or trailing stops.\n")
	}

	return sb.String()
}

// trackRecentCommand adds a command summary to the ring buffer and updates AI context.
func (m *Model) trackRecentCommand(summary string) {
	m.recentCommands = append(m.recentCommands, summary)
	if len(m.recentCommands) > 3 {
		m.recentCommands = m.recentCommands[len(m.recentCommands)-3:]
	}
	if m.agent != nil {
		m.agent.SetRecentActivity("Recent: " + strings.Join(m.recentCommands, " | "))
	}
}

// streamOriginHints returns next-step hints based on stream origin and user context.
func (m Model) streamOriginHints(origin string) string {
	ctx := m.buildGuidanceCtx()
	hints := guidance.NextStepAfterCommand(origin, ctx)
	if len(hints) == 0 {
		return ""
	}
	return NextSteps(hints...)
}

// --- Helper: streaming ---

// waitForStreamToken returns a tea.Cmd that reads the next token from a
// streaming channel. Returns nil when the channel is closed.
func waitForStreamToken(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		token, ok := <-ch
		if !ok {
			return nil
		}
		return aiStreamMsg{token: token}
	}
}

// waitForConfirmation listens for AI trade confirmation requests.
// Runs concurrently with the streaming agent — when the agent's place_order
// blocks on ConfirmCh, this returns an aiTradeConfirmMsg to the Bubbletea loop.
func waitForConfirmation(ch <-chan tools.ConfirmRequest) tea.Cmd {
	return func() tea.Msg {
		req, ok := <-ch
		if !ok {
			return nil
		}
		return aiTradeConfirmMsg{req: req}
	}
}

// randomID generates a random alphanumeric string of length n.
func randomID(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	crand.Read(b)
	for i := range b {
		b[i] = chars[b[i]%byte(len(chars))]
	}
	return string(b)
}

// --- Helper: add bot message ---

func (m *Model) addBotMessage(content string) {
	m.messages = append(m.messages, message{
		content: content,
		isUser:  false,
	})
}

// requireAgent returns an error message if no AI agent is configured, or empty string if OK.
func (m *Model) requireAgent() string {
	if m.agent != nil {
		return ""
	}
	return BotMsgStyle.Render("nick: ") +
		"I need an Anthropic API key to use AI features. Set one with " +
		CommandStyle.Render("/config set anthropic_key <key>") +
		" or " + DimStyle.Render("export ANTHROPIC_API_KEY=...")
}

// refreshInputStyles updates the textInput styles to match the current theme.
// Must be called after ApplyTheme().
func (m *Model) refreshInputStyles() {
	m.textInput.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	m.textInput.TextStyle = lipgloss.NewStyle().Foreground(ColorWhite)
	m.textInput.PlaceholderStyle = lipgloss.NewStyle().Foreground(ColorDim)
}

// updatePlaceholder sets the input placeholder based on current state.
func (m *Model) updatePlaceholder() {
	switch {
	case m.agent == nil:
		m.textInput.Placeholder = "No API key configured — type /config set to get started"
	case m.mcpManager != nil && m.mcpManager.ConnectionCount() > 0:
		m.textInput.Placeholder = "Ask NickAI anything, type / for commands, or use MCP tools..."
	default:
		m.textInput.Placeholder = "Ask NickAI anything or type / for commands..."
	}
}

// riskPromptFromLimits builds the risk info string for the AI system prompt.
func riskPromptFromLimits(limits *risk.RiskLimits) string {
	if limits == nil || limits.IsEmpty() {
		return ""
	}
	var parts []string
	parts = append(parts, "RISK GUARDRAILS ARE ACTIVE. Before placing trades, be aware of these limits:")
	if limits.MaxOrderValue > 0 {
		parts = append(parts, fmt.Sprintf("- Maximum single order value: $%.0f", limits.MaxOrderValue))
	}
	if limits.MaxPositionPct > 0 {
		parts = append(parts, fmt.Sprintf("- Maximum position size: %.0f%% of portfolio", limits.MaxPositionPct))
	}
	if limits.DailyLossPct > 0 {
		parts = append(parts, fmt.Sprintf("- Daily loss limit: %.0f%% (all trades blocked if exceeded)", limits.DailyLossPct))
	}
	parts = append(parts, "If a trade is rejected by risk limits, explain the reason to the user and suggest an alternative that fits within limits.")
	return strings.Join(parts, "\n")
}

// waitForJournalEntry listens for journal entries from tool executors.
func waitForJournalEntry(ch <-chan journal.JournalEntry) tea.Cmd {
	return func() tea.Msg {
		entry, ok := <-ch
		if !ok {
			return nil
		}
		return journalEntryMsg{entry: entry}
	}
}

// --- Session persistence ---

// historyFilePath returns ~/.nickai/input_history.json.
func historyFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/.nickai/input_history.json"
}

// loadInputHistory reads persisted input history from disk.
func loadInputHistory() []string {
	path := historyFilePath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var history []string
	if err := json.Unmarshal(data, &history); err != nil {
		return nil
	}
	// Keep last 100.
	if len(history) > 100 {
		history = history[len(history)-100:]
	}
	return history
}

// saveInputHistory persists input history to disk.
func saveInputHistory(history []string) {
	path := historyFilePath()
	if path == "" {
		return
	}
	// Keep last 100.
	if len(history) > 100 {
		history = history[len(history)-100:]
	}
	data, err := json.Marshal(history)
	if err != nil {
		return
	}
	_ = os.MkdirAll(home()+"/.nickai", 0700)
	_ = safefile.AtomicWrite(path, data, 0600)
}

func home() string {
	h, _ := os.UserHomeDir()
	return h
}

// welcomeContent returns the welcome screen, using cache when available.
func (m *Model) welcomeContent() string {
	if m.cachedWelcome != "" && !m.welcomeDirty {
		return m.cachedWelcome
	}
	memCount := 0
	if m.memoryStore != nil {
		memCount = len(m.memoryStore.Entries)
	}
	mcpCount := 0
	if m.mcpManager != nil {
		mcpCount = len(m.mcpManager.Connections())
	}
	ctx := m.buildGuidanceCtx()
	stage := guidance.DetectStage(ctx)
	actions := guidance.ActionsForStage(stage, ctx)
	m.cachedWelcome = RenderWelcome(m.width, stage, actions, m.cfg.Vibe, ctx, memCount, mcpCount)
	m.welcomeDirty = false
	return m.cachedWelcome
}

// buildGuidanceCtx constructs the guidance context from cached model state.
// IMPORTANT: This is called from View() — must never make blocking API calls.
func (m Model) buildGuidanceCtx() guidance.StageContext {
	ctx := guidance.StageContext{
		HasAPIKey:     m.client.IsConfigured(),
		HasAIKey:      m.agent != nil,
		TradeCount:    m.cachedTradeCount,
		HasAnalyzed:   m.cachedHasAnalyzed,
		HasBacktested: m.cachedHasBacktested,
	}
	if m.mcpManager != nil {
		ctx.MCPCount = m.mcpManager.ConnectionCount()
	}
	if m.memoryStore != nil {
		ctx.MemoryCount = len(m.memoryStore.Entries)
	}
	if m.cachedPortfolio != nil {
		ctx.PortfolioValue = m.cachedPortfolio.TotalValue
		ctx.CashBalance = m.cachedPortfolio.AvailableCash
		for _, pos := range m.cachedPortfolio.Assets {
			if pos.Quantity > 0 {
				ctx.TopPositions = append(ctx.TopPositions, strings.TrimSuffix(pos.Symbol, "USDT"))
			}
		}
	}
	return ctx
}

// refreshGuidanceCaches updates cached data used by buildGuidanceCtx.
// Called from ticker updates and after trades — safe to make API calls here.
func (m *Model) refreshGuidanceCaches() {
	if m.client.IsConfigured() {
		if orders, err := m.client.GetOrders(); err == nil {
			count := 0
			for _, o := range orders {
				if o.Status == "filled" || o.Status == "completed" {
					count++
				}
			}
			m.cachedTradeCount = count
		}
	}
	if m.memoryStore != nil {
		for _, e := range m.memoryStore.Entries {
			for _, tag := range e.Tags {
				if tag == "analyzed" {
					m.cachedHasAnalyzed = true
				}
				if tag == "backtested" {
					m.cachedHasBacktested = true
				}
			}
		}
	}

	m.updateJourneyContext(true)
}

// updateJourneyContext recomputes journey context, emits stage-up cards, and
// refreshes AI guidance context. When emitMilestone is true, stage increases
// create an in-chat "quest complete" event.
func (m *Model) updateJourneyContext(emitMilestone bool) {
	ctx := m.buildGuidanceCtx()
	stage := guidance.DetectStage(ctx)
	prev := m.lastJourneyStage

	m.guidanceCtx = ctx
	m.welcomeDirty = true

	if prev == "" {
		m.lastJourneyStage = stage
	} else if guidance.StageOrdinal(stage) > guidance.StageOrdinal(prev) {
		m.lastJourneyStage = stage
		m.welcomeDirty = true
		if emitMilestone && !m.booting {
			m.addBotMessage(renderStageUpCard(prev, stage, ctx))
			m.updateViewport()
		}
	} else {
		m.lastJourneyStage = stage
	}

	if m.agent != nil {
		m.agent.SetGuidanceContext(buildGuidancePrompt(stage, ctx))
	}
}

func renderStageUpCard(prev guidance.Stage, next guidance.Stage, ctx guidance.StageContext) string {
	xp, level, rank := guidance.Experience(ctx)
	ch := guidance.StageChallenge(next, ctx)

	lines := []string{
		BrandStyle.Render("  QUEST COMPLETE"),
		lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).
			Render("  Stage Up: " + guidance.StageLabel(prev) + " → " + guidance.StageLabel(next)),
		DimStyle.Render(fmt.Sprintf("  Level %d · %s · %d XP", level, rank, xp)),
		"",
		SecondaryStyle.Render("  New Mission: " + ch.Title),
		DimStyle.Render("  " + ch.Goal),
		CommandStyle.Render("  "+ch.Command) + "  " + DimStyle.Render(ch.Reward),
	}
	return strings.Join(lines, "\n")
}

// buildGuidancePrompt creates a context string telling Nick where the user is in their journey.
func buildGuidancePrompt(stage guidance.Stage, ctx guidance.StageContext) string {
	var sb strings.Builder
	xp, level, rank := guidance.Experience(ctx)
	ch := guidance.StageChallenge(stage, ctx)
	sb.WriteString("USER JOURNEY CONTEXT:\n")
	sb.WriteString(fmt.Sprintf("Stage: %s | Trades: %d | MCP servers: %d | Memories: %d\n", stage, ctx.TradeCount, ctx.MCPCount, ctx.MemoryCount))
	sb.WriteString(fmt.Sprintf("Progression: Level %d (%s), %d XP\n", level, rank, xp))
	sb.WriteString(fmt.Sprintf("Current mission: %s | Goal: %s | Action: %s\n", ch.Title, ch.Goal, ch.Command))

	// Tell Nick what the user hasn't discovered yet — so he can naturally introduce features.
	var unused []string
	if ctx.TradeCount == 0 {
		unused = append(unused, "hasn't placed a trade yet — encourage trying /buy")
	}
	if !ctx.HasAnalyzed {
		unused = append(unused, "hasn't used /analyze — mention it when discussing technicals")
	}
	if !ctx.HasBacktested {
		unused = append(unused, "hasn't tried /backtest — suggest it when discussing strategies")
	}
	if ctx.MCPCount == 0 {
		unused = append(unused, "has no MCP tools — suggest /mcp quick for free market data")
	}
	if len(unused) > 0 {
		sb.WriteString("Undiscovered features: ")
		sb.WriteString(strings.Join(unused, "; "))
		sb.WriteString("\nNaturally weave these suggestions into conversation — don't dump a list.\n")
	}

	if len(ctx.TopPositions) > 0 {
		sb.WriteString("Open positions: " + strings.Join(ctx.TopPositions, ", ") + "\n")
	}

	return sb.String()
}

// waitForWSPrice returns a Bubbletea command that waits for the next
// price update from the websocket channel.
func waitForWSPrice(ch <-chan wsPriceMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return wsDisconnectedMsg{}
		}
		return msg
	}
}
