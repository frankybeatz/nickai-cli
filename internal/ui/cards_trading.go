package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/market"
	"github.com/nickai/cli/internal/mcp"
	"github.com/nickai/cli/internal/mock"
)

// --- /agents: mock fallback or live orders ---

func RenderAgentCard(agent mock.Agent, width int) string {
	status := StatusIndicator(agent.Status.String())
	name := BrandStyle.Render(agent.Name)
	header := status + name

	strategy := DimStyle.Render("Strategy: ") + agent.Strategy
	pnlStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
	if strings.HasPrefix(agent.PnL, "-") {
		pnlStyle = lipgloss.NewStyle().Foreground(ColorError)
	}
	pnl := DimStyle.Render("PnL: ") + pnlStyle.Render(agent.PnL)
	uptime := DimStyle.Render("Uptime: ") + agent.Uptime

	body := fmt.Sprintf("%s\n%s    %s    %s", header, strategy, pnl, uptime)
	return AgentCard(width).Render(body)
}

func RenderAgentListMock(width int) string {
	agents := mock.Agents()
	cardWidth := min(width-4, 64)

	title := SectionHeaderWithCount("Your Agents", len(agents))

	var cards []string
	cards = append(cards, title)
	for _, a := range agents {
		cards = append(cards, RenderAgentCard(a, cardWidth))
	}
	cards = append(cards, "")
	cards = append(cards, DimStyle.Render("  (mock data) ")+connectPrompt())
	return strings.Join(cards, "\n")
}

func RenderOrderCard(order api.Order, width int) string {
	statusStr := strings.ToLower(order.Status)
	status := StatusIndicator(orderStatusToIndicator(statusStr))

	side := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(strings.ToUpper(order.Side))
	if strings.ToLower(order.Side) == "sell" {
		side = lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render(strings.ToUpper(order.Side))
	}

	name := BrandStyle.Render(order.Symbol)
	header := fmt.Sprintf("%s%s  %s  %s", status, name, side, DimStyle.Render(order.Type))

	qty := DimStyle.Render("Qty: ") + fmt.Sprintf("%.4f", order.Quantity)

	// Use filledPrice for filled orders, fall back to limit price.
	displayPrice := order.FilledPrice
	if displayPrice == 0 {
		displayPrice = order.Price
	}
	price := DimStyle.Render("Price: ") + formatPrice(displayPrice)
	statusLine := DimStyle.Render("Status: ") + renderOrderStatus(statusStr)

	var timePart string
	if order.FilledAt != "" {
		timePart = DimStyle.Render("  " + order.FilledAt)
	}

	body := fmt.Sprintf("%s\n%s    %s    %s%s", header, qty, price, statusLine, timePart)
	return AgentCard(width).Render(body)
}

func RenderOrderList(client *api.PapernickClient, width int) string {
	orders, err := client.GetOrders()
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch orders: ") + err.Error()
	}

	cardWidth := min(width-4, 64)
	title := SectionHeaderWithCount("Recent Orders", len(orders))

	var cards []string
	cards = append(cards, title)

	if len(orders) == 0 {
		cards = append(cards, EmptyState("No orders yet. Place your first trade!"))
	} else {
		limit := min(len(orders), 10)
		for _, o := range orders[:limit] {
			cards = append(cards, RenderOrderCard(o, cardWidth))
		}
		if len(orders) > 10 {
			cards = append(cards, DimStyle.Render(fmt.Sprintf("  ... and %d more", len(orders)-10)))
		}
	}
	return strings.Join(cards, "\n") + NextSteps("/pnl", "/history")
}

// --- /status: mock fallback or live portfolio ---

func RenderStatusMock(mcpMgr *mcp.ClientManager) string {
	header := SectionHeader("Platform Status")
	lines := []string{
		header,
		"  " + StatusIndicator("running") + "API Gateway        " + BrandStyle.Render("operational"),
		"  " + StatusIndicator("running") + "Agent Runtime      " + BrandStyle.Render("operational"),
		"  " + StatusIndicator("running") + "Market Data Feed   " + BrandStyle.Render("operational"),
		"  " + StatusIndicator("stopped") + "Backtesting Engine " + WarningStyle.Render("maintenance"),
		"",
		"  " + DimStyle.Render("Agents running: ") + "2" +
			"    " + DimStyle.Render("Uptime: ") + "99.7%",
	}
	lines = append(lines, renderMCPStatus(mcpMgr)...)
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("  (mock data) ")+connectPrompt())
	return strings.Join(lines, "\n")
}

func RenderStatusLive(client *api.PapernickClient, mcpMgr *mcp.ClientManager, width int) string {
	portfolio, err := client.GetPortfolio()
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch portfolio: ") + err.Error()
	}

	cardWidth := min(width-4, 64)
	header := SectionHeader("Portfolio Status")
	var lines []string
	lines = append(lines, header)

	// Cash section (from portfolio response).
	lines = append(lines, "  "+DimStyle.Render("Cash Available:  ")+
		BrandStyle.Render(formatMoney(portfolio.AvailableCash)))
	if portfolio.ReservedCash > 0 {
		lines = append(lines, "  "+DimStyle.Render("Cash Reserved:   ")+
			WarningStyle.Render(formatMoney(portfolio.ReservedCash)))
	}
	lines = append(lines, "  "+DimStyle.Render("Cash Total:      ")+
		formatMoney(portfolio.Cash))

	lines = append(lines, "")

	// Total value.
	lines = append(lines, "  "+DimStyle.Render("Total Value:     ")+
		BrandStyle.Render(formatMoney(portfolio.TotalValue)))

	// Positions section.
	if len(portfolio.Assets) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+SecondaryStyle.Render("Positions"))

		for _, pos := range portfolio.Assets {
			lines = append(lines, renderPositionCard(pos, cardWidth))
		}
	} else {
		lines = append(lines, "")
		lines = append(lines, "  "+DimStyle.Render("No open positions."))
	}

	lines = append(lines, renderMCPStatus(mcpMgr)...)

	return strings.Join(lines, "\n")
}

// renderMCPStatus returns lines showing MCP connection status.
func renderMCPStatus(mcpMgr *mcp.ClientManager) []string {
	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  "+SecondaryStyle.Render("MCP Servers"))

	if mcpMgr == nil || mcpMgr.ConnectionCount() == 0 {
		lines = append(lines, "  "+DimStyle.Render("No servers connected.")+
			"  "+CommandStyle.Render("/mcp search")+DimStyle.Render(" to browse."))
		return lines
	}

	for _, conn := range mcpMgr.Connections() {
		lines = append(lines, "  "+StatusIndicator("running")+
			BrandStyle.Render(conn.Name)+
			DimStyle.Render(fmt.Sprintf("  %d tools", len(conn.Tools))))
	}
	return lines
}

func renderPositionCard(pos api.Position, width int) string {
	name := BrandStyle.Render(pos.Symbol)
	qty := DimStyle.Render("Qty: ") + fmt.Sprintf("%.4f", pos.Quantity)
	val := DimStyle.Render("Value: ") + BrandStyle.Render(formatMoney(pos.Value))

	var availPart string
	if pos.ReservedQuantity > 0 {
		availPart = "\n" + DimStyle.Render("Available: ") +
			fmt.Sprintf("%.4f", pos.AvailableQuantity) +
			"    " + DimStyle.Render("Reserved: ") +
			WarningStyle.Render(fmt.Sprintf("%.4f", pos.ReservedQuantity))
	}

	body := fmt.Sprintf("%s\n%s    %s%s", name, qty, val, availPart)
	return AgentCard(width).Render(body)
}

// --- /price: live quotes ---

func RenderPrices(client *api.PapernickClient, symbols []string, width int) string {
	prices, err := client.GetPrices(symbols)
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch prices: ") + err.Error()
	}

	if len(prices) == 0 {
		return DimStyle.Render("  No price data returned for: ") + strings.Join(symbols, ", ")
	}

	cardWidth := min(width-4, 64)
	title := SectionHeader("Live Quotes")
	var cards []string
	cards = append(cards, title)

	for _, p := range prices {
		// Try to fetch a short price history for the sparkline.
		var history []float64
		if candles, err := market.FetchKlines(p.Symbol, "1h", 24); err == nil && len(candles) > 0 {
			history = market.ClosePrices(candles)
		}
		cards = append(cards, renderPriceCard(p, history, cardWidth))
	}
	sym := strings.TrimSuffix(symbols[0], "USDT")
	return strings.Join(cards, "\n") + NextSteps("/chart "+sym, "/buy "+sym)
}

func renderPriceCard(p api.Price, history []float64, width int) string {
	name := BrandStyle.Render(p.Symbol)
	price := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).
		Render(formatPrice(p.Price))

	sparkWidth := 24
	sparkStr := ""
	if len(history) >= 2 {
		up := lipgloss.NewStyle().Foreground(ColorPrimary)
		down := lipgloss.NewStyle().Foreground(ColorError)
		sparkStr = "  " + SparklineWithColor(history, sparkWidth, up, down)
	}

	body := fmt.Sprintf("%s  %s%s", name, price, sparkStr)
	return Card(width).Render(body)
}

// --- /watch: live price dashboard ---

func RenderWatch(client *api.PapernickClient, symbols []string, width int) string {
	if !client.IsConfigured() {
		return connectPrompt()
	}

	prices, err := client.GetPrices(symbols)
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch prices: ") + err.Error()
	}

	if len(prices) == 0 {
		return DimStyle.Render("  No price data returned for: ") + strings.Join(symbols, ", ")
	}

	cardWidth := min(width-4, 60)

	liveIndicator := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render("◉ LIVE")

	var rows []string
	rows = append(rows, liveIndicator)
	rows = append(rows, "")

	for _, p := range prices {
		sym := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).
			Render(padRight(p.Symbol, 12))
		price := BrandStyle.Render(formatPrice(p.Price))
		rows = append(rows, sym+price)
	}

	content := strings.Join(rows, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Width(cardWidth).
		Render(content)

	return box
}

// --- /buy, /sell: order confirmation ---

// RenderOrderConfirmation renders a styled card for a placed order.
func RenderOrderConfirmation(order *api.Order, width int) string {
	cardWidth := min(width-4, 64)

	sideColor := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	if strings.ToLower(order.Side) == "sell" {
		sideColor = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
	}

	checkmark := lipgloss.NewStyle().Foreground(ColorPrimary).Render("✓ ")
	title := checkmark + sideColor.Render(strings.ToUpper(order.Side)) +
		" " + BrandStyle.Render(order.Symbol)

	qty := DimStyle.Render("Quantity:  ") + fmt.Sprintf("%.4f", order.Quantity)
	orderType := DimStyle.Render("Type:      ") + order.Type

	displayPrice := order.FilledPrice
	if displayPrice == 0 {
		displayPrice = order.Price
	}
	var priceLine string
	if displayPrice > 0 {
		priceLine = DimStyle.Render("Price:     ") + formatPrice(displayPrice)
	}

	statusLine := DimStyle.Render("Status:    ") + renderOrderStatus(strings.ToLower(order.Status))
	idLine := DimStyle.Render("Order ID:  ") + DimStyle.Render(order.ID)

	var bodyParts []string
	bodyParts = append(bodyParts, title)
	bodyParts = append(bodyParts, qty)
	bodyParts = append(bodyParts, orderType)
	if priceLine != "" {
		bodyParts = append(bodyParts, priceLine)
	}
	bodyParts = append(bodyParts, statusLine)
	bodyParts = append(bodyParts, idLine)

	body := strings.Join(bodyParts, "\n")

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Width(cardWidth).
		Render(body)

	return BotMsgStyle.Render("nick: ") + "Order placed.\n" + card
}

// RenderOrderError renders a styled error for a failed order.
func RenderOrderError(side, symbol string, err error) string {
	return ErrorStyle.Render("  Order failed: ") +
		strings.ToUpper(side) + " " + symbol + "\n" +
		"  " + DimStyle.Render(err.Error())
}

// --- Trade confirmation card ---

// RenderTradeConfirmCard renders a styled confirmation card for a pending trade.
func RenderTradeConfirmCard(req *api.PlaceOrderRequest, width int) string {
	cardWidth := min(width-4, 64)

	sideColor := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	if req.Side == "sell" {
		sideColor = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, sideColor.Render(strings.ToUpper(req.Side))+" "+BrandStyle.Render(req.Symbol))
	lines = append(lines, DimStyle.Render("Quantity:  ")+fmt.Sprintf("%.4f", req.Quantity))
	lines = append(lines, DimStyle.Render("Type:      ")+req.Type)
	if req.Price > 0 {
		lines = append(lines, DimStyle.Render("Price:     ")+formatPrice(req.Price))
	}
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).
		Render("Press y to confirm, n to cancel"))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorWarning).
		Padding(0, 1).
		Width(cardWidth).
		Render(content)

	return WarningStyle.Render("  CONFIRM TRADE") + "\n" + box
}

// --- /templates: unchanged, always mock ---

func RenderTemplateCard(tmpl mock.Template, width int) string {
	name := SecondaryStyle.Render(tmpl.Name)
	author := DimStyle.Render("by " + tmpl.Author)
	stars := WarningStyle.Render(fmt.Sprintf("★ %d", tmpl.Stars))
	header := fmt.Sprintf("%s  %s  %s", name, author, stars)

	desc := tmpl.Description

	var tagParts []string
	for _, t := range tmpl.Tags {
		tagParts = append(tagParts, DimStyle.Render("["+t+"]"))
	}
	tags := strings.Join(tagParts, " ")

	body := fmt.Sprintf("%s\n%s\n%s", header, desc, tags)
	return Card(width).Render(body)
}

func RenderTemplateList(width int) string {
	templates := mock.Templates()
	cardWidth := min(width-4, 64)

	title := SectionHeaderWithCount("Template Marketplace", len(templates))

	var cards []string
	cards = append(cards, title)
	for _, t := range templates {
		cards = append(cards, RenderTemplateCard(t, cardWidth))
	}
	return strings.Join(cards, "\n")
}
