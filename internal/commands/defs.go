package commands

// CommandDef is the single source of truth for a slash command.
// allCommands (routing) and PaletteEntries (Ctrl+K) are both derived from this.
type CommandDef struct {
	Type        CommandType
	Primary     string   // e.g. "/buy"
	Aliases     []string // e.g. ["/b"]
	Description string   // short description for palette/help
	Category    string   // "Trading", "Analysis", "Tools", "Setup", "Multi-Vertical"
	// SubCommands lists child commands (e.g. "/mcp list", "/mcp add").
	// When present, the parent entry is omitted from the palette and
	// each subcommand gets its own palette entry.
	SubCommands []SubCommandDef
}

// SubCommandDef describes a subcommand under a parent command.
type SubCommandDef struct {
	Name        string // e.g. "list"
	Description string
}

// Registry is the canonical list of all commands.
var Registry = []CommandDef{
	// --- Navigation & Info ---
	{Type: TypeHelp, Primary: "/help", Description: "Show all commands", Category: "Setup"},
	{Type: TypeStatus, Primary: "/status", Description: "Portfolio & positions", Category: "Trading"},
	{Type: TypeOrders, Primary: "/orders", Description: "Recent orders", Category: "Trading"},
	{Type: TypePrice, Primary: "/price", Aliases: []string{"/p"}, Description: "Live price quotes", Category: "Trading"},
	{Type: TypeWatch, Primary: "/watch", Description: "Live price dashboard", Category: "Trading"},
	{Type: TypeSnapshot, Primary: "/snapshot", Aliases: []string{"/snap"}, Description: "Combined dashboard", Category: "Trading"},
	{Type: TypeMarket, Primary: "/market", Description: "Full market overview", Category: "Trading"},
	{Type: TypeDashboard, Primary: "/dashboard", Aliases: []string{"/dash"}, Description: "Scrolling dashboard", Category: "Trading"},
	{Type: TypePnl, Primary: "/pnl", Description: "Profit & loss summary", Category: "Trading"},
	{Type: TypeHistory, Primary: "/history", Aliases: []string{"/journal"}, Description: "Trade journal", Category: "Trading"},

	// --- Trading ---
	{Type: TypeBuy, Primary: "/buy", Aliases: []string{"/b"}, Description: "Market buy order", Category: "Trading"},
	{Type: TypeSell, Primary: "/sell", Aliases: []string{"/s"}, Description: "Limit/market sell", Category: "Trading"},
	{Type: TypeAlert, Primary: "/alert", Description: "Set price alerts", Category: "Trading"},
	{Type: TypeChart, Primary: "/chart", Description: "ASCII sparkline chart", Category: "Trading"},
	{Type: TypeFunding, Primary: "/funding", Description: "Perpetual funding rates", Category: "Trading"},

	// --- Analysis & AI ---
	{Type: TypeAnalyze, Primary: "/analyze", Description: "Technical analysis for a symbol", Category: "Analysis", SubCommands: []SubCommandDef{
		{Name: "presets", Description: "Browse analysis presets"},
		{Name: "run", Description: "Run an analysis preset"},
	}},
	{Type: TypeBacktest, Primary: "/backtest", Aliases: []string{"/bt"}, Description: "Backtest a trading strategy", Category: "Analysis", SubCommands: []SubCommandDef{
		{Name: "presets", Description: "Browse preset strategies"},
		{Name: "run", Description: "Run a preset backtest"},
		{Name: "analyze", Description: "AI analysis of last backtest"},
	}},
	{Type: TypeConsensus, Primary: "/consensus", Aliases: []string{"/con"}, Description: "Multi-LLM trading consensus", Category: "Analysis", SubCommands: []SubCommandDef{
		{Name: "models", Description: "Available consensus models"},
	}},
	{Type: TypeAnalytics, Primary: "/analytics", Description: "Portfolio analytics dashboard", Category: "Analysis"},
	{Type: TypePolymarket, Primary: "/polymarket", Aliases: []string{"/pm"}, Description: "Prediction market analysis", Category: "Analysis", SubCommands: []SubCommandDef{
		{Name: "scan", Description: "Scan top events"},
		{Name: "analyze", Description: "Deep dive on an event"},
	}},

	// --- Automation ---
	{Type: TypeTrigger, Primary: "/trigger", Aliases: []string{"/trig"}, Description: "Conditional trade triggers", Category: "Tools", SubCommands: []SubCommandDef{
		{Name: "list", Description: "View active triggers"},
		{Name: "add", Description: "Create conditional trade trigger"},
		{Name: "clear", Description: "Remove all triggers"},
	}},
	{Type: TypeRisk, Primary: "/risk", Description: "Risk guardrails", Category: "Tools", SubCommands: []SubCommandDef{
		{Name: "show", Description: "View risk guardrails"},
		{Name: "set", Description: "Set risk limits"},
		{Name: "clear", Description: "Remove all risk limits"},
	}},
	{Type: TypeStrategy, Primary: "/strategy", Aliases: []string{"/strat"}, Description: "TWAP strategies", Category: "Tools", SubCommands: []SubCommandDef{
		{Name: "list", Description: "View TWAP strategies"},
		{Name: "twap", Description: "Create TWAP strategy"},
		{Name: "cancel", Description: "Cancel a strategy"},
	}},
	{Type: TypeNotify, Primary: "/notify", Description: "Notification settings", Category: "Tools", SubCommands: []SubCommandDef{
		{Name: "show", Description: "Notification settings"},
		{Name: "set", Description: "Configure notifications"},
		{Name: "test", Description: "Send test notification"},
	}},
	{Type: TypeAuto, Primary: "/auto", Aliases: []string{"/automation"}, Description: "Automation rules", Category: "Tools", SubCommands: []SubCommandDef{
		{Name: "list", Description: "View automation rules"},
		{Name: "pause", Description: "Pause an automation rule"},
		{Name: "remove", Description: "Remove an automation rule"},
	}},

	// --- Memory & Guidance ---
	{Type: TypeGuide, Primary: "/guide", Description: "Interactive walkthrough", Category: "Tools", SubCommands: []SubCommandDef{
		{Name: "trading", Description: "Trading basics"},
		{Name: "backtest", Description: "Backtesting guide"},
		{Name: "mcp", Description: "MCP integrations guide"},
	}},
	{Type: TypeMemory, Primary: "/memory", Aliases: []string{"/mem"}, Description: "View saved AI memories", Category: "Tools", SubCommands: []SubCommandDef{
		{Name: "clear", Description: "Clear all memories"},
	}},

	// --- Setup ---
	{Type: TypeModel, Primary: "/model", Aliases: []string{"/models"}, Description: "Switch AI model", Category: "Setup"},
	{Type: TypeTheme, Primary: "/theme", Description: "Switch color theme", Category: "Setup"},
	{Type: TypeVibe, Primary: "/vibe", Description: "Switch personality vibe", Category: "Setup"},
	{Type: TypeConfig, Primary: "/config", Description: "Manage settings", Category: "Setup", SubCommands: []SubCommandDef{
		{Name: "init", Description: "Create account & API key"},
	}},
	{Type: TypeMCP, Primary: "/mcp", Description: "MCP server management", Category: "Setup", SubCommands: []SubCommandDef{
		{Name: "list", Description: "Connected MCP servers & tools"},
		{Name: "search", Description: "Browse MCP server directory"},
		{Name: "add", Description: "Install an MCP server"},
		{Name: "info", Description: "Details on an MCP server"},
		{Name: "remove", Description: "Disconnect an MCP server"},
		{Name: "quick", Description: "Install all free MCP servers"},
	}},
	{Type: TypeCredential, Primary: "/credential", Aliases: []string{"/cred"}, Description: "Manage API keys", Category: "Setup"},
	{Type: TypeWorkflow, Primary: "/workflow", Aliases: []string{"/wf"}, Description: "Manage workflows", Category: "Setup"},
	{Type: TypeLogs, Primary: "/logs", Aliases: []string{"/log"}, Description: "Workflow logs", Category: "Setup"},
	{Type: TypeMan, Primary: "/man", Aliases: []string{"/manual"}, Description: "Manual pages", Category: "Setup"},
	{Type: TypeAgents, Primary: "/agents", Description: "List trading agents", Category: "Setup"},
	{Type: TypeTemplates, Primary: "/templates", Description: "Browse marketplace", Category: "Setup"},
	{Type: TypeClear, Primary: "/clear", Description: "Clear chat", Category: "Setup"},
	{Type: TypeQuit, Primary: "/quit", Aliases: []string{"/exit"}, Description: "Exit NickAI", Category: "Setup"},

	// --- Multi-Vertical ---
	{Type: TypeConnect, Primary: "/connect", Description: "Connect an exchange", Category: "Multi-Vertical", SubCommands: []SubCommandDef{
		{Name: "list", Description: "Show connected exchanges"},
	}},
	{Type: TypeBalances, Primary: "/balances", Aliases: []string{"/bal"}, Description: "Unified balance view", Category: "Multi-Vertical"},
	{Type: TypePositions, Primary: "/positions", Aliases: []string{"/pos"}, Description: "Open positions across exchanges", Category: "Multi-Vertical"},
	{Type: TypeMarkets, Primary: "/markets", Description: "Trending prediction markets", Category: "Multi-Vertical"},
	{Type: TypeBet, Primary: "/bet", Description: "Place a prediction market bet", Category: "Multi-Vertical"},
	{Type: TypeWallet, Primary: "/wallet", Description: "Wallet management", Category: "Multi-Vertical", SubCommands: []SubCommandDef{
		{Name: "balance", Description: "Check wallet balances"},
	}},
	{Type: TypeSwap, Primary: "/swap", Description: "Token swap (DEX)", Category: "Multi-Vertical"},
	{Type: TypeGas, Primary: "/gas", Description: "Gas price estimates", Category: "Multi-Vertical"},
	{Type: TypeStock, Primary: "/stock", Description: "Stock analysis", Category: "Multi-Vertical"},
	{Type: TypeScreen, Primary: "/screen", Description: "Stock screener", Category: "Multi-Vertical"},
	{Type: TypeOdds, Primary: "/odds", Description: "Betting odds lookup", Category: "Multi-Vertical"},
	{Type: TypeLines, Primary: "/lines", Description: "Line movement tracker", Category: "Multi-Vertical"},

	// --- Export ---
	{Type: TypeExport, Primary: "/export", Description: "Export data to CSV", Category: "Tools", SubCommands: []SubCommandDef{
		{Name: "trades", Description: "Export trade history"},
		{Name: "portfolio", Description: "Export portfolio positions"},
		{Name: "backtest", Description: "Export last backtest results"},
	}},

	// --- Plugins ---
	{Type: TypePlugin, Primary: "/plugin", Aliases: []string{"/plugins"}, Description: "Manage MCP server plugins", Category: "Setup", SubCommands: []SubCommandDef{
		{Name: "list", Description: "List installed and available plugins"},
		{Name: "install", Description: "Install an MCP server plugin"},
		{Name: "remove", Description: "Remove an installed plugin"},
		{Name: "search", Description: "Search available plugins"},
	}},
	{Type: TypeNode, Primary: "/node", Description: "Connect to a Nick Node for always-on execution", Category: "Setup", SubCommands: []SubCommandDef{
		{Name: "connect", Description: "Connect to a running node"},
		{Name: "status", Description: "Show node status"},
		{Name: "deploy", Description: "Deploy a strategy to the node"},
		{Name: "strategies", Description: "List running strategies on node"},
		{Name: "stop", Description: "Stop a strategy on the node"},
		{Name: "disconnect", Description: "Disconnect from node"},
	}},
}

// BuildCommandMap generates the allCommands routing map from the Registry.
func BuildCommandMap() map[string]CommandType {
	m := make(map[string]CommandType, len(Registry)*2)
	for _, def := range Registry {
		m[def.Primary] = def.Type
		for _, alias := range def.Aliases {
			m[alias] = def.Type
		}
	}
	return m
}

// PaletteEntries generates the "command|description" list for the Ctrl+K palette.
func PaletteEntries() []string {
	var entries []string
	for _, def := range Registry {
		if len(def.SubCommands) > 0 {
			// Show parent with its own description first (if it has one).
			entries = append(entries, def.Primary+"|"+def.Description)
			for _, sub := range def.SubCommands {
				entries = append(entries, def.Primary+" "+sub.Name+"|"+sub.Description)
			}
		} else {
			entries = append(entries, def.Primary+"|"+def.Description)
		}
	}
	return entries
}
