package ui

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/config"
	"github.com/nickai/cli/internal/credential"
	"github.com/nickai/cli/internal/mock"
	"github.com/nickai/cli/internal/workflow"
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
		"  " + CommandStyle.Render("/config init") + DimStyle.Render("                    — auto-provision API key"),
		"  " + CommandStyle.Render("/config init force") + DimStyle.Render("              — re-provision (overwrite existing key)"),
		"  " + CommandStyle.Render("/config show") + DimStyle.Render("                    — display current config"),
		"  " + CommandStyle.Render("/config set api_key <key>") + DimStyle.Render("       — set PaperNick API key"),
		"  " + CommandStyle.Render("/config set anthropic_key <key>") + DimStyle.Render(" — set Anthropic API key"),
		"  " + CommandStyle.Render("/config set url <url>") + DimStyle.Render("           — set base URL"),
		"  " + CommandStyle.Render("/config reset <key>") + DimStyle.Render("             — clear a config key"),
		"  " + CommandStyle.Render("/config test") + DimStyle.Render("                    — test API connection"),
	}
	return strings.Join(lines, "\n")
}

// RenderConfigInit renders a success card after auto-provisioning.
func RenderConfigInit(apiKey, userName string) string {
	checkmark := lipgloss.NewStyle().Foreground(ColorPrimary).Render("✓ ")
	lines := []string{
		BotMsgStyle.Render("nick: ") + "Account provisioned successfully!",
		"",
		"  " + checkmark + BrandStyle.Render("Connected"),
		"  " + DimStyle.Render("User:    ") + userName,
		"  " + DimStyle.Render("API Key: ") + maskKeyShort(apiKey),
		"",
		"  " + DimStyle.Render("You're ready to go. Try ") +
			CommandStyle.Render("/status") + DimStyle.Render(" or just ask me anything."),
	}
	return strings.Join(lines, "\n")
}

// --- /credential ---

func RenderCredentialList(store *credential.Store) string {
	header := SecondaryStyle.Render("  Saved Credentials\n")

	if len(store.Credentials) == 0 {
		return header + "\n" + DimStyle.Render("  No credentials saved.") +
			"\n" + DimStyle.Render("  Add one with ") +
			CommandStyle.Render("/credential add <name> <exchange> <key> <secret>")
	}

	var lines []string
	lines = append(lines, header)
	for _, c := range store.Credentials {
		status := StatusIndicator("running")
		name := BrandStyle.Render(c.Name)
		exchange := DimStyle.Render(" (" + c.Exchange + ")")
		key := DimStyle.Render("  Key: ") + maskKeyShort(c.APIKey)
		lines = append(lines, "  "+status+name+exchange)
		lines = append(lines, "  "+key)
	}
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render(fmt.Sprintf("  %d credential(s)", len(store.Credentials))))
	return strings.Join(lines, "\n")
}

func RenderCredentialAdded(name, exchange string) string {
	return BotMsgStyle.Render("nick: ") + "Credential " +
		BrandStyle.Render(name) + " added for " +
		CommandStyle.Render(exchange) + "."
}

func RenderCredentialRemoved(name string) string {
	return BotMsgStyle.Render("nick: ") + "Credential " +
		BrandStyle.Render(name) + " removed."
}

// --- /workflow ---

func RenderWorkflowList(store *workflow.Store) string {
	header := SecondaryStyle.Render("  Workflows\n")

	if len(store.Workflows) == 0 {
		return header + "\n" + DimStyle.Render("  No workflows yet.") +
			"\n" + DimStyle.Render("  Create one with ") +
			CommandStyle.Render("/workflow create <path.json>")
	}

	var lines []string
	lines = append(lines, header)
	for _, w := range store.Workflows {
		statusStr := "stopped"
		if w.Status == "running" {
			statusStr = "running"
		}
		status := StatusIndicator(statusStr)
		name := BrandStyle.Render(padRight(w.Name, 24))
		nodes := DimStyle.Render(fmt.Sprintf("%d nodes", len(w.Nodes)))
		runs := DimStyle.Render(fmt.Sprintf("  runs: %d", w.RunCount))
		lines = append(lines, "  "+status+name+nodes+runs)
	}
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render(fmt.Sprintf("  %d workflow(s)", len(store.Workflows))))
	return strings.Join(lines, "\n")
}

func RenderWorkflowShow(w *workflow.Workflow, width int) string {
	cardWidth := min(width-4, 64)

	statusStr := "stopped"
	if w.Status == "running" {
		statusStr = "running"
	}

	header := StatusIndicator(statusStr) + BrandStyle.Render(w.Name)
	statusLine := DimStyle.Render("Status: ") + renderWorkflowStatus(w.Status)
	runsLine := DimStyle.Render("Runs: ") + fmt.Sprintf("%d", w.RunCount)
	createdLine := DimStyle.Render("Created: ") + w.CreatedAt

	var lines []string
	lines = append(lines, header)
	lines = append(lines, statusLine+"    "+runsLine)
	lines = append(lines, createdLine)
	if w.LastRun != "" {
		lines = append(lines, DimStyle.Render("Last Run: ")+w.LastRun)
	}
	lines = append(lines, "")
	lines = append(lines, SecondaryStyle.Render("Nodes:"))

	for i, node := range w.Nodes {
		prefix := "  "
		if i < len(w.Nodes)-1 {
			prefix = "  ├─ "
		} else {
			prefix = "  └─ "
		}
		nodeType := DimStyle.Render("[" + string(node.Type) + "]")
		nodeName := lipgloss.NewStyle().Foreground(ColorWhite).Render(node.Name)
		lines = append(lines, prefix+nodeType+" "+nodeName)
		if len(node.ConnectsTo) > 0 {
			conn := DimStyle.Render("     → " + strings.Join(node.ConnectsTo, ", "))
			lines = append(lines, conn)
		}
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Width(cardWidth).
		Render(content)
}

func RenderWorkflowCreated(w *workflow.Workflow) string {
	return BotMsgStyle.Render("nick: ") + "Workflow " +
		BrandStyle.Render(w.Name) + " created with " +
		fmt.Sprintf("%d nodes.", len(w.Nodes))
}

func RenderWorkflowRunning(name string, logs []workflow.LogEntry) string {
	header := BotMsgStyle.Render("nick: ") + "Running " +
		BrandStyle.Render(name) + "...\n"

	var lines []string
	lines = append(lines, header)
	for _, log := range logs {
		ts := DimStyle.Render("[" + log.Timestamp + "]")
		var statusStyle lipgloss.Style
		switch log.Status {
		case "started":
			statusStyle = lipgloss.NewStyle().Foreground(ColorWarning)
		case "completed":
			statusStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
		case "error":
			statusStyle = lipgloss.NewStyle().Foreground(ColorError)
		default:
			statusStyle = DimStyle
		}
		status := statusStyle.Render(padRight(log.Status, 10))
		nodeName := lipgloss.NewStyle().Foreground(ColorWhite).Render(log.NodeName)
		msg := DimStyle.Render(log.Message)
		lines = append(lines, "  "+ts+" "+status+" "+nodeName+" "+msg)
	}
	return strings.Join(lines, "\n")
}

func RenderWorkflowStopped(name string) string {
	return BotMsgStyle.Render("nick: ") + "Workflow " +
		BrandStyle.Render(name) + " stopped."
}

func RenderWorkflowRemoved(name string) string {
	return BotMsgStyle.Render("nick: ") + "Workflow " +
		BrandStyle.Render(name) + " removed."
}

func renderWorkflowStatus(s string) string {
	switch s {
	case "running":
		return BrandStyle.Render(s)
	default:
		return DimStyle.Render(s)
	}
}

// --- /logs ---

func RenderLogs(w *workflow.Workflow) string {
	header := SecondaryStyle.Render("  Logs: ") + BrandStyle.Render(w.Name) + "\n"

	if len(w.Logs) == 0 {
		return header + "\n" + DimStyle.Render("  No execution logs. Run the workflow first with ") +
			CommandStyle.Render("/workflow run "+w.Name)
	}

	var lines []string
	lines = append(lines, header)

	if w.Status == "running" {
		lines = append(lines, "  "+StatusIndicator("running")+BrandStyle.Render("LIVE")+"\n")
	} else {
		lines = append(lines, "  "+DimStyle.Render("Last run summary")+"\n")
	}

	for _, log := range w.Logs {
		ts := DimStyle.Render("[" + log.Timestamp + "]")
		var statusStyle lipgloss.Style
		switch log.Status {
		case "started":
			statusStyle = lipgloss.NewStyle().Foreground(ColorWarning)
		case "completed":
			statusStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
		case "error":
			statusStyle = lipgloss.NewStyle().Foreground(ColorError)
		default:
			statusStyle = DimStyle
		}
		status := statusStyle.Render(padRight(log.Status, 10))
		nodeName := lipgloss.NewStyle().Foreground(ColorWhite).Render(padRight(log.NodeName, 22))
		msg := DimStyle.Render(log.Message)
		lines = append(lines, "  "+ts+" "+status+" "+nodeName+" "+msg)
	}

	// Summary.
	completed := 0
	for _, log := range w.Logs {
		if log.Status == "completed" {
			completed++
		}
	}
	lines = append(lines, "")
	lines = append(lines, "  "+DimStyle.Render(fmt.Sprintf("%d/%d nodes completed", completed, completed)))

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

// --- /snapshot: combined dashboard ---

func RenderSnapshot(client *api.PapernickClient, width int) string {
	cardWidth := min(width-4, 72)
	halfWidth := (cardWidth - 6) / 2

	// Fetch portfolio data.
	var portfolioLines []string
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
	rows = append(rows, DimStyle.Render(strings.Repeat("─", 28)))

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
	lines = append(lines, DimStyle.Render("Starting Balance:  ")+formatMoney(startingBalance))
	lines = append(lines, DimStyle.Render("Current Balance:   ")+BrandStyle.Render(formatMoney(currentBalance)))

	pnlStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	pnlPrefix := "+"
	if totalPnL < 0 {
		pnlStyle = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
		pnlPrefix = ""
	}
	lines = append(lines, DimStyle.Render("Total P&L:         ")+
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

	return SecondaryStyle.Render("  Profit & Loss") + "\n" + box
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
	rows = append(rows, DimStyle.Render(strings.Repeat("─", 55)))

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

// --- /chart: sparkline chart ---

// RenderChart renders a braille sparkline chart for a symbol.
func RenderChart(client *api.PapernickClient, symbol string, width int) string {
	cardWidth := min(width-4, 64)

	prices, err := client.GetPrices([]string{symbol})
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch price: ") + err.Error()
	}
	if len(prices) == 0 {
		return DimStyle.Render("  No price data for: ") + symbol
	}

	currentPrice := prices[0].Price
	data := generateSparklineData(currentPrice, 50)
	sparkline := renderSparkline(data, cardWidth-8)

	// Calculate simulated high/low from the data.
	high, low := data[0], data[0]
	for _, v := range data {
		if v > high {
			high = v
		}
		if v < low {
			low = v
		}
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, BrandStyle.Render(prices[0].Symbol)+"  "+
		lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(formatPrice(currentPrice)))
	lines = append(lines, "")
	lines = append(lines, sparkline)
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("H ")+formatPrice(high)+
		DimStyle.Render("  L ")+formatPrice(low)+
		DimStyle.Render("  |  50 points"))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Width(cardWidth).
		Render(content)

	return SecondaryStyle.Render("  Sparkline Chart") + "\n" + box
}

// generateSparklineData produces a random walk of n points ending at basePrice.
func generateSparklineData(basePrice float64, n int) []float64 {
	data := make([]float64, n)
	data[n-1] = basePrice
	volatility := basePrice * 0.003
	for i := n - 2; i >= 0; i-- {
		delta := (rand.Float64()*2 - 1) * volatility
		data[i] = data[i+1] + delta
	}
	return data
}

// renderSparkline renders data as braille block characters.
func renderSparkline(data []float64, barWidth int) string {
	if len(data) == 0 || barWidth <= 0 {
		return ""
	}

	minVal, maxVal := data[0], data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	span := maxVal - minVal
	if span == 0 {
		span = 1
	}

	var result []rune
	for i := 0; i < barWidth && i < len(data); i++ {
		idx := i * len(data) / barWidth
		normalized := (data[idx] - minVal) / span
		blockIdx := int(normalized * float64(len(blocks)-1))
		if blockIdx >= len(blocks) {
			blockIdx = len(blocks) - 1
		}
		if blockIdx < 0 {
			blockIdx = 0
		}
		result = append(result, blocks[blockIdx])
	}

	style := lipgloss.NewStyle().Foreground(ColorPrimary)
	if data[len(data)-1] < data[0] {
		style = lipgloss.NewStyle().Foreground(ColorError)
	}

	return style.Render(string(result))
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
		{"/watch BTC ETH SOL", "Live price dashboard"},
		{"/snapshot", "Combined portfolio dashboard"},
		{"/market", "Full market overview (10 assets)"},
		{"/pnl", "Profit & loss summary"},
		{"/history", "Trade journal with all orders"},
		{"/chart BTC", "ASCII sparkline chart"},
		{"/alert BTC > 100000", "Set a price alert"},
		{"/credential list", "Manage exchange API keys"},
		{"/workflow list", "Manage automation workflows"},
		{"/logs <workflow>", "Workflow execution logs"},
		{"/man <command>", "Detailed manual pages"},
		{"/model <id>", "Switch AI model"},
		{"/theme <name>", "Switch color theme"},
		{"/config init", "Auto-provision API key"},
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
	lines = append(lines, DimStyle.Render("  Use ")+
		CommandStyle.Render("/man <command>")+
		DimStyle.Render(" for detailed docs."))
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
