package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/config"
	"github.com/nickai/cli/internal/mock"
)

// --- Connect prompt ---

func connectPrompt() string {
	return DimStyle.Render("  No API key configured. ") +
		"Connect your account with " +
		CommandStyle.Render("/config set api_key <key>")
}

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

	title := SecondaryStyle.Render("  Your Agents") +
		DimStyle.Render(fmt.Sprintf("  (%d total)", len(agents)))

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
	title := SecondaryStyle.Render("  Recent Orders") +
		DimStyle.Render(fmt.Sprintf("  (%d total)", len(orders)))

	var cards []string
	cards = append(cards, title)

	if len(orders) == 0 {
		cards = append(cards, DimStyle.Render("  No orders yet. Place your first trade!"))
	} else {
		limit := min(len(orders), 10)
		for _, o := range orders[:limit] {
			cards = append(cards, RenderOrderCard(o, cardWidth))
		}
		if len(orders) > 10 {
			cards = append(cards, DimStyle.Render(fmt.Sprintf("  ... and %d more", len(orders)-10)))
		}
	}
	return strings.Join(cards, "\n")
}

// --- /status: mock fallback or live portfolio ---

func RenderStatusMock() string {
	header := SecondaryStyle.Render("  Platform Status\n")
	lines := []string{
		header,
		"  " + StatusIndicator("running") + "API Gateway        " + BrandStyle.Render("operational"),
		"  " + StatusIndicator("running") + "Agent Runtime      " + BrandStyle.Render("operational"),
		"  " + StatusIndicator("running") + "Market Data Feed   " + BrandStyle.Render("operational"),
		"  " + StatusIndicator("stopped") + "Backtesting Engine " + WarningStyle.Render("maintenance"),
		"",
		"  " + DimStyle.Render("Agents running: ") + "2" +
			"    " + DimStyle.Render("Uptime: ") + "99.7%",
		"",
		DimStyle.Render("  (mock data) ") + connectPrompt(),
	}
	return strings.Join(lines, "\n")
}

func RenderStatusLive(client *api.PapernickClient, width int) string {
	portfolio, err := client.GetPortfolio()
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch portfolio: ") + err.Error()
	}

	cardWidth := min(width-4, 64)
	header := SecondaryStyle.Render("  Portfolio Status\n")
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

	return strings.Join(lines, "\n")
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
	title := SecondaryStyle.Render("  Live Quotes\n")
	var cards []string
	cards = append(cards, title)

	for _, p := range prices {
		cards = append(cards, renderPriceCard(p, cardWidth))
	}
	return strings.Join(cards, "\n")
}

func renderPriceCard(p api.Price, width int) string {
	name := BrandStyle.Render(p.Symbol)
	price := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).
		Render(formatPrice(p.Price))

	body := fmt.Sprintf("%s  %s", name, price)
	return Card(width).Render(body)
}

// --- /config ---

func RenderConfigShow(cfg *config.Config) string {
	header := SecondaryStyle.Render("  Configuration\n")

	anthropicStatus := cfg.MaskedAnthropicKey()
	if cfg.AnthropicKey == "" {
		if envKey := cfg.AnthropicKeyOrEnv(); envKey != "" {
			anthropicStatus = maskKeyShort(envKey) + DimStyle.Render(" (from env)")
		}
	}

	lines := []string{
		header,
		"  " + DimStyle.Render("API Key:        ") + cfg.MaskedKey(),
		"  " + DimStyle.Render("Anthropic Key:  ") + anthropicStatus,
		"  " + DimStyle.Render("Base URL:       ") + cfg.BaseURL,
	}
	return strings.Join(lines, "\n")
}

func maskKeyShort(k string) string {
	if k == "" {
		return "(not set)"
	}
	if len(k) <= 8 {
		return k[:2] + "***"
	}
	return k[:4] + "..." + k[len(k)-4:]
}

func RenderConfigSet(key, value string) string {
	return BotMsgStyle.Render("nick: ") + "Set " +
		CommandStyle.Render(key) + " successfully."
}

func RenderConfigTest(client *api.PapernickClient) string {
	user, err := client.TestConnection()
	if err != nil {
		return ErrorStyle.Render("  Connection failed: ") + err.Error()
	}
	lines := []string{
		BotMsgStyle.Render("nick: ") + "Connection successful!",
		"",
		"  " + StatusIndicator("running") + BrandStyle.Render("Connected"),
		"  " + DimStyle.Render("User:  ") + user.Name,
		"  " + DimStyle.Render("Email: ") + user.Email,
		"  " + DimStyle.Render("ID:    ") + user.ID,
	}
	return strings.Join(lines, "\n")
}

func RenderConfigHelp() string {
	header := SecondaryStyle.Render("  /config usage\n")
	lines := []string{
		header,
		"  " + CommandStyle.Render("/config show") + DimStyle.Render("                    — display current config"),
		"  " + CommandStyle.Render("/config set api_key <key>") + DimStyle.Render("       — set PaperNick API key"),
		"  " + CommandStyle.Render("/config set anthropic_key <key>") + DimStyle.Render(" — set Anthropic API key"),
		"  " + CommandStyle.Render("/config set url <url>") + DimStyle.Render("           — set base URL"),
		"  " + CommandStyle.Render("/config test") + DimStyle.Render("                    — test API connection"),
	}
	return strings.Join(lines, "\n")
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

	title := SecondaryStyle.Render("  Template Marketplace") +
		DimStyle.Render(fmt.Sprintf("  (%d available)", len(templates)))

	var cards []string
	cards = append(cards, title)
	for _, t := range templates {
		cards = append(cards, RenderTemplateCard(t, cardWidth))
	}
	return strings.Join(cards, "\n")
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

// NormalizeSymbol delegates to api.NormalizeSymbol.
func NormalizeSymbol(s string) string {
	return api.NormalizeSymbol(s)
}

// --- /help ---

func RenderHelp() string {
	header := SecondaryStyle.Render("  NickAI Commands\n")

	cmds := []struct{ cmd, desc string }{
		{"/help", "Show this help message"},
		{"/status", "Portfolio, positions & cash"},
		{"/orders", "Recent orders & trades"},
		{"/agents", "List your trading agents"},
		{"/templates", "Browse marketplace templates"},
		{"/buy BTC 0.1", "Market buy"},
		{"/sell ETH 1.0", "Market sell"},
		{"/buy BTC 0.1 limit 65000", "Limit buy at price"},
		{"/price BTC ETH", "Live price quotes"},
		{"/config", "Manage API key & connection"},
		{"/clear", "Clear chat history"},
		{"/quit", "Exit NickAI"},
	}

	var lines []string
	lines = append(lines, header)
	for _, c := range cmds {
		line := CommandStyle.Render(padRight(c.cmd, 28)) + DimStyle.Render(c.desc)
		lines = append(lines, "  "+line)
	}

	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("  Or just type naturally — ask me to create, deploy, or manage agents."))
	return strings.Join(lines, "\n")
}

// --- Formatting helpers ---

func formatMoney(v float64) string {
	if v >= 1_000_000 || v <= -1_000_000 {
		return fmt.Sprintf("$%.2fM", v/1_000_000)
	}
	return fmt.Sprintf("$%.2f", v)
}

func formatPrice(v float64) string {
	if v >= 1 {
		return fmt.Sprintf("$%.2f", v)
	}
	return fmt.Sprintf("$%.6f", v)
}

func orderStatusToIndicator(s string) string {
	switch s {
	case "filled", "completed":
		return "running"
	case "cancelled", "rejected", "expired":
		return "stopped"
	case "failed":
		return "error"
	default:
		return "running" // pending, open, partial
	}
}

func renderOrderStatus(s string) string {
	switch s {
	case "filled", "completed":
		return BrandStyle.Render(s)
	case "cancelled", "rejected", "expired":
		return DimStyle.Render(s)
	case "failed":
		return ErrorStyle.Render(s)
	default:
		return WarningStyle.Render(s) // pending, open, partial
	}
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
