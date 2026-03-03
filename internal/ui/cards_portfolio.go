package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/mock"
)

// --- /snapshot: combined dashboard ---

func RenderSnapshot(client *api.PapernickClient, width int) string {
	cardWidth := min(width-4, 72)
	halfWidth := (cardWidth - 6) / 2

	// Fetch portfolio data.
	var portfolioLines []string
	portfolioLines = append(portfolioLines, "")
	portfolioLines = append(portfolioLines, SecondaryStyle.Render("PORTFOLIO"))
	portfolio, err := client.GetPortfolio()
	if err != nil {
		portfolioLines = append(portfolioLines, ErrorStyle.Render("Error: "+err.Error()))
	} else {
		portfolioLines = append(portfolioLines, DimStyle.Render("Cash:  ")+BrandStyle.Render(formatMoney(portfolio.AvailableCash)))
		for _, pos := range portfolio.Assets {
			portfolioLines = append(portfolioLines, DimStyle.Render(padRight(pos.Symbol+":", 7))+fmt.Sprintf("%.4f", pos.Quantity))
		}
		portfolioLines = append(portfolioLines, DimStyle.Render("Total: ")+BrandStyle.Render(formatMoney(portfolio.TotalValue)))
	}

	// Fetch market data.
	var marketLines []string
	marketLines = append(marketLines, "")
	marketLines = append(marketLines, SecondaryStyle.Render("MARKET"))
	topSymbols := []string{"BTC", "ETH", "SOL", "DOGE"}
	prices, err := client.GetPrices(topSymbols)
	if err != nil {
		marketLines = append(marketLines, ErrorStyle.Render("Error: "+err.Error()))
	} else {
		for _, p := range prices {
			sym := padRight(p.Symbol, 12)
			marketLines = append(marketLines, lipgloss.NewStyle().Foreground(ColorWhite).Render(sym)+formatPrice(p.Price))
		}
	}

	// Pad columns to equal height.
	for len(portfolioLines) < len(marketLines) {
		portfolioLines = append(portfolioLines, "")
	}
	for len(marketLines) < len(portfolioLines) {
		marketLines = append(marketLines, "")
	}

	leftCol := lipgloss.NewStyle().Width(halfWidth).Render(strings.Join(portfolioLines, "\n"))
	rightCol := lipgloss.NewStyle().Width(halfWidth).Render(strings.Join(marketLines, "\n"))
	topSection := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "  ", rightCol)

	// Agents section (mock).
	var agentLines []string
	agentLines = append(agentLines, SecondaryStyle.Render("AGENTS"))
	for _, a := range mock.Agents() {
		status := StatusIndicator(a.Status.String())
		pnlStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
		if strings.HasPrefix(a.PnL, "-") {
			pnlStyle = lipgloss.NewStyle().Foreground(ColorError)
		}
		agentLines = append(agentLines, status+padRight(a.Name, 18)+DimStyle.Render(padRight(a.Status.String(), 10))+pnlStyle.Render(a.PnL))
	}
	agentSection := strings.Join(agentLines, "\n")

	// Recent trades section.
	var tradeLines []string
	tradeLines = append(tradeLines, SecondaryStyle.Render("RECENT TRADES"))
	orders, err := client.GetOrders()
	if err != nil {
		tradeLines = append(tradeLines, ErrorStyle.Render("Error: "+err.Error()))
	} else {
		limit := min(len(orders), 5)
		if limit == 0 {
			tradeLines = append(tradeLines, DimStyle.Render("No trades yet."))
		}
		for _, o := range orders[:limit] {
			sideStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
			if strings.ToLower(o.Side) == "sell" {
				sideStyle = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
			}
			side := sideStyle.Render(padRight(strings.ToUpper(o.Side), 5))
			sym := padRight(o.Symbol, 10)
			price := formatMoney(o.Quantity)
			status := renderOrderStatus(strings.ToLower(o.Status))
			tradeLines = append(tradeLines, side+sym+padRight(price, 12)+status)
		}
	}
	tradeSection := strings.Join(tradeLines, "\n")

	// Assemble all sections.
	content := strings.Join([]string{
		"",
		topSection,
		"",
		agentSection,
		"",
		tradeSection,
		"",
	}, "\n")

	title := " NickAI Snapshot "
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Width(cardWidth).
		Render(content)

	return SecondaryStyle.Render("  "+title) + "\n" + box
}

// --- /market: market overview ---

func RenderMarket(client *api.PapernickClient, width int) string {
	cardWidth := min(width-4, 60)
	symbols := []string{"BTC", "ETH", "SOL", "DOGE", "ADA", "AVAX", "LINK", "DOT", "MATIC", "XRP"}

	prices, err := client.GetPrices(symbols)
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch prices: ") + err.Error()
	}

	var rows []string
	rows = append(rows, "")

	// Header row.
	header := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).
		Render(padRight("Asset", 12) + padRight("Price", 16))
	rows = append(rows, header)
	rows = append(rows, Divider(28))

	for _, p := range prices {
		sym := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).
			Render(padRight(p.Symbol, 12))
		price := BrandStyle.Render(padRight(formatPrice(p.Price), 16))
		rows = append(rows, sym+price)
	}

	rows = append(rows, "")
	rows = append(rows, DimStyle.Render("Last updated: "+time.Now().UTC().Format("3:04 PM UTC")))
	rows = append(rows, "")

	content := strings.Join(rows, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(0, 2).
		Width(cardWidth).
		Render(content)

	return SecondaryStyle.Render("  Market Overview") + "\n" + box
}

// --- /pnl: profit and loss ---

func RenderPnl(client *api.PapernickClient, width int) string {
	cardWidth := min(width-4, 60)
	startingBalance := 100000.0

	portfolio, err := client.GetPortfolio()
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch portfolio: ") + err.Error()
	}

	currentBalance := portfolio.TotalValue
	if currentBalance == 0 {
		currentBalance = portfolio.Cash
	}
	totalPnL := currentBalance - startingBalance
	pnlPct := (totalPnL / startingBalance) * 100

	// Fetch orders for trade stats.
	orders, _ := client.GetOrders()
	totalTrades := len(orders)

	// Analyze trades.
	won := 0
	lost := 0
	bestTrade := 0.0
	worstTrade := 0.0
	bestSymbol := ""
	worstSymbol := ""

	// Simple heuristic: filled sell orders with positive filledPrice vs buy average.
	// Since we can't perfectly reconstruct P&L per trade, we use order-level data.
	for _, o := range orders {
		if strings.ToLower(o.Status) != "filled" {
			continue
		}
		if strings.ToLower(o.Side) == "sell" && o.FilledPrice > 0 {
			// Rough estimate: treat each sell as a win if price > 0.
			pnl := o.FilledPrice * o.Quantity
			if pnl > bestTrade {
				bestTrade = pnl
				bestSymbol = o.Symbol
			}
			won++
		} else if strings.ToLower(o.Side) == "buy" {
			lost++ // count buys as separate for trade count
		}
	}
	if worstSymbol == "" && len(orders) > 0 {
		worstTrade = 0
		worstSymbol = "N/A"
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("  Starting Balance:  ")+formatMoney(startingBalance))
	lines = append(lines, DimStyle.Render("  Current Balance:   ")+BrandStyle.Render(formatMoney(currentBalance)))

	pnlStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	pnlPrefix := "+"
	if totalPnL < 0 {
		pnlStyle = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
		pnlPrefix = ""
	}
	lines = append(lines, DimStyle.Render("  Total P&L:         ")+
		pnlStyle.Render(fmt.Sprintf("%s$%.2f (%s%.2f%%)", pnlPrefix, totalPnL, pnlPrefix, pnlPct)))

	// Progress bar.
	lines = append(lines, "")
	barWidth := 30
	filledWidth := int(math.Abs(pnlPct) / 100.0 * float64(barWidth))
	if filledWidth > barWidth {
		filledWidth = barWidth
	}
	if filledWidth < 0 {
		filledWidth = 0
	}
	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)
	barStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
	if totalPnL < 0 {
		barStyle = lipgloss.NewStyle().Foreground(ColorError)
	}
	lines = append(lines, barStyle.Render(bar)+"  "+pnlStyle.Render(fmt.Sprintf("%s%.2f%%", pnlPrefix, pnlPct)))

	// Trade stats.
	lines = append(lines, "")
	winRate := 0.0
	if won+lost > 0 {
		winRate = float64(won) / float64(won+lost) * 100
	}
	lines = append(lines, DimStyle.Render(fmt.Sprintf("Trades: %d  |  Won: %d  |  Lost: %d", totalTrades, won, lost)))
	lines = append(lines, DimStyle.Render(fmt.Sprintf("Win Rate: %.1f%%", winRate)))
	if bestSymbol != "" && bestTrade > 0 {
		lines = append(lines, DimStyle.Render("Best Trade:  ")+
			lipgloss.NewStyle().Foreground(ColorPrimary).Render(fmt.Sprintf("+$%.0f (%s)", bestTrade, bestSymbol)))
	}
	if worstSymbol != "" && worstTrade != 0 {
		lines = append(lines, DimStyle.Render("Worst Trade: ")+
			lipgloss.NewStyle().Foreground(ColorError).Render(fmt.Sprintf("-$%.0f (%s)", math.Abs(worstTrade), worstSymbol)))
	}
	lines = append(lines, "")

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Width(cardWidth).
		Render(content)

	return SecondaryStyle.Render("  Profit & Loss") + "\n" + box +
		NextSteps("/analytics", "/risk")
}

// --- /history: trade journal ---

func RenderHistory(client *api.PapernickClient, width int) string {
	cardWidth := min(width-4, 68)

	orders, err := client.GetOrders()
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch orders: ") + err.Error()
	}

	var rows []string
	rows = append(rows, "")

	// Header.
	header := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).
		Render(padRight("#", 5) + padRight("Time", 14) + padRight("Side", 6) + padRight("Asset", 12) + padRight("Amount", 12) + "Status")
	rows = append(rows, header)
	rows = append(rows, Divider(55))

	if len(orders) == 0 {
		rows = append(rows, DimStyle.Render("  No trades yet. Place your first trade!"))
	} else {
		todayCount := 0
		today := time.Now().Format("2006-01-02")
		for i, o := range orders {
			num := DimStyle.Render(padRight(fmt.Sprintf("%d", i+1), 5))

			timeStr := o.FilledAt
			if timeStr == "" {
				timeStr = "pending"
			} else {
				// Try to extract just time portion.
				if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
					timeStr = t.Format("03:04 PM")
					if t.Format("2006-01-02") == today {
						todayCount++
					}
				}
			}
			timeCol := DimStyle.Render(padRight(timeStr, 14))

			sideStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
			if strings.ToLower(o.Side) == "sell" {
				sideStyle = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
			}
			side := sideStyle.Render(padRight(strings.ToUpper(o.Side), 6))
			asset := lipgloss.NewStyle().Foreground(ColorWhite).Render(padRight(o.Symbol, 12))
			amount := DimStyle.Render(padRight(fmt.Sprintf("$%.2f", o.Quantity*o.FilledPrice), 12))
			if o.FilledPrice == 0 {
				amount = DimStyle.Render(padRight(fmt.Sprintf("%.4f", o.Quantity), 12))
			}
			status := renderOrderStatus(strings.ToLower(o.Status))

			rows = append(rows, num+timeCol+side+asset+amount+status)
		}

		rows = append(rows, "")
		rows = append(rows, DimStyle.Render(fmt.Sprintf("Total trades: %d  |  Today: %d", len(orders), todayCount)))
	}
	rows = append(rows, "")

	content := strings.Join(rows, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(0, 2).
		Width(cardWidth).
		Render(content)

	return SecondaryStyle.Render("  Trade Journal") + "\n" + box
}
