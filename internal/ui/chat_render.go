package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/commands"
	"github.com/nickai/cli/internal/guidance"
)

// renderBootSequence renders the animated boot screen.
func (m Model) renderBootSequence() string {
	var lines []string

	// Add top padding to roughly center vertically.
	topPad := (m.height - 18) / 3 // 18 ≈ content lines
	if topPad < 1 {
		topPad = 1
	}
	for i := 0; i < topPad; i++ {
		lines = append(lines, "")
	}

	logoStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	checkStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
	pad := "   " // left padding for all content

	// Phase 1: Logo lines appearing one by one (frames 1..6).
	visibleLogoLines := m.bootFrame
	if visibleLogoLines > len(bootLogo) {
		visibleLogoLines = len(bootLogo)
	}
	for i := 0; i < visibleLogoLines; i++ {
		lines = append(lines, pad+logoStyle.Render(bootLogo[i]))
	}

	// Pad remaining logo space so content below doesn't jump.
	for i := visibleLogoLines; i < len(bootLogo); i++ {
		lines = append(lines, "")
	}
	lines = append(lines, "")

	// Phase 2: Tagline types out character by character.
	taglineStart := len(bootLogo)
	if m.bootFrame > taglineStart {
		charsVisible := m.bootFrame - taglineStart
		if charsVisible > len(m.bootTagline) {
			charsVisible = len(m.bootTagline)
		}
		taglineText := m.bootTagline[:charsVisible]
		cursor := ""
		if charsVisible < len(m.bootTagline) {
			cursor = lipgloss.NewStyle().Foreground(ColorSecondary).Render("█")
		}
		lines = append(lines, pad+DimStyle.Render("  \"")+DimStyle.Render(taglineText)+cursor+DimStyle.Render("\""))
	} else {
		lines = append(lines, "")
	}
	lines = append(lines, "")

	// Phase 3: Boot checks (after tagline completes).
	checksStart := taglineStart + len(m.bootTagline) + 1
	type bootCheck struct {
		label string
		ok    bool
	}

	hasAPI := m.client.IsConfigured()
	hasAnthropicKey := m.cfg.AnthropicKeyOrEnv() != ""
	hasMCP := m.mcpManager != nil && m.mcpManager.ConnectionCount() > 0

	checks := []bootCheck{
		{"API connected", hasAPI},
		{"Paper trading active", hasAPI},
		{"AI agent ready", hasAnthropicKey},
	}
	if hasMCP {
		checks = append(checks, bootCheck{
			fmt.Sprintf("MCP servers (%d)", m.mcpManager.ConnectionCount()), true,
		})
	}
	checks = append(checks, bootCheck{"Vim mode enabled", true})

	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9900"))

	for i, check := range checks {
		checkFrame := checksStart + i*2
		if m.bootFrame >= checkFrame+1 {
			if check.ok {
				lines = append(lines, pad+"  "+checkStyle.Render("✓ "+check.label))
			} else {
				lines = append(lines, pad+"  "+warnStyle.Render("○ "+check.label)+DimStyle.Render("  (not configured)"))
			}
		} else if m.bootFrame >= checkFrame {
			spinIdx := m.bootFrame % len(spinnerFrames)
			spinner := lipgloss.NewStyle().Foreground(ColorSecondary).Render(spinnerFrames[spinIdx])
			lines = append(lines, pad+"  "+spinner+" "+DimStyle.Render(check.label+"..."))
		}
	}

	// Phase 4: Ready message — context-aware.
	readyFrame := checksStart + len(checks)*2
	if m.bootFrame >= readyFrame {
		lines = append(lines, "")
		if !hasAPI {
			lines = append(lines, pad+"  "+DimStyle.Render("Run ")+CommandStyle.Render("/config init")+DimStyle.Render(" to get started."))
		} else {
			lines = append(lines, pad+DimStyle.Render("  Ready. Type /help or just ask me anything."))
		}
	}

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(content)
}

// renderStatusRight returns the right-aligned top bar status (MCP, memories, model, risk).
func (m Model) renderStatusRight() string {
	// Show status flash if active.
	if m.statusFlash != "" && time.Now().Before(m.statusFlashExpiry) {
		return lipgloss.NewStyle().Foreground(ColorPrimary).Render(m.statusFlash)
	}

	var parts []string

	// MCP connections count.
	if m.mcpManager != nil && m.mcpManager.ConnectionCount() > 0 {
		parts = append(parts,
			lipgloss.NewStyle().Foreground(ColorPrimary).Render("●")+
				DimStyle.Render(fmt.Sprintf(" %d MCP", m.mcpManager.ConnectionCount())))
	}

	// Memory count.
	if m.memoryStore != nil && len(m.memoryStore.Entries) > 0 {
		parts = append(parts,
			lipgloss.NewStyle().Foreground(ColorPrimary).Render("●")+
				DimStyle.Render(fmt.Sprintf(" %d memories", len(m.memoryStore.Entries))))
	}

	// Journey level + mission completion snapshot.
	ctx := m.guidanceCtx
	if ctx.HasAPIKey != m.client.IsConfigured() || ctx.HasAIKey != (m.agent != nil) {
		ctx = m.buildGuidanceCtx()
	}
	xp, lvl, rank := guidance.Experience(ctx)
	done, total := guidance.JourneyProgress(ctx)
	parts = append(parts, DimStyle.Render(fmt.Sprintf("LV%d %s %d/%d %dXP", lvl, rank, done, total, xp)))

	// Active model.
	if m.agent != nil {
		parts = append(parts, DimStyle.Render(m.agent.ModelID()))
	}

	// Risk guardrail indicator.
	if m.riskLimits != nil && !m.riskLimits.IsEmpty() {
		parts = append(parts, lipgloss.NewStyle().Foreground(ColorWarning).Render("🛡"))
	}

	if len(parts) == 0 {
		return ""
	}
	return DimStyle.Render(strings.Join(parts, "  "))
}

// statusIndicators returns the right-aligned status string (API dot, model, alerts).
func (m Model) statusIndicators() string {
	connDot := lipgloss.NewStyle().Foreground(ColorPrimary).Render("●")
	if !m.client.IsConfigured() {
		connDot = DimStyle.Render("○")
	}

	modelName := ""
	if m.agent != nil {
		modelName = DimStyle.Render(" ┃ ") +
			lipgloss.NewStyle().Foreground(ColorSecondary).Render(m.agent.ModelID())
	}

	alertPart := ""
	statusParts := []string{}
	if len(m.alerts) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("⚡%d", len(m.alerts)))
	}
	if len(m.triggers) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("⏱%d", len(m.triggers)))
	}
	activeStrats := 0
	for _, s := range m.strategies {
		if s.Status == "active" {
			activeStrats++
		}
	}
	if activeStrats > 0 {
		statusParts = append(statusParts, fmt.Sprintf("📊%d", activeStrats))
	}
	if m.riskLimits != nil && !m.riskLimits.IsEmpty() {
		statusParts = append(statusParts, "🛡")
	}
	if len(statusParts) > 0 {
		alertPart = DimStyle.Render(" ┃ ") +
			lipgloss.NewStyle().Foreground(ColorWarning).Render(strings.Join(statusParts, " "))
	}

	return connDot + modelName + alertPart
}

// renderInputBar renders the bottom input area based on vim mode.
func (m Model) renderInputBar() string {
	var borderColor lipgloss.Color
	var content string

	switch m.vimMode {
	case ModeInsert:
		// No badge — clean input with status info right-aligned.
		borderColor = ColorDim
		inputView := m.textInput.View()
		status := m.statusIndicators()
		statusWidth := lipgloss.Width(status)
		inputWidth := m.width - statusWidth - 4
		if inputWidth < 20 {
			inputWidth = 20
		}
		leftSide := lipgloss.NewStyle().Width(inputWidth).Render(inputView)
		content = leftSide + status

	case ModeNormal:
		borderColor = ColorPrimary
		badge := lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Render(" NORMAL ")

		hints := DimStyle.Render("i") + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(":insert") +
			DimStyle.Render("  :") + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("cmd") +
			DimStyle.Render("  /") + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("search") +
			DimStyle.Render("  q") + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(":quit")

		// Show search match count if a pattern is active.
		searchHint := ""
		if m.searchPattern != "" && len(m.searchMatches) > 0 {
			searchHint = fmt.Sprintf("  /%s [%d/%d]", m.searchPattern, m.searchCurrent+1, len(m.searchMatches))
		} else if m.searchPattern != "" {
			searchHint = fmt.Sprintf("  /%s [0/0]", m.searchPattern)
		}

		status := m.statusIndicators()
		middle := hints + lipgloss.NewStyle().Foreground(ColorWarning).Render(searchHint) + "  " + status
		middleWidth := m.width - 14
		if middleWidth < 0 {
			middleWidth = 0
		}
		rightAligned := lipgloss.NewStyle().Width(middleWidth).Align(lipgloss.Right).Render(middle)
		content = badge + " " + rightAligned

	case ModeCommand:
		borderColor = ColorSecondary
		badge := lipgloss.NewStyle().
			Background(ColorSecondary).
			Foreground(ColorWhite).
			Bold(true).
			Render(" COMMAND ")

		bufferStyle := lipgloss.NewStyle().Foreground(ColorWhite)
		content = badge + " " + bufferStyle.Render(":"+m.commandBuffer) +
			lipgloss.NewStyle().Foreground(ColorWhite).Render("█")

	case ModeSearch:
		borderColor = ColorWarning
		badge := lipgloss.NewStyle().
			Background(ColorWarning).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Render(" SEARCH ")

		bufferStyle := lipgloss.NewStyle().Foreground(ColorWarning)
		content = badge + " " + bufferStyle.Render("/"+m.searchBuffer) +
			lipgloss.NewStyle().Foreground(ColorWarning).Render("█")

	case ModeConfirm:
		borderColor = ColorWarning
		badge := lipgloss.NewStyle().
			Background(ColorWarning).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Render(" CONFIRM ")
		hint := lipgloss.NewStyle().Foreground(ColorWarning).
			Render("  y: confirm  n: cancel")
		content = badge + hint
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(borderColor).
		Width(m.width).
		Padding(0, 1).
		Render(content)
}

func (m *Model) updateViewport() {
	if len(m.messages) == 0 {
		content := m.welcomeContent()
		m.viewContent = content
		m.viewport.SetContent(content)
		return
	}

	// If loading, update the spinner on the last message.
	if m.loading && len(m.messages) > 0 && !m.messages[len(m.messages)-1].isUser {
		frame := spinnerFrames[m.loadingFrame%len(spinnerFrames)]
		text := thinkingTexts[(m.loadingFrame/15)%len(thinkingTexts)]
		if m.streaming && m.loadingFrame > 60 {
			text += DimStyle.Render(" (streaming)")
		}
		m.messages[len(m.messages)-1].content = BotMsgStyle.Render("nick: ") + frame + " " + text
	}

	welcome := m.welcomeContent()
	var parts []string
	parts = append(parts, welcome)
	for _, msg := range m.messages {
		// Apply left-border accent bars.
		if msg.isUser {
			parts = append(parts, UserMsgBar(msg.content))
		} else {
			parts = append(parts, BotMsgBar(msg.content))
		}
		parts = append(parts, "")
	}
	content := strings.Join(parts, "\n")
	m.viewContent = content
	atBottom := m.viewport.AtBottom()
	m.viewport.SetContent(content)
	if atBottom {
		m.viewport.GotoBottom()
	}
}

// apiLoadingText returns the loading message for a given command type.
func apiLoadingText(t commands.CommandType) string {
	switch t {
	case commands.TypePrice:
		return "Fetching prices..."
	case commands.TypeStatus:
		return "Loading portfolio..."
	case commands.TypeOrders:
		return "Loading orders..."
	case commands.TypeBuy, commands.TypeSell:
		return "Executing trade..."
	case commands.TypeSnapshot:
		return "Loading snapshot..."
	case commands.TypeMarket:
		return "Fetching market data..."
	case commands.TypePnl:
		return "Calculating P&L..."
	case commands.TypeHistory:
		return "Loading trade history..."
	case commands.TypeChart:
		return "Generating chart..."
	default:
		return "Loading..."
	}
}
