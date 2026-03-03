package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/nickai/cli/internal/api"
)

// renderDashboard renders the Bloomberg-style split-pane dashboard.
func (m Model) renderDashboard() string {
	// Terminal too narrow for split panes — show a message.
	if m.width < 55 {
		msg := lipgloss.NewStyle().Foreground(ColorDim).Render(
			"\n  Terminal too narrow for dashboard.\n  Resize to 55+ columns or press Esc.\n")
		return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(msg)
	}

	// Calculate pane dimensions ensuring they never exceed terminal width.
	rightWidth := 26
	if m.width < 80 {
		rightWidth = 20
	}
	leftWidth := m.width - rightWidth - 3 // 3 for borders/separator
	// Ensure total doesn't exceed terminal.
	if leftWidth+rightWidth+3 > m.width {
		leftWidth = m.width - rightWidth - 3
	}
	if leftWidth < 28 {
		leftWidth = 28
		rightWidth = m.width - leftWidth - 3
	}

	// Height budget: subtract top bar (1) + input bar (1) + borders (3)
	innerHeight := m.height - 5
	if innerHeight < 10 {
		innerHeight = 10
	}
	marketHeight := innerHeight / 2
	chatHeight := innerHeight - marketHeight

	// Build panes.
	marketPane := m.dashboardMarketPane(leftWidth-2, marketHeight-2)
	chatPane := m.dashboardChatPane(leftWidth-2, chatHeight-2)
	portfolioPane := m.dashboardPortfolioPane(rightWidth-2, innerHeight-2)

	// Style the panes.
	paneStyle := func(width, height int) lipgloss.Style {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDim).
			Width(width).
			Height(height)
	}

	marketBox := paneStyle(leftWidth, marketHeight).Render(marketPane)
	chatBox := paneStyle(leftWidth, chatHeight).Render(chatPane)
	portfolioBox := paneStyle(rightWidth, innerHeight).Render(portfolioPane)

	leftColumn := lipgloss.JoinVertical(lipgloss.Left, marketBox, chatBox)
	layout := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, portfolioBox)

	return layout
}

// dashboardMarketPane renders the market data pane.
func (m Model) dashboardMarketPane(width, height int) string {
	var lines []string

	header := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("  MARKET")
	lines = append(lines, header)
	lines = append(lines, DimStyle.Render(strings.Repeat("─", width)))

	if len(m.tickerPrices) > 0 {
		for _, p := range m.tickerPrices {
			sym := strings.TrimSuffix(p.Symbol, "USDT")
			arrow := DimStyle.Render("─")
			priceColor := lipgloss.NewStyle().Foreground(ColorWhite)
			if prev, ok := m.prevTickerPrices[p.Symbol]; ok {
				if p.Price > prev {
					arrow = lipgloss.NewStyle().Foreground(ColorPrimary).Render("▲")
					priceColor = lipgloss.NewStyle().Foreground(ColorPrimary)
				} else if p.Price < prev {
					arrow = lipgloss.NewStyle().Foreground(ColorError).Render("▼")
					priceColor = lipgloss.NewStyle().Foreground(ColorError)
				}
			}
			line := fmt.Sprintf("  %s %-5s %s",
				arrow,
				sym,
				priceColor.Render(formatPrice(p.Price)))
			lines = append(lines, line)
		}
	} else {
		lines = append(lines, DimStyle.Render("  Waiting for price data..."))
	}

	// Fear & Greed if we have space.
	if height > 8 {
		lines = append(lines, "")
		lines = append(lines, DimStyle.Render("  Fear & Greed: ")+BrandStyle.Render("--"))
	}

	// Pad to fill height.
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// dashboardChatPane renders the last N messages with input.
func (m Model) dashboardChatPane(width, height int) string {
	var lines []string

	header := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("  CHAT")
	lines = append(lines, header)
	lines = append(lines, DimStyle.Render(strings.Repeat("─", width)))

	// Show last N messages that fit.
	msgLines := height - 3 // header + divider + input line
	if msgLines < 1 {
		msgLines = 1
	}

	var msgParts []string
	for _, msg := range m.messages {
		prefix := BotMsgStyle.Render("nick: ")
		if msg.isUser {
			prefix = UserMsgStyle.Render("you: ")
		}
		// Use raw content for dashboard — strip ANSI first, then re-style as dim.
		content := ansi.Strip(msg.content)
		// Remove any "nick: " or "you: " prefix from the raw content.
		content = strings.TrimPrefix(content, "nick: ")
		content = strings.TrimPrefix(content, "you: ")
		content = strings.TrimPrefix(content, "nick:")
		content = strings.TrimPrefix(content, "you:")
		content = strings.TrimSpace(content)
		// Take first line only for dashboard.
		if idx := strings.IndexByte(content, '\n'); idx >= 0 {
			content = content[:idx]
		}
		// ANSI-aware truncation.
		maxWidth := width - 10
		if maxWidth < 10 {
			maxWidth = 10
		}
		if ansi.StringWidth(content) > maxWidth {
			content = ansi.Truncate(content, maxWidth, "...")
		}
		msgParts = append(msgParts, "  "+prefix+DimStyle.Render(content))
	}

	// Take last N lines.
	if len(msgParts) > msgLines {
		msgParts = msgParts[len(msgParts)-msgLines:]
	}
	lines = append(lines, msgParts...)

	// Pad.
	for len(lines) < height-1 {
		lines = append(lines, "")
	}

	// Input hint at bottom.
	lines = append(lines, "  "+BrandStyle.Render("nick → ")+DimStyle.Render("type here..."))

	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// dashboardPortfolioPane renders the portfolio sidebar.
func (m Model) dashboardPortfolioPane(width, height int) string {
	var lines []string

	header := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("  PORTFOLIO")
	lines = append(lines, header)
	lines = append(lines, DimStyle.Render(strings.Repeat("─", width)))

	// Use cached portfolio data (never call API from render path).
	if m.cachedPortfolio != nil {
		portfolio := m.cachedPortfolio
		// Total value.
		valStr := fmt.Sprintf("$%.0f", portfolio.TotalValue)
		lines = append(lines, "  "+lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(valStr))
		lines = append(lines, "")

		// Positions.
		for _, pos := range portfolio.Assets {
			if pos.Quantity <= 0 {
				continue
			}
			sym := strings.TrimSuffix(pos.Symbol, "USDT")
			line := fmt.Sprintf("  %-5s %.4f", sym, pos.Quantity)
			lines = append(lines, DimStyle.Render(line))
		}

		// Cash.
		lines = append(lines, "")
		cashStr := fmt.Sprintf("  Cash  $%.0f", portfolio.AvailableCash)
		lines = append(lines, DimStyle.Render(cashStr))

		// P&L.
		lines = append(lines, "")
		startingCapital := 100000.0
		pnl := portfolio.TotalValue - startingCapital
		pnlPct := (pnl / startingCapital) * 100
		pnlStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
		pnlSign := "+"
		if pnl < 0 {
			pnlStyle = lipgloss.NewStyle().Foreground(ColorError)
			pnlSign = ""
		}
		lines = append(lines, "  "+pnlStyle.Render(fmt.Sprintf("P&L %s%.1f%%", pnlSign, pnlPct)))
	} else if m.client != nil && m.client.IsConfigured() {
		lines = append(lines, DimStyle.Render("  Loading..."))
	} else {
		lines = append(lines, DimStyle.Render("  Not connected"))
		lines = append(lines, DimStyle.Render("  /config init"))
	}

	// Alert/Trigger counts.
	lines = append(lines, "")
	if len(m.alerts) > 0 {
		lines = append(lines, DimStyle.Render(fmt.Sprintf("  Alerts: %d", len(m.alerts))))
	}
	if len(m.triggers) > 0 {
		lines = append(lines, DimStyle.Render(fmt.Sprintf("  Triggers: %d", len(m.triggers))))
	}

	// Pad to fill height.
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// formatPortfolioTopBar returns a compact portfolio string for the top bar.
func formatPortfolioTopBar(portfolio *api.Portfolio) string {
	if portfolio == nil {
		return ""
	}
	valStr := fmt.Sprintf("$%.1fK", portfolio.TotalValue/1000)
	pnl := portfolio.TotalValue - 100000
	pnlPct := (pnl / 100000) * 100
	sign := "+"
	pnlColor := ColorPrimary
	if pnl < 0 {
		sign = ""
		pnlColor = ColorError
	}
	return lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(valStr) +
		" " +
		lipgloss.NewStyle().Foreground(pnlColor).Render(fmt.Sprintf("%s%.1f%%", sign, pnlPct))
}
