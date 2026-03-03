package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/market"
)

// defaultWatchSymbols are shown when the user runs /watch with no arguments.
var defaultWatchSymbols = []string{"BTC", "ETH", "SOL"}

// handleWatchCommand enters watch mode for the given symbols.
func (m Model) handleWatchCommand(args []string) (tea.Model, tea.Cmd) {
	if !m.client.IsConfigured() {
		m.addBotMessage(connectPrompt())
		m.updateViewport()
		return m, nil
	}

	// Toggle off if already watching.
	if m.watchMode {
		m.watchMode = false
		m.watchPrices = nil
		m.watchHistory = nil
		m.addBotMessage(BotMsgStyle.Render("nick: ") + "Exited watch mode.")
		m.updateViewport()
		return m, nil
	}

	symbols := defaultWatchSymbols
	if len(args) > 0 {
		symbols = make([]string, len(args))
		for i, s := range args {
			symbols[i] = strings.ToUpper(s)
		}
	}

	m.watchMode = true
	m.dashboardMode = false // only one mode at a time
	m.watchSymbols = symbols
	m.watchPrices = nil
	m.watchHistory = nil

	m.addBotMessage(BotMsgStyle.Render("nick: ") + "Watch mode for " +
		BrandStyle.Render(strings.Join(symbols, ", ")) +
		". Press " + CommandStyle.Render("Esc") + " to exit.")
	m.updateViewport()

	// Kick off the first data fetch immediately, then schedule ticks.
	client := m.client
	symCopy := make([]string, len(symbols))
	copy(symCopy, symbols)
	return m, tea.Batch(
		func() tea.Msg {
			prices, err := client.GetPrices(symCopy)
			if err != nil {
				return watchDataMsg{}
			}
			history := make(map[string][]float64)
			for _, sym := range symCopy {
				if candles, err := market.FetchKlines(sym, "1h", 24); err == nil && len(candles) > 0 {
					history[sym] = market.ClosePrices(candles)
				}
			}
			return watchDataMsg{prices: prices, history: history}
		},
		tea.Tick(5*time.Second, func(t time.Time) tea.Msg { return watchTickMsg{} }),
	)
}

// renderWatchView renders the full-screen watch dashboard.
func (m Model) renderWatchView() string {
	// Height budget: subtract top bar (1) + input bar (1) + borders (2).
	innerHeight := m.height - 4
	if innerHeight < 6 {
		innerHeight = 6
	}

	if len(m.watchPrices) == 0 {
		loading := lipgloss.NewStyle().
			Width(m.width).
			Height(innerHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(ColorDim).
			Render("Loading watch data...")
		return loading
	}

	// Card layout: arrange symbol cards in a grid.
	cardWidth := m.width - 4
	if cardWidth > 70 {
		cardWidth = 70
	}

	var cards []string

	// Live indicator header.
	liveIndicator := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render("  ◉ LIVE WATCH")
	symList := DimStyle.Render("  " + strings.Join(m.watchSymbols, " | "))
	cards = append(cards, liveIndicator+"  "+symList)
	cards = append(cards, "")

	for _, p := range m.watchPrices {
		card := m.renderWatchCard(p.Symbol, p.Price, cardWidth)
		cards = append(cards, card)
	}

	// Portfolio summary if available.
	if m.cachedPortfolio != nil {
		cards = append(cards, "")
		cards = append(cards, m.renderWatchPortfolio(cardWidth))
	}

	// Pad with empty lines to fill available height.
	content := strings.Join(cards, "\n")
	contentLines := strings.Count(content, "\n") + 1
	if contentLines < innerHeight-1 {
		content += strings.Repeat("\n", innerHeight-1-contentLines)
	}

	// Hint at bottom.
	hint := DimStyle.Render("  Press Esc to exit watch mode  |  Refreshes every 5s")
	content += "\n" + hint

	return content
}

// renderWatchCard renders a single symbol card for the watch view.
func (m Model) renderWatchCard(symbol string, price float64, width int) string {
	upStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
	downStyle := lipgloss.NewStyle().Foreground(ColorError)

	// Symbol and price header.
	sym := BrandStyle.Render(symbol)
	priceStr := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(formatPrice(price))

	// 24h change from history data.
	var changeStr string
	var sparkStr string
	var highLowStr string

	history := m.watchHistory[symbol]
	if len(history) >= 2 {
		first := history[0]
		last := history[len(history)-1]
		pctChange := (last - first) / first * 100

		if pctChange >= 0 {
			changeStr = upStyle.Render(fmt.Sprintf("+%.2f%%", pctChange))
		} else {
			changeStr = downStyle.Render(fmt.Sprintf("%.2f%%", pctChange))
		}

		// Sparkline for 24h.
		sparkWidth := width - 8
		if sparkWidth > 50 {
			sparkWidth = 50
		}
		if sparkWidth < 10 {
			sparkWidth = 10
		}
		sparkStr = SparklineWithColor(history, sparkWidth, upStyle, downStyle)

		// High/Low from history.
		cleaned := cleanData(history)
		if len(cleaned) > 0 {
			high, low := bounds(cleaned)
			highLowStr = DimStyle.Render("H ") + formatPrice(high) +
				DimStyle.Render("  L ") + formatPrice(low)
		}
	}

	// Build card content.
	var lines []string
	headerLine := sym + "  " + priceStr
	if changeStr != "" {
		headerLine += "  " + changeStr
	}
	lines = append(lines, headerLine)

	if sparkStr != "" {
		lines = append(lines, sparkStr)
	}
	if highLowStr != "" {
		lines = append(lines, highLowStr)
	}

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorDim).
		Padding(0, 2).
		Width(width).
		Render(content)

	return box
}

// renderWatchPortfolio renders a compact portfolio summary for the watch view.
func (m Model) renderWatchPortfolio(width int) string {
	portfolio := m.cachedPortfolio
	if portfolio == nil {
		return ""
	}

	valStr := fmt.Sprintf("$%.2f", portfolio.TotalValue)
	startingCapital := 100000.0
	pnl := portfolio.TotalValue - startingCapital
	pnlPct := (pnl / startingCapital) * 100

	pnlStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
	pnlSign := "+"
	if pnl < 0 {
		pnlStyle = lipgloss.NewStyle().Foreground(ColorError)
		pnlSign = ""
	}

	cashStr := fmt.Sprintf("$%.2f", portfolio.AvailableCash)

	content := DimStyle.Render("Portfolio  ") +
		lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(valStr) +
		"  " + pnlStyle.Render(fmt.Sprintf("%s%.1f%%", pnlSign, pnlPct)) +
		DimStyle.Render("  |  Cash ") + cashStr

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorDim).
		Padding(0, 2).
		Width(width).
		Render(content)

	return box
}
