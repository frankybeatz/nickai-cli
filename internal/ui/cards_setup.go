package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/config"
	"github.com/nickai/cli/internal/credential"
	"github.com/nickai/cli/internal/memory"
	"github.com/nickai/cli/internal/mcp"
	"github.com/nickai/cli/internal/workflow"
)

// --- /config ---

func storageLabel(storage string) string {
	switch storage {
	case "keyring":
		return DimStyle.Render(" (keyring)")
	case "env":
		return DimStyle.Render(" (from env)")
	case "config":
		return DimStyle.Render(" (config file)")
	default:
		return ""
	}
}

func RenderConfigShow(cfg *config.Config) string {
	header := SectionHeader("Configuration")

	apiStorage := cfg.KeyStorage("api_key")
	anthropicStorage := cfg.KeyStorage("anthropic_key")

	anthropicStatus := cfg.MaskedAnthropicKey()
	if anthropicStorage == "(not set)" {
		anthropicStatus = "(not set)"
	} else {
		anthropicStatus += storageLabel(anthropicStorage)
	}

	apiStatus := cfg.MaskedKey()
	if apiStorage != "(not set)" {
		apiStatus += storageLabel(apiStorage)
	}

	keyringLine := "  " + DimStyle.Render("Keyring:        ")
	if config.UseKeyring() {
		keyringLine += "available (secrets stored securely)"
	} else {
		keyringLine += "unavailable (using config file)"
	}

	lines := []string{
		header,
		"  " + DimStyle.Render("API Key:        ") + apiStatus,
		"  " + DimStyle.Render("Anthropic Key:  ") + anthropicStatus,
		"  " + DimStyle.Render("Base URL:       ") + cfg.BaseURL,
		keyringLine,
	}
	return strings.Join(lines, "\n")
}

func RenderConfigSet(key, value string) string {
	msg := BotMsgStyle.Render("nick: ") + "Set " +
		CommandStyle.Render(key) + " successfully."
	// Indicate storage location for secret keys.
	switch key {
	case "api_key", "anthropic_key", "minimax_key", "openrouter_key":
		if config.UseKeyring() {
			msg += DimStyle.Render(" (stored in OS keyring)")
		} else {
			msg += DimStyle.Render(" (stored in config file)")
		}
	}
	return msg
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
	header := SectionHeader("/config usage")
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
	header := SectionHeader("/mcp — manage trading integrations")
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
	lines := []string{SectionHeader("MCP Server Registry")}
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
		SectionHeader(e.DisplayName),
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
	header := SectionHeader("Saved Credentials")

	if len(store.Credentials) == 0 {
		return header + "\n" + EmptyState("No credentials saved.") +
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
	header := SectionHeader("Workflows")

	if len(store.Workflows) == 0 {
		return header + "\n" + EmptyState("No workflows yet.") +
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
	header := SectionHeader("Logs: " + w.Name)

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

// --- /help ---

func RenderHelp() string {
	header := SectionHeader("NickAI Commands")

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
	lines = append(lines, cmdLine("/alert list", "List active alerts"))
	lines = append(lines, cmdLine("/alert clear", "Clear all alerts"))

	lines = append(lines, sectionHeader("Portfolio"))
	lines = append(lines, cmdLine("/status", "Positions & cash balance"))
	lines = append(lines, cmdLine("/orders", "Recent orders & trades"))
	lines = append(lines, cmdLine("/snapshot", "Combined portfolio dashboard"))
	lines = append(lines, cmdLine("/market", "Full market overview (10 assets)"))
	lines = append(lines, cmdLine("/pnl", "Profit & loss summary"))
	lines = append(lines, cmdLine("/history", "Trade journal with all orders"))
	lines = append(lines, cmdLine("/chart BTC", "ASCII sparkline chart"))
	lines = append(lines, cmdLine("/analytics", "Portfolio analytics & stats"))

	lines = append(lines, sectionHeader("Analysis & AI"))
	lines = append(lines, cmdLine("/analyze BTC", "Technical analysis"))
	lines = append(lines, cmdLine("/analyze sentiment ETH", "Sentiment analysis"))
	lines = append(lines, cmdLine("/analyze whale BTC", "On-chain whale tracking"))
	lines = append(lines, cmdLine("/analyze defi", "Top DeFi yields"))
	lines = append(lines, cmdLine("/analyze presets", "List analysis presets"))
	lines = append(lines, cmdLine("/analyze run <preset> <sym>", "Run an analysis preset"))
	lines = append(lines, cmdLine("/consensus BTC", "Multi-LLM consensus (tier 1)"))
	lines = append(lines, cmdLine("/consensus all BTC", "All model tiers consensus"))
	lines = append(lines, cmdLine("/consensus budget BTC", "Free models only consensus"))
	lines = append(lines, cmdLine("/consensus models", "Show model tiers & costs"))
	lines = append(lines, cmdLine("/backtest presets", "List backtest strategies"))
	lines = append(lines, cmdLine("/backtest run rsi-reversal BTC", "Run a backtest preset"))
	lines = append(lines, cmdLine("/backtest activate <preset> <sym>", "Activate as live rule"))
	lines = append(lines, cmdLine("/polymarket scan", "Prediction market analysis"))

	lines = append(lines, sectionHeader("Strategy & Automation"))
	lines = append(lines, cmdLine("/strategy twap ETH buy $2000 4h", "Create TWAP strategy"))
	lines = append(lines, cmdLine("/strategy list", "List all strategies"))
	lines = append(lines, cmdLine("/strategy cancel <id>", "Cancel a running strategy"))
	lines = append(lines, cmdLine("/trigger add BTC < 60000 sell 0.5", "Conditional trigger"))
	lines = append(lines, cmdLine("/trigger list", "List active triggers"))
	lines = append(lines, cmdLine("/trigger remove <id>", "Remove a trigger"))
	lines = append(lines, cmdLine("/trigger clear", "Clear all triggers"))
	lines = append(lines, cmdLine("/auto list", "View automation rules"))
	lines = append(lines, cmdLine("/auto pause <id>", "Pause an automation rule"))
	lines = append(lines, cmdLine("/auto resume <id>", "Resume a paused rule"))
	lines = append(lines, cmdLine("/auto remove <id>", "Delete an automation rule"))
	lines = append(lines, cmdLine("/risk set max-order 5000", "Set max order size"))
	lines = append(lines, cmdLine("/risk set daily-loss 5", "Set daily loss limit %"))
	lines = append(lines, cmdLine("/risk show", "View current risk limits"))
	lines = append(lines, cmdLine("/risk clear", "Remove all risk limits"))
	lines = append(lines, cmdLine("/notify set desktop on", "Toggle desktop alerts"))
	lines = append(lines, cmdLine("/notify set webhook <url>", "Set webhook URL"))
	lines = append(lines, cmdLine("/notify test", "Send test notification"))

	lines = append(lines, sectionHeader("Agents & Workflows"))
	lines = append(lines, cmdLine("/agents", "List your trading agents"))
	lines = append(lines, cmdLine("/templates", "Browse marketplace templates"))
	lines = append(lines, cmdLine("/workflow", "Manage automation workflows"))
	lines = append(lines, cmdLine("/logs <workflow>", "Workflow execution logs"))
	lines = append(lines, cmdLine("/guide", "Interactive guide"))

	lines = append(lines, sectionHeader("Prediction Markets"))
	lines = append(lines, cmdLine("/markets", "Trending prediction markets"))
	lines = append(lines, cmdLine("/markets <query>", "Search markets"))
	lines = append(lines, cmdLine("/bet <market> <side> <amt>", "Place a prediction bet"))
	lines = append(lines, cmdLine("/positions", "Unified positions view"))

	lines = append(lines, sectionHeader("Exchange"))
	lines = append(lines, cmdLine("/connect <exchange>", "Connect an exchange"))
	lines = append(lines, cmdLine("/connect list", "Show connected exchanges"))
	lines = append(lines, cmdLine("/balances", "Unified balance view"))
	lines = append(lines, cmdLine("/funding", "Perpetual funding rates"))

	lines = append(lines, sectionHeader("Onchain"))
	lines = append(lines, cmdLine("/wallet balance <addr>", "Check wallet balances"))
	lines = append(lines, cmdLine("/swap SOL USDC 10", "Token swap (DEX)"))
	lines = append(lines, cmdLine("/gas", "Gas price estimates"))

	lines = append(lines, sectionHeader("Stocks"))
	lines = append(lines, cmdLine("/stock AAPL", "Stock analysis"))
	lines = append(lines, cmdLine("/screen <filters>", "Stock screener"))

	lines = append(lines, sectionHeader("Betting"))
	lines = append(lines, cmdLine("/odds Lakers vs Celtics", "Betting odds lookup"))
	lines = append(lines, cmdLine("/lines Super Bowl", "Line movement tracker"))

	lines = append(lines, sectionHeader("Memory"))
	lines = append(lines, cmdLine("/memory", "View saved memories"))
	lines = append(lines, cmdLine("/memory clear", "Clear all memories"))
	lines = append(lines, cmdLine("/memory remove <id>", "Remove a specific memory"))

	lines = append(lines, sectionHeader("Setup & Integrations"))
	lines = append(lines, cmdLine("/config init", "Create account & API key"))
	lines = append(lines, cmdLine("/config", "Manage settings & keys"))
	lines = append(lines, cmdLine("/mcp", "MCP server integrations"))
	lines = append(lines, cmdLine("/credential", "Exchange API keys"))
	lines = append(lines, cmdLine("/model <id>", "Switch AI model (9 models)"))
	lines = append(lines, cmdLine("/model <slug>", "Custom OpenRouter model"))
	lines = append(lines, cmdLine("/theme <name>", "Switch color theme"))
	lines = append(lines, cmdLine("/vibe <name>", "Switch AI personality"))

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

// RenderMemoryList renders saved memories as a styled card.
func RenderMemoryList(entries []memory.Entry) string {
	if len(entries) == 0 {
		return BotMsgStyle.Render("nick: ") + "No memories saved yet."
	}

	header := lipgloss.NewStyle().
		Foreground(ColorPrimary).Bold(true).
		Render("  Saved Memories")
	divider := "  " + Divider(50)

	var rows []string
	rows = append(rows, "", header, divider, "")

	for _, e := range entries {
		typeTag := lipgloss.NewStyle().Foreground(ColorSecondary).Render(fmt.Sprintf("[%s]", e.Type))
		date := DimStyle.Render(e.CreatedAt.Format("2006-01-02"))
		idHint := DimStyle.Render("(" + e.ID[:6] + ")")
		content := lipgloss.NewStyle().Foreground(ColorWhite).Render(e.Content)
		rows = append(rows, fmt.Sprintf("  %s %s %s %s", typeTag, content, date, idHint))
		if len(e.Tags) > 0 {
			rows = append(rows, "    "+DimStyle.Render("tags: "+strings.Join(e.Tags, ", ")))
		}
	}

	rows = append(rows, "", DimStyle.Render("  /memory clear to reset  •  /memory remove <id> to delete one"))
	return strings.Join(rows, "\n") + NextSteps("/memory clear")
}
