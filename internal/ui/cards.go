package ui

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/analytics"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/automation"
	"github.com/nickai/cli/internal/config"
	"github.com/nickai/cli/internal/credential"
	"github.com/nickai/cli/internal/backtest"
	"github.com/nickai/cli/internal/indicators"
	"github.com/nickai/cli/internal/journal"
	"github.com/nickai/cli/internal/market"
	"github.com/nickai/cli/internal/mcp"
	"github.com/nickai/cli/internal/mock"
	"github.com/nickai/cli/internal/notify"
	"github.com/nickai/cli/internal/risk"
	"github.com/nickai/cli/internal/strategy"
	"github.com/nickai/cli/internal/trigger"
	"github.com/nickai/cli/internal/workflow"
)

// --- Connect prompt ---

func connectPrompt() string {
	return DimStyle.Render("  No API key configured. ") +
		"Run " + CommandStyle.Render("/config init") +
		DimStyle.Render(" to create an account, or ") +
		CommandStyle.Render("/config set api_key <key>") +
		DimStyle.Render(" if you have one.")
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

func RenderStatusMock(mcpMgr *mcp.ClientManager) string {
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
	num := func(n string) string {
		return lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(n)
	}
	lines := []string{
		BotMsgStyle.Render("nick: ") + "Paper trading account created!",
		"",
		"  " + checkmark + BrandStyle.Render("Connected to PaperNick"),
		"  " + DimStyle.Render("User:    ") + userName,
		"  " + DimStyle.Render("API Key: ") + maskKeyShort(apiKey),
		"  " + DimStyle.Render("You're trading with fake money — experiment freely."),
		"",
		"  " + lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render("Try these:"),
		"",
		"  " + num("1. ") + CommandStyle.Render("/status") + DimStyle.Render("              — see your paper portfolio"),
		"  " + num("2. ") + CommandStyle.Render("/price BTC ETH SOL") + DimStyle.Render("   — check live prices"),
		"  " + num("3. ") + CommandStyle.Render("/buy BTC 0.01") + DimStyle.Render("        — make your first paper trade"),
		"  " + num("4. ") + CommandStyle.Render("/snapshot") + DimStyle.Render("            — full portfolio dashboard"),
		"  " + num("5. ") + CommandStyle.Render("/help") + DimStyle.Render("                — see all commands"),
		"",
		"  " + DimStyle.Render("Want AI chat? Set your Anthropic key:"),
		"  " + CommandStyle.Render("/config set anthropic_key <your-key>"),
	}
	return strings.Join(lines, "\n")
}

// --- /mcp ---

// RenderMCPHelp shows available /mcp subcommands.
func RenderMCPHelp() string {
	header := SecondaryStyle.Render("  /mcp — manage trading integrations\n")
	lines := []string{
		header,
		"  " + CommandStyle.Render("/mcp list") + DimStyle.Render("                — show connected servers & tools"),
		"  " + CommandStyle.Render("/mcp search <query>") + DimStyle.Render("      — browse available servers"),
		"  " + CommandStyle.Render("/mcp info <name>") + DimStyle.Render("         — details on a server"),
		"  " + CommandStyle.Render("/mcp add <name>") + DimStyle.Render("          — install a server from registry"),
		"  " + CommandStyle.Render("/mcp remove <name>") + DimStyle.Render("       — disconnect a server"),
		"  " + CommandStyle.Render("/mcp quick") + DimStyle.Render("               — install all free servers (no keys needed)"),
		"",
		DimStyle.Render("  Try: ") + CommandStyle.Render("/mcp search trading") +
			DimStyle.Render("  or  ") + CommandStyle.Render("/mcp quick"),
	}
	return strings.Join(lines, "\n")
}

// RenderMCPSearchResults shows matching servers from the curated registry.
func RenderMCPSearchResults(results []mcp.RegistryEntry) string {
	lines := []string{SecondaryStyle.Render("  MCP Server Registry\n")}
	for _, e := range results {
		tier := DimStyle.Render("[community]")
		if e.Tier == mcp.TierVerified {
			tier = lipgloss.NewStyle().Foreground(ColorPrimary).Render("[verified]")
		}
		lines = append(lines, "  "+BrandStyle.Render(padRight(e.Name, 16))+
			DimStyle.Render(e.Description)+"  "+tier)
	}
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("  Use ")+
		CommandStyle.Render("/mcp info <name>")+
		DimStyle.Render(" for details, or ")+
		CommandStyle.Render("/mcp add <name>")+
		DimStyle.Render(" to install."))
	return strings.Join(lines, "\n")
}

// RenderMCPInfo shows details for a single registry entry.
func RenderMCPInfo(e *mcp.RegistryEntry) string {
	tier := DimStyle.Render("community")
	if e.Tier == mcp.TierVerified {
		tier = lipgloss.NewStyle().Foreground(ColorPrimary).Render("verified")
	}
	lines := []string{
		SecondaryStyle.Render("  " + e.DisplayName + "\n"),
		"  " + DimStyle.Render(e.Description),
		"",
		"  " + DimStyle.Render("Name:    ") + e.Name,
		"  " + DimStyle.Render("Trust:   ") + tier,
		"  " + DimStyle.Render("Command: ") + CommandStyle.Render(e.Command+" "+strings.Join(e.Args, " ")),
		"  " + DimStyle.Render("Repo:    ") + e.Repo,
	}
	if len(e.EnvKeys) > 0 {
		lines = append(lines, "  "+DimStyle.Render("Env:     ")+strings.Join(e.EnvKeys, ", "))
	}
	caps := make([]string, len(e.Capabilities))
	for i, c := range e.Capabilities {
		caps[i] = string(c)
	}
	lines = append(lines, "  "+DimStyle.Render("Caps:    ")+strings.Join(caps, ", "))
	lines = append(lines, "")
	lines = append(lines, "  "+CommandStyle.Render("/mcp add "+e.Name)+DimStyle.Render("  — install this server"))
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
	// Fetch real price history from Binance, fallback to synthetic.
	var data []float64
	if candles, err := market.FetchKlines(symbol, "1d", 50); err == nil && len(candles) > 0 {
		data = market.ClosePrices(candles)
	} else {
		data = generateSparklineData(currentPrice, 50)
	}
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

	sectionHeader := func(title string) string {
		return "\n  " + lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(title)
	}
	cmdLine := func(cmd, desc string) string {
		return "  " + CommandStyle.Render(padRight(cmd, 28)) + DimStyle.Render(desc)
	}

	var lines []string
	lines = append(lines, header)

	lines = append(lines, sectionHeader("Trading"))
	lines = append(lines, cmdLine("/buy BTC 0.1", "Market buy"))
	lines = append(lines, cmdLine("/sell ETH 1.0", "Market sell"))
	lines = append(lines, cmdLine("/buy BTC 0.1 limit 65000", "Limit buy at price"))
	lines = append(lines, cmdLine("/price BTC ETH", "Live price quotes"))
	lines = append(lines, cmdLine("/watch BTC ETH SOL", "Live price dashboard"))
	lines = append(lines, cmdLine("/alert BTC > 100000", "Set a price alert"))

	lines = append(lines, sectionHeader("Portfolio"))
	lines = append(lines, cmdLine("/status", "Positions & cash balance"))
	lines = append(lines, cmdLine("/orders", "Recent orders & trades"))
	lines = append(lines, cmdLine("/snapshot", "Combined portfolio dashboard"))
	lines = append(lines, cmdLine("/market", "Full market overview (10 assets)"))
	lines = append(lines, cmdLine("/pnl", "Profit & loss summary"))
	lines = append(lines, cmdLine("/history", "Trade journal with all orders"))
	lines = append(lines, cmdLine("/chart BTC", "ASCII sparkline chart"))

	lines = append(lines, sectionHeader("Agents & Automation"))
	lines = append(lines, cmdLine("/agents", "List your trading agents"))
	lines = append(lines, cmdLine("/templates", "Browse marketplace templates"))
	lines = append(lines, cmdLine("/workflow", "Manage automation workflows"))
	lines = append(lines, cmdLine("/trigger add BTC < 60000 sell 0.5", "Conditional trade"))
	lines = append(lines, cmdLine("/risk set max-order 5000", "Risk guardrails"))
	lines = append(lines, cmdLine("/strategy twap ETH buy $2000 4h", "TWAP strategy"))
	lines = append(lines, cmdLine("/auto list", "Automation rules"))
	lines = append(lines, cmdLine("/notify set desktop on", "Desktop notifications"))
	lines = append(lines, cmdLine("/analytics", "Portfolio analytics"))
	lines = append(lines, cmdLine("/analyze BTC", "Market analysis"))
	lines = append(lines, cmdLine("/backtest presets", "Backtest preset strategies"))
	lines = append(lines, cmdLine("/backtest run rsi-reversal BTC", "Run a backtest preset"))
	lines = append(lines, cmdLine("/polymarket scan", "Prediction market analysis"))
	lines = append(lines, cmdLine("/guide", "Interactive guide"))
	lines = append(lines, cmdLine("/logs <workflow>", "Workflow execution logs"))

	lines = append(lines, sectionHeader("Setup & Integrations"))
	lines = append(lines, cmdLine("/config init", "Create account & API key"))
	lines = append(lines, cmdLine("/config", "Manage settings & keys"))
	lines = append(lines, cmdLine("/mcp", "MCP server integrations"))
	lines = append(lines, cmdLine("/credential", "Exchange API keys"))
	lines = append(lines, cmdLine("/model <id>", "Switch AI model"))
	lines = append(lines, cmdLine("/theme <name>", "Switch color theme"))

	lines = append(lines, sectionHeader("General"))
	lines = append(lines, cmdLine("/help", "Show this help"))
	lines = append(lines, cmdLine("/man <command>", "Detailed manual pages"))
	lines = append(lines, cmdLine("/clear", "Clear chat history"))
	lines = append(lines, cmdLine("/quit", "Exit NickAI"))

	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("  Use ")+
		CommandStyle.Render("/man <command>")+
		DimStyle.Render(" for detailed docs. Or just type naturally."))
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

// --- Trigger rendering ---

// RenderTriggerList shows all active triggers in a formatted list.
func RenderTriggerList(triggers []trigger.Trigger) string {
	var lines []string
	lines = append(lines, SecondaryStyle.Render("  Active Triggers\n"))
	for _, t := range triggers {
		condition := BrandStyle.Render(t.Symbol) + " " + t.Operator + " " + formatPrice(t.Target)
		action := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(
			strings.ToUpper(t.Action.Side) + " " + fmt.Sprintf("%g", t.Action.Quantity) + " " + t.Symbol)
		id := DimStyle.Render("  [" + t.ID[:6] + "]")
		lines = append(lines, "  "+StatusIndicator("running")+
			DimStyle.Render("if ")+condition+DimStyle.Render(" → ")+action+id)
	}
	return strings.Join(lines, "\n")
}

// RenderTriggerConfirm renders a confirmation card for a fired trigger.
func RenderTriggerConfirm(t trigger.Trigger, currentPrice float64) string {
	return WarningStyle.Render("  TRIGGER FIRED ") + "\n" +
		"  " + BrandStyle.Render(t.Symbol) + " hit " + formatPrice(currentPrice) +
		DimStyle.Render(fmt.Sprintf("  (condition: %s %s)", t.Operator, formatPrice(t.Target))) + "\n" +
		"  " + lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(
			strings.ToUpper(t.Action.Side)+" "+fmt.Sprintf("%g", t.Action.Quantity)+" "+t.Symbol+" ("+t.Action.Type+")") + "\n" +
		DimStyle.Render("  Press y to execute, n to skip")
}

// --- Risk rendering ---

// RenderRiskLimits displays current risk limits.
func RenderRiskLimits(limits *risk.RiskLimits) string {
	if limits == nil || limits.IsEmpty() {
		return DimStyle.Render("  No risk limits set.") + "\n" +
			DimStyle.Render("  Set with ") + CommandStyle.Render("/risk set max-order 5000")
	}

	var lines []string
	lines = append(lines, SecondaryStyle.Render("  Risk Guardrails\n"))

	if limits.MaxOrderValue > 0 {
		lines = append(lines, "  "+StatusIndicator("running")+
			DimStyle.Render("Max Order Value:    ")+
			BrandStyle.Render(fmt.Sprintf("$%.0f", limits.MaxOrderValue)))
	}
	if limits.MaxPositionPct > 0 {
		lines = append(lines, "  "+StatusIndicator("running")+
			DimStyle.Render("Max Position:       ")+
			BrandStyle.Render(fmt.Sprintf("%.0f%%", limits.MaxPositionPct)))
	}
	if limits.DailyLossPct > 0 {
		lines = append(lines, "  "+StatusIndicator("running")+
			DimStyle.Render("Daily Loss Limit:   ")+
			BrandStyle.Render(fmt.Sprintf("%.0f%%", limits.DailyLossPct)))
	}
	return strings.Join(lines, "\n")
}

// RenderRiskHelp shows /risk usage.
func RenderRiskHelp() string {
	header := SecondaryStyle.Render("  /risk — portfolio risk guardrails\n")
	lines := []string{
		header,
		"  " + CommandStyle.Render("/risk show") + DimStyle.Render("                     — view current limits"),
		"  " + CommandStyle.Render("/risk set max-order <$>") + DimStyle.Render("        — max single order value"),
		"  " + CommandStyle.Render("/risk set max-position <%>") + DimStyle.Render("     — max position as % of portfolio"),
		"  " + CommandStyle.Render("/risk set daily-loss <%>") + DimStyle.Render("       — daily loss limit %"),
		"  " + CommandStyle.Render("/risk clear") + DimStyle.Render("                    — remove all limits"),
	}
	return strings.Join(lines, "\n")
}

// --- Strategy rendering ---

// RenderStrategyList shows all TWAP strategies.
func RenderStrategyList(strategies []strategy.TWAPStrategy) string {
	if len(strategies) == 0 {
		return DimStyle.Render("  No strategies.") + "\n" +
			DimStyle.Render("  Create one with ") + CommandStyle.Render("/strategy twap ETH buy $2000 4h")
	}

	var lines []string
	lines = append(lines, SecondaryStyle.Render("  TWAP Strategies\n"))

	for _, s := range strategies {
		statusStr := "running"
		if s.Status != "active" {
			statusStr = "stopped"
		}
		status := StatusIndicator(statusStr)

		progress := fmt.Sprintf("%d/%d", s.Executed, s.SliceCount)
		side := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(strings.ToUpper(s.Side))
		if s.Side == "sell" {
			side = lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render("SELL")
		}

		lines = append(lines, "  "+status+
			side+" "+BrandStyle.Render(s.Symbol)+
			DimStyle.Render(fmt.Sprintf("  $%.0f over %s  [%s]  ", s.TotalValue, s.Duration, progress))+
			renderStrategyStatus(s.Status)+
			DimStyle.Render("  ["+s.ID[:6]+"]"))

		if s.Status == "active" {
			nextStr := s.NextSliceAt.Format("3:04 PM")
			lines = append(lines, "    "+DimStyle.Render("Next slice: ")+nextStr+
				DimStyle.Render(fmt.Sprintf("  ($%.2f/slice)", s.SliceValue)))
		}
	}
	return strings.Join(lines, "\n")
}

func renderStrategyStatus(s string) string {
	switch s {
	case "active":
		return BrandStyle.Render(s)
	case "completed":
		return lipgloss.NewStyle().Foreground(ColorPrimary).Render(s)
	case "cancelled":
		return DimStyle.Render(s)
	default:
		return DimStyle.Render(s)
	}
}

// RenderStrategySliceConfirm renders a confirmation card for a TWAP slice.
func RenderStrategySliceConfirm(s strategy.TWAPStrategy, price float64) string {
	qty := s.SliceValue / price
	side := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(strings.ToUpper(s.Side))
	if s.Side == "sell" {
		side = lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render("SELL")
	}

	return WarningStyle.Render("  TWAP SLICE ") +
		DimStyle.Render(fmt.Sprintf(" (%d/%d)", s.Executed+1, s.SliceCount)) + "\n" +
		"  " + side + " " + BrandStyle.Render(s.Symbol) +
		DimStyle.Render(fmt.Sprintf("  %.4f @ %s ≈ $%.2f", qty, formatPrice(price), s.SliceValue)) + "\n" +
		DimStyle.Render("  Press y to execute, n to skip")
}

// RenderStrategyHelp shows /strategy usage.
func RenderStrategyHelp() string {
	header := SecondaryStyle.Render("  /strategy — TWAP execution strategies\n")
	lines := []string{
		header,
		"  " + CommandStyle.Render("/strategy twap <SYM> <buy|sell> $<VALUE> <DURATION>") + DimStyle.Render(" — create TWAP"),
		"  " + CommandStyle.Render("/strategy list") + DimStyle.Render("                                      — show strategies"),
		"  " + CommandStyle.Render("/strategy cancel <id>") + DimStyle.Render("                               — cancel strategy"),
		"",
		DimStyle.Render("  Duration examples: 4h, 1h, 30m, 2h30m"),
	}
	return strings.Join(lines, "\n")
}

// --- Notification rendering ---

// RenderNotifyConfig displays the current notification configuration.
func RenderNotifyConfig(cfg *notify.Config) string {
	if cfg == nil || cfg.IsEmpty() {
		return DimStyle.Render("  No notification channels configured.") + "\n" +
			DimStyle.Render("  Set with ") + CommandStyle.Render("/notify set desktop on")
	}

	var lines []string
	lines = append(lines, SecondaryStyle.Render("  Notification Settings\n"))

	indicator := func(enabled bool) string {
		if enabled {
			return StatusIndicator("running")
		}
		return StatusIndicator("stopped")
	}

	lines = append(lines, "  "+indicator(cfg.Desktop)+
		DimStyle.Render("Desktop:  ")+boolStr(cfg.Desktop))
	lines = append(lines, "  "+indicator(cfg.Sound)+
		DimStyle.Render("Sound:    ")+boolStr(cfg.Sound))

	webhookStr := DimStyle.Render("(none)")
	if cfg.WebhookURL != "" {
		webhookStr = BrandStyle.Render(cfg.WebhookURL)
	}
	lines = append(lines, "  "+indicator(cfg.WebhookURL != "")+
		DimStyle.Render("Webhook:  ")+webhookStr)

	return strings.Join(lines, "\n")
}

func boolStr(v bool) string {
	if v {
		return BrandStyle.Render("on")
	}
	return DimStyle.Render("off")
}

// RenderNotifyHelp shows /notify usage.
func RenderNotifyHelp() string {
	header := SecondaryStyle.Render("  /notify — desktop & webhook notifications\n")
	lines := []string{
		header,
		"  " + CommandStyle.Render("/notify show") + DimStyle.Render("                      — view settings"),
		"  " + CommandStyle.Render("/notify set desktop on|off") + DimStyle.Render("        — toggle desktop alerts"),
		"  " + CommandStyle.Render("/notify set sound on|off") + DimStyle.Render("          — toggle sound"),
		"  " + CommandStyle.Render("/notify set webhook <url>") + DimStyle.Render("         — set webhook URL"),
		"  " + CommandStyle.Render("/notify clear") + DimStyle.Render("                     — reset all settings"),
		"  " + CommandStyle.Render("/notify test") + DimStyle.Render("                      — send a test notification"),
	}
	return strings.Join(lines, "\n")
}

// --- Analytics rendering ---

// RenderAnalytics displays portfolio analytics with metrics and allocation chart.
func RenderAnalytics(client *api.PapernickClient, width int) string {
	cardWidth := min(width-4, 64)

	// Load journal entries.
	entries, _ := journal.All()

	// Load portfolio.
	portfolio, err := client.GetPortfolio()
	if err != nil {
		return ErrorStyle.Render("  Failed to load portfolio: ") + err.Error()
	}

	// Build price map.
	symbolSet := make(map[string]bool)
	for _, e := range entries {
		base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(e.Symbol, "USDT"), "USDC"), "USD")
		symbolSet[base] = true
	}
	for _, a := range portfolio.Assets {
		base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(a.Symbol, "USDT"), "USDC"), "USD")
		symbolSet[base] = true
	}
	symbols := make([]string, 0, len(symbolSet))
	for s := range symbolSet {
		symbols = append(symbols, s)
	}
	priceMap := make(map[string]float64)
	if len(symbols) > 0 {
		if prices, err := client.GetPrices(symbols); err == nil {
			for _, p := range prices {
				priceMap[p.Symbol] = p.Price
				base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(p.Symbol, "USDT"), "USDC"), "USD")
				priceMap[base] = p.Price
			}
		}
	}

	metrics := analytics.Calculate(entries, priceMap)
	allocs := analytics.CalcAllocation(portfolio)

	// Build metrics display.
	var lines []string
	lines = append(lines, "")

	// Key metrics row.
	sharpeColor := ColorPrimary
	if metrics.SharpeRatio < 0 {
		sharpeColor = ColorError
	}
	lines = append(lines,
		DimStyle.Render("Sharpe Ratio:   ")+lipgloss.NewStyle().Foreground(sharpeColor).Bold(true).Render(fmt.Sprintf("%.2f", metrics.SharpeRatio))+
			DimStyle.Render("        Win Rate:      ")+lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(fmt.Sprintf("%.1f%%", metrics.WinRate)))
	lines = append(lines,
		DimStyle.Render("Max Drawdown:   ")+ErrorStyle.Render(fmt.Sprintf("%.1f%%", metrics.MaxDrawdownPct))+
			DimStyle.Render("        Profit Factor: ")+BrandStyle.Render(fmt.Sprintf("%.2f", metrics.ProfitFactor)))
	lines = append(lines,
		DimStyle.Render("Total Trades:   ")+fmt.Sprintf("%d", metrics.TotalTrades)+
			DimStyle.Render("            W/L:           ")+fmt.Sprintf("%d/%d", metrics.WinCount, metrics.LossCount))

	pnlStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
	if metrics.TotalPnL < 0 {
		pnlStyle = lipgloss.NewStyle().Foreground(ColorError)
	}
	lines = append(lines,
		DimStyle.Render("Total P&L:      ")+pnlStyle.Bold(true).Render(fmt.Sprintf("$%.2f", metrics.TotalPnL)))

	if metrics.WinCount > 0 || metrics.LossCount > 0 {
		lines = append(lines,
			DimStyle.Render("Avg Win:        ")+BrandStyle.Render(fmt.Sprintf("$%.2f", metrics.AvgWin))+
				DimStyle.Render("       Avg Loss:       ")+ErrorStyle.Render(fmt.Sprintf("$%.2f", metrics.AvgLoss)))
		lines = append(lines,
			DimStyle.Render("Best Trade:     ")+BrandStyle.Render(fmt.Sprintf("$%.2f", metrics.BestTrade))+
				DimStyle.Render("     Worst Trade:    ")+ErrorStyle.Render(fmt.Sprintf("$%.2f", metrics.WorstTrade)))
	}

	// Allocation bar chart.
	if len(allocs) > 0 {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("Allocation"))
		barWidth := cardWidth - 24
		if barWidth < 10 {
			barWidth = 10
		}
		for _, a := range allocs {
			filled := int(a.Percent / 100 * float64(barWidth))
			if filled < 0 {
				filled = 0
			}
			if filled > barWidth {
				filled = barWidth
			}
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			lines = append(lines,
				"  "+padRight(a.Symbol, 6)+
					lipgloss.NewStyle().Foreground(ColorPrimary).Render(bar)+
					DimStyle.Render(fmt.Sprintf(" %5.1f%% $%.0f", a.Percent, a.Value)))
		}
	}

	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Width(cardWidth).
		Render(content)

	return SecondaryStyle.Render("  Portfolio Analytics") + "\n" + box
}

// --- Market Analysis rendering ---

// RenderAnalysis displays technical analysis for a symbol.
func RenderAnalysis(client *api.PapernickClient, symbol string, width int) string {
	cardWidth := min(width-4, 64)
	symbol = strings.ToUpper(symbol)

	// Fetch current price.
	prices, err := client.GetPrices([]string{symbol})
	if err != nil || len(prices) == 0 {
		return ErrorStyle.Render("  Failed to fetch price for ") + symbol
	}
	currentPrice := prices[0].Price

	// Fetch real price history from Binance, fallback to synthetic.
	var history []float64
	if candles, err := market.FetchKlines(symbol, "1d", 50); err == nil && len(candles) > 0 {
		history = market.ClosePrices(candles)
	} else {
		history = generateSparklineData(currentPrice, 50)
	}

	// Fear & Greed.
	fg, fgLabel, _ := indicators.FetchFearGreed()

	a := indicators.Analyze(symbol, currentPrice, history, fg, fgLabel)

	// Build display.
	var lines []string
	lines = append(lines, "")
	lines = append(lines,
		BrandStyle.Render(symbol)+"  "+
			lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(formatPrice(currentPrice)))
	lines = append(lines, "")

	// Indicator badges.
	badge := func(label, value, signal string) string {
		signalColor := ColorDim
		switch signal {
		case "overbought", "bearish", "above":
			signalColor = ColorError
		case "oversold", "bullish", "below":
			signalColor = ColorPrimary
		}
		return DimStyle.Render(padRight(label, 14)) +
			lipgloss.NewStyle().Foreground(ColorWhite).Render(value) +
			"  " + lipgloss.NewStyle().Foreground(signalColor).Bold(true).Render(signal)
	}

	lines = append(lines, badge("RSI (14)", fmt.Sprintf("%.1f", a.RSI), a.RSISignal))
	lines = append(lines, badge("MACD", fmt.Sprintf("%.2f", a.MACD), a.MACDTrend))
	lines = append(lines, badge("Bollinger", fmt.Sprintf("%.0f / %.0f", a.BollingerLower, a.BollingerUpper), a.BollingerPos))
	lines = append(lines, badge("SMA 20", formatPrice(a.SMA20), ""))
	if a.SMA50 > 0 {
		lines = append(lines, badge("SMA 50", formatPrice(a.SMA50), ""))
	}
	lines = append(lines, badge("Trend", a.Trend, a.Trend))

	// Fear & Greed.
	if a.FearGreedLabel != "" {
		fgColor := ColorDim
		switch {
		case a.FearGreed <= 25:
			fgColor = ColorError
		case a.FearGreed >= 75:
			fgColor = ColorPrimary
		case a.FearGreed >= 50:
			fgColor = ColorWarning
		}
		lines = append(lines, "")
		lines = append(lines, DimStyle.Render("Fear & Greed:   ")+
			lipgloss.NewStyle().Foreground(fgColor).Bold(true).Render(
				fmt.Sprintf("%d/100 (%s)", a.FearGreed, a.FearGreedLabel)))
	}

	// Sparkline.
	lines = append(lines, "")
	sparkWidth := cardWidth - 6
	if sparkWidth > 40 {
		sparkWidth = 40
	}
	lines = append(lines, "  "+renderSparkline(history, sparkWidth))

	// Summary.
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render(a.Summary))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Width(cardWidth).
		Render(content)

	return SecondaryStyle.Render("  Market Analysis") + "\n" + box
}

// --- Automation rendering ---

// RenderAutoList shows all automation rules.
func RenderAutoList(rules []automation.AutoRule) string {
	if len(rules) == 0 {
		return DimStyle.Render("  No automation rules.") + "\n" +
			DimStyle.Render("  Ask the AI to create one, e.g. ") +
			CommandStyle.Render("\"buy $100 of BTC every day\"")
	}

	var lines []string
	lines = append(lines, SecondaryStyle.Render("  Automation Rules\n"))

	for _, r := range rules {
		statusStr := "running"
		if r.Status != "active" {
			statusStr = "stopped"
		}
		status := StatusIndicator(statusStr)

		typeTag := DimStyle.Render("[" + string(r.Type) + "]")
		desc := lipgloss.NewStyle().Foreground(ColorWhite).Render(r.Description)
		idStr := DimStyle.Render("  [" + r.ID[:6] + "]")

		lines = append(lines, "  "+status+desc+idStr)

		// Details line.
		details := "    " + typeTag
		if r.Schedule != "" {
			details += DimStyle.Render("  every ") + r.Schedule
		}
		action := fmt.Sprintf("  %s %s $%.0f", strings.ToUpper(r.Action), r.ActionSymbol, r.ActionValue)
		details += lipgloss.NewStyle().Foreground(ColorSecondary).Render(action)
		if r.FireCount > 0 {
			details += DimStyle.Render(fmt.Sprintf("  (%d fires)", r.FireCount))
		}
		lines = append(lines, details)
	}
	return strings.Join(lines, "\n")
}

// RenderAutoConfirm renders a confirmation card for an automation fire.
func RenderAutoConfirm(rule automation.AutoRule, price float64) string {
	side := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(strings.ToUpper(rule.Action))
	if rule.Action == "sell" || rule.Action == "sell_all" {
		side = lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render(strings.ToUpper(rule.Action))
	}

	qty := 0.0
	if price > 0 {
		qty = rule.ActionValue / price
	}

	return WarningStyle.Render("  AUTOMATION FIRED ") + "\n" +
		"  " + DimStyle.Render(rule.Description) + "\n" +
		"  " + side + " " + BrandStyle.Render(rule.ActionSymbol) +
		DimStyle.Render(fmt.Sprintf("  %.4f @ %s ≈ $%.0f", qty, formatPrice(price), rule.ActionValue)) + "\n" +
		DimStyle.Render("  Press y to execute, n to skip")
}

// RenderAutoHelp shows /auto usage.
func RenderAutoHelp() string {
	header := SecondaryStyle.Render("  /auto — natural language automation\n")
	lines := []string{
		header,
		"  " + CommandStyle.Render("/auto list") + DimStyle.Render("                    — view all rules"),
		"  " + CommandStyle.Render("/auto pause <id>") + DimStyle.Render("              — pause a rule"),
		"  " + CommandStyle.Render("/auto resume <id>") + DimStyle.Render("             — resume a rule"),
		"  " + CommandStyle.Render("/auto remove <id>") + DimStyle.Render("             — delete a rule"),
		"",
		DimStyle.Render("  Create rules by asking the AI:"),
		"  " + CommandStyle.Render("\"buy $100 of BTC every day\""),
		"  " + CommandStyle.Render("\"sell ETH if it goes above 5000\""),
	}
	return strings.Join(lines, "\n")
}

// --- Backtest rendering ---

// RenderBacktestCard renders backtest results as a styled card.
func RenderBacktestCard(result *backtest.Result) string {
	if result == nil {
		return ErrorStyle.Render("  No backtest results.")
	}

	cardWidth := 64

	var lines []string
	lines = append(lines, "")

	// Header.
	name := result.Strategy
	if name == "" {
		name = "Custom Strategy"
	}
	lines = append(lines, BrandStyle.Render(name)+"  "+
		lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(result.Symbol)+
		DimStyle.Render("  "+result.Period))
	lines = append(lines, "")

	// Metrics table.
	metricLine := func(label string, value string) string {
		return DimStyle.Render(padRight("  "+label, 22)) +
			lipgloss.NewStyle().Foreground(ColorWhite).Render(value)
	}

	returnColor := ColorPrimary
	if result.TotalReturn < 0 {
		returnColor = ColorError
	}

	lines = append(lines, metricLine("Total Trades", fmt.Sprintf("%d", result.TotalTrades)))
	lines = append(lines, metricLine("Win Rate", fmt.Sprintf("%.1f%%", result.WinRate)))
	lines = append(lines, DimStyle.Render(padRight("  Total Return", 22))+
		lipgloss.NewStyle().Foreground(returnColor).Bold(true).Render(fmt.Sprintf("%.2f%%", result.TotalReturn)))
	lines = append(lines, metricLine("Sharpe Ratio", fmt.Sprintf("%.2f", result.SharpeRatio)))
	lines = append(lines, metricLine("Max Drawdown", fmt.Sprintf("%.2f%%", result.MaxDrawdown)))
	if !math.IsInf(result.ProfitFactor, 0) {
		lines = append(lines, metricLine("Profit Factor", fmt.Sprintf("%.2f", result.ProfitFactor)))
	}

	// Best/Worst trade.
	if result.TotalTrades > 0 {
		lines = append(lines, "")
		bestColor := ColorPrimary
		if result.BestTrade < 0 {
			bestColor = ColorError
		}
		worstColor := ColorError
		if result.WorstTrade > 0 {
			worstColor = ColorPrimary
		}
		lines = append(lines, DimStyle.Render(padRight("  Best Trade", 22))+
			lipgloss.NewStyle().Foreground(bestColor).Render(fmt.Sprintf("%+.2f%%", result.BestTrade)))
		lines = append(lines, DimStyle.Render(padRight("  Worst Trade", 22))+
			lipgloss.NewStyle().Foreground(worstColor).Render(fmt.Sprintf("%+.2f%%", result.WorstTrade)))
	}

	// Equity curve sparkline.
	if len(result.EquityCurve) > 2 {
		lines = append(lines, "")
		sparkWidth := cardWidth - 6
		if sparkWidth > 40 {
			sparkWidth = 40
		}
		lines = append(lines, "  "+renderSparkline(result.EquityCurve, sparkWidth))
		lines = append(lines, DimStyle.Render("  Equity curve"))
	}

	// Trade list (last 10).
	if len(result.Trades) > 0 {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("  Recent Trades"))
		showCount := len(result.Trades)
		startIdx := 0
		if showCount > 10 {
			startIdx = showCount - 10
			lines = append(lines, DimStyle.Render(fmt.Sprintf("  (showing last 10 of %d)", showCount)))
		}
		for i := startIdx; i < showCount; i++ {
			t := result.Trades[i]
			pnlColor := ColorPrimary
			if t.PnLPct < 0 {
				pnlColor = ColorError
			}
			entry := t.EntryTime.Format("Jan 02")
			exit := t.ExitTime.Format("Jan 02")
			pnl := lipgloss.NewStyle().Foreground(pnlColor).Render(fmt.Sprintf("%+.2f%%", t.PnLPct))
			reason := DimStyle.Render(t.Reason)
			lines = append(lines, fmt.Sprintf("  %s → %s  %s  %s", entry, exit, pnl, reason))
		}
	}

	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Width(cardWidth).
		Render(content)

	return SecondaryStyle.Render("  Backtest Results") + "\n" + box
}

// RenderBacktestHelp shows /backtest usage.
func RenderBacktestHelp() string {
	header := SecondaryStyle.Render("  /backtest — strategy backtesting\n")
	lines := []string{
		header,
		"  " + CommandStyle.Render("/backtest presets") + DimStyle.Render("                — list preset strategies"),
		"  " + CommandStyle.Render("/backtest run <preset> <symbol>") + DimStyle.Render(" — run a preset"),
		"  " + CommandStyle.Render("/backtest run momentum ETH 90d") + DimStyle.Render(" — preset with custom period"),
		"",
		DimStyle.Render("  Or describe a custom strategy:"),
		"  " + CommandStyle.Render("/backtest BTC RSI below 30 with 5% stop loss"),
		"  " + CommandStyle.Render("\"backtest buying ETH when RSI drops below 30\""),
		"",
		DimStyle.Render("  Available presets: rsi-reversal, macd-crossover, bollinger-bounce,"),
		DimStyle.Render("  golden-cross, momentum, fear-and-greed, dip-buyer"),
	}
	return strings.Join(lines, "\n")
}

// RenderBacktestPresets shows all available preset strategies.
func RenderBacktestPresets() string {
	presets := backtest.GetPresets()

	var lines []string
	lines = append(lines, SecondaryStyle.Render("  Backtest Presets\n"))

	for _, p := range presets {
		name := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(padRight(p.Strategy.Name, 20))
		desc := DimStyle.Render(p.Description)
		lines = append(lines, "  "+name+desc)

		// Details.
		details := "    "
		if p.Strategy.StopLossPct > 0 {
			details += DimStyle.Render(fmt.Sprintf("SL: %.0f%%", p.Strategy.StopLossPct))
		}
		if p.Strategy.TakeProfitPct > 0 {
			details += DimStyle.Render(fmt.Sprintf("  TP: %.0f%%", p.Strategy.TakeProfitPct))
		}
		lines = append(lines, details)
	}

	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("  Run: ")+
		CommandStyle.Render("/backtest run <name> <symbol> [period]"))
	lines = append(lines, DimStyle.Render("  Example: ")+
		CommandStyle.Render("/backtest run rsi-reversal BTC 90d"))

	return strings.Join(lines, "\n")
}

// --- Guide rendering ---

// RenderGuideCard renders an interactive guide section.
func RenderGuideCard(section string) string {
	switch section {
	case "start", "":
		return renderGuideStart()
	case "trading":
		return renderGuideTrading()
	case "analysis":
		return renderGuideAnalysis()
	case "backtest":
		return renderGuideBacktest()
	case "ai":
		return renderGuideAI()
	case "mcp":
		return renderGuideMCP()
	case "risk":
		return renderGuideRisk()
	case "polymarket":
		return renderGuidePolymarket()
	default:
		return ErrorStyle.Render("  Unknown guide section: ") + section + "\n" +
			DimStyle.Render("  Available: start, trading, analysis, backtest, ai, mcp, risk, polymarket")
	}
}

func renderGuideStart() string {
	lines := []string{
		SecondaryStyle.Render("  NickAI Guide\n"),
		lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("  Welcome to NickAI!"),
		DimStyle.Render("  Your AI trading analyst in the terminal.\n"),
		DimStyle.Render("  Sections:"),
		"  " + CommandStyle.Render("/guide trading") + DimStyle.Render("      — paper trading basics"),
		"  " + CommandStyle.Render("/guide analysis") + DimStyle.Render("     — technical analysis + charts"),
		"  " + CommandStyle.Render("/guide backtest") + DimStyle.Render("     — backtesting strategies"),
		"  " + CommandStyle.Render("/guide ai") + DimStyle.Render("           — talking to Nick effectively"),
		"  " + CommandStyle.Render("/guide mcp") + DimStyle.Render("          — connecting external tools"),
		"  " + CommandStyle.Render("/guide risk") + DimStyle.Render("         — setting up guardrails"),
		"  " + CommandStyle.Render("/guide polymarket") + DimStyle.Render("   — prediction market analysis"),
		"",
		DimStyle.Render("  Start with: ") + CommandStyle.Render("/guide trading"),
	}
	return strings.Join(lines, "\n")
}

func renderGuideTrading() string {
	lines := []string{
		SecondaryStyle.Render("  Paper Trading Basics\n"),
		"  " + DimStyle.Render("1.") + " Check prices:              " + CommandStyle.Render("/price BTC ETH SOL"),
		"  " + DimStyle.Render("2.") + " Buy an asset:              " + CommandStyle.Render("/buy BTC 0.1"),
		"  " + DimStyle.Render("3.") + " Check your portfolio:      " + CommandStyle.Render("/status"),
		"  " + DimStyle.Render("4.") + " View trade history:        " + CommandStyle.Render("/history"),
		"  " + DimStyle.Render("5.") + " Sell when ready:           " + CommandStyle.Render("/sell BTC 0.05"),
		"",
		DimStyle.Render("  Or just ask Nick:"),
		"  " + CommandStyle.Render("\"should I buy ETH right now?\""),
		"  " + CommandStyle.Render("\"buy $500 of SOL\""),
		"",
		DimStyle.Render("  Next: ") + CommandStyle.Render("/guide analysis"),
	}
	return strings.Join(lines, "\n")
}

func renderGuideAnalysis() string {
	lines := []string{
		SecondaryStyle.Render("  Technical Analysis\n"),
		"  " + DimStyle.Render("1.") + " Quick analysis:            " + CommandStyle.Render("/analyze BTC"),
		"  " + DimStyle.Render("2.") + " Sparkline chart:           " + CommandStyle.Render("/chart ETH"),
		"  " + DimStyle.Render("3.") + " Portfolio analytics:       " + CommandStyle.Render("/analytics"),
		"  " + DimStyle.Render("4.") + " Market overview:           " + CommandStyle.Render("/market"),
		"",
		DimStyle.Render("  Analysis uses real Binance OHLCV data with RSI, MACD,"),
		DimStyle.Render("  Bollinger Bands, SMAs, and Fear & Greed Index."),
		"",
		DimStyle.Render("  Next: ") + CommandStyle.Render("/guide backtest"),
	}
	return strings.Join(lines, "\n")
}

func renderGuideBacktest() string {
	lines := []string{
		SecondaryStyle.Render("  Backtesting Strategies\n"),
		"  " + DimStyle.Render("1.") + " Try a preset strategy:",
		"     " + CommandStyle.Render("/backtest run rsi-reversal BTC"),
		"",
		"  " + DimStyle.Render("2.") + " See all presets:",
		"     " + CommandStyle.Render("/backtest presets"),
		"",
		"  " + DimStyle.Render("3.") + " Describe your own strategy:",
		"     " + CommandStyle.Render("\"backtest buying ETH when RSI drops below 30\""),
		"",
		"  " + DimStyle.Render("4.") + " Iterate with Nick:",
		"     " + CommandStyle.Render("\"add a 5% stop loss and try again\""),
		"",
		DimStyle.Render("  Next: ") + CommandStyle.Render("/guide risk"),
	}
	return strings.Join(lines, "\n")
}

func renderGuideAI() string {
	lines := []string{
		SecondaryStyle.Render("  Talking to Nick\n"),
		DimStyle.Render("  Nick understands natural language. Try:"),
		"",
		"  " + CommandStyle.Render("\"What's your take on ETH?\""),
		"  " + CommandStyle.Render("\"Build me a diversified crypto portfolio\""),
		"  " + CommandStyle.Render("\"Rebalance to 50% BTC 30% ETH 20% SOL\""),
		"  " + CommandStyle.Render("\"DCA $100 into BTC daily\""),
		"  " + CommandStyle.Render("\"Backtest buying dips with RSI and Fear & Greed\""),
		"",
		DimStyle.Render("  Nick always asks for confirmation before trading."),
		"",
		DimStyle.Render("  Next: ") + CommandStyle.Render("/guide mcp"),
	}
	return strings.Join(lines, "\n")
}

func renderGuideMCP() string {
	lines := []string{
		SecondaryStyle.Render("  Connecting External Tools (MCP)\n"),
		DimStyle.Render("  MCP servers give Nick extra powers:\n"),
		"  " + CommandStyle.Render("/mcp add defillama") + DimStyle.Render("    — DeFi yields & TVL (free)"),
		"  " + CommandStyle.Render("/mcp add tradingview") + DimStyle.Render("  — charts & screeners (free)"),
		"  " + CommandStyle.Render("/mcp add onchain") + DimStyle.Render("      — on-chain data (free)"),
		"  " + CommandStyle.Render("/mcp add brave-search") + DimStyle.Render(" — web search for sentiment"),
		"  " + CommandStyle.Render("/mcp add ccxt") + DimStyle.Render("         — live exchange trading"),
		"",
		"  " + CommandStyle.Render("/mcp quick") + DimStyle.Render("            — install all free servers"),
		"  " + CommandStyle.Render("/mcp list") + DimStyle.Render("             — see what's connected"),
		"",
		DimStyle.Render("  Next: ") + CommandStyle.Render("/guide risk"),
	}
	return strings.Join(lines, "\n")
}

func renderGuideRisk() string {
	lines := []string{
		SecondaryStyle.Render("  Risk Guardrails\n"),
		DimStyle.Render("  Set limits before going live:\n"),
		"  " + CommandStyle.Render("/risk set max_order_value 5000") + DimStyle.Render("  — $5K per order cap"),
		"  " + CommandStyle.Render("/risk set max_position_pct 25") + DimStyle.Render("   — 25% max per asset"),
		"  " + CommandStyle.Render("/risk set daily_loss_pct 5") + DimStyle.Render("      — stop if down 5%"),
		"  " + CommandStyle.Render("/risk show") + DimStyle.Render("                          — view current limits"),
		"",
		DimStyle.Render("  All trades (manual & AI) are checked against these limits."),
		DimStyle.Render("  MCP trade tools also require confirmation."),
		"",
		DimStyle.Render("  Next: ") + CommandStyle.Render("/guide polymarket"),
	}
	return strings.Join(lines, "\n")
}

func renderGuidePolymarket() string {
	lines := []string{
		SecondaryStyle.Render("  Prediction Market Analysis\n"),
		DimStyle.Render("  Analyze Polymarket events with AI:\n"),
		"  " + CommandStyle.Render("/polymarket scan") + DimStyle.Render("          — find mispriced contracts"),
		"  " + CommandStyle.Render("/polymarket analyze <event>") + DimStyle.Render(" — deep dive on an event"),
		"  " + CommandStyle.Render("/polymarket hot") + DimStyle.Render("            — trending events"),
		"",
		DimStyle.Render("  Requires MCP servers:"),
		"  " + CommandStyle.Render("/mcp add polymarket"),
		"  " + CommandStyle.Render("/mcp add brave-search"),
		"",
		DimStyle.Render("  That's the guide! Type anything to start trading."),
	}
	return strings.Join(lines, "\n")
}
