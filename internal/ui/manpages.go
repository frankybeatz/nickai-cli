package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	manHeader = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Bold(true).
			Underline(true)

	manBody = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC"))

	manExample = lipgloss.NewStyle().
			Foreground(ColorPrimary)
)

type manPage struct {
	name     string
	synopsis string
	desc     string
	examples []string
	seeAlso  []string
}

var manPages = map[string]manPage{
	"help": {
		name:     "help - display available commands",
		synopsis: "/help",
		desc: `Show a summary of all available NickAI terminal commands.
Each command is listed with a short description. For detailed
documentation on a specific command, use /man <command>.`,
		examples: []string{"/help"},
		seeAlso:  []string{"man"},
	},
	"status": {
		name:     "status - portfolio and platform status",
		synopsis: "/status",
		desc: `Display current portfolio status when an API key is configured.
Shows cash balance (available, reserved, total), total portfolio
value, and all open positions with quantities and values.

If no API key is set, displays mock platform health data.`,
		examples: []string{"/status"},
		seeAlso:  []string{"orders", "price", "config"},
	},
	"price": {
		name:     "price - live price quotes",
		synopsis: "/price <SYMBOL> [SYMBOL...]",
		desc: `Fetch live price quotes from the PaperNick API for one or more
cryptocurrency symbols. Symbols are automatically normalized
(e.g. BTC becomes BTCUSDT).

Requires a configured API key.`,
		examples: []string{
			"/price BTC",
			"/price BTC ETH SOL",
			"/price BTCUSDT ETHUSDT",
		},
		seeAlso: []string{"watch", "buy", "sell"},
	},
	"buy": {
		name:     "buy - place a buy order",
		synopsis: "/buy <SYMBOL> <QUANTITY> [limit <PRICE>]",
		desc: `Place a buy order on PaperNick. Defaults to market order.
Optionally specify "limit" followed by a price for limit orders.
Symbol is auto-normalized (BTC becomes BTCUSDT).

Requires a configured API key.`,
		examples: []string{
			"/buy BTC 0.1",
			"/buy ETH 2.5 limit 3800",
			"/buy SOL 100",
		},
		seeAlso: []string{"sell", "orders", "price"},
	},
	"sell": {
		name:     "sell - place a sell order",
		synopsis: "/sell <SYMBOL> <QUANTITY> [limit <PRICE>]",
		desc: `Place a sell order on PaperNick. Defaults to market order.
Optionally specify "limit" followed by a price for limit orders.
Symbol is auto-normalized (ETH becomes ETHUSDT).

Requires a configured API key.`,
		examples: []string{
			"/sell BTC 0.05",
			"/sell ETH 1.0 limit 4200",
		},
		seeAlso: []string{"buy", "orders", "price"},
	},
	"orders": {
		name:     "orders - list recent orders",
		synopsis: "/orders",
		desc: `Display recent orders including filled, pending, and cancelled
orders. Shows up to 10 most recent orders with symbol, side,
type, quantity, price, and status.

Requires a configured API key.`,
		examples: []string{"/orders"},
		seeAlso:  []string{"buy", "sell", "status"},
	},
	"agents": {
		name:     "agents - list trading agents",
		synopsis: "/agents",
		desc: `Display all configured trading agents with their current status,
strategy, PnL performance, and uptime. Currently shows mock data
when no API connection is established.`,
		examples: []string{"/agents"},
		seeAlso:  []string{"templates", "workflow"},
	},
	"templates": {
		name:     "templates - browse marketplace",
		synopsis: "/templates",
		desc: `Browse the NickAI template marketplace. Shows available strategy
templates with author, star ratings, descriptions, and tags.
Templates can be used to quickly deploy new trading agents.`,
		examples: []string{"/templates"},
		seeAlso:  []string{"agents", "workflow"},
	},
	"config": {
		name:     "config - manage configuration",
		synopsis: "/config <show|set|test> [args...]",
		desc: `Manage NickAI CLI configuration including API keys and base URL.

Subcommands:
  show                      Display current configuration
  set api_key <key>         Set PaperNick API key
  set anthropic_key <key>   Set Anthropic API key for AI chat
  set url <url>             Set custom base URL
  test                      Test API connection

Configuration is stored at ~/.nickai/config.json with restricted
file permissions (0600).`,
		examples: []string{
			"/config show",
			"/config set api_key pk_live_abc123",
			"/config set anthropic_key sk-ant-...",
			"/config test",
		},
		seeAlso: []string{"credential", "status"},
	},
	"clear": {
		name:     "clear - clear chat history",
		synopsis: "/clear",
		desc: `Clear all messages from the chat viewport and reset to the
welcome screen. Does not affect configuration or saved data.`,
		examples: []string{"/clear"},
		seeAlso:  []string{"quit"},
	},
	"quit": {
		name:     "quit - exit NickAI",
		synopsis: "/quit",
		desc: `Exit the NickAI terminal. Also available as /exit.
In vim NORMAL mode, press q or type :q to quit.`,
		examples: []string{"/quit", "/exit"},
		seeAlso:  []string{"clear"},
	},
	"credential": {
		name:     "credential - manage exchange API keys",
		synopsis: "/credential <list|add|remove> [args...]",
		desc: `Manage saved exchange API credentials for trading integrations.

Subcommands:
  list                                      Show all saved credentials
  add <name> <exchange> <key> <secret>      Save a new credential
  remove <name>                             Delete a saved credential

Alias: /cred

Supported exchanges: binance, coinbase, hyperliquid, alpaca,
polymarket.

Credentials are stored at ~/.nickai/credentials.json with
restricted file permissions (0600). API secrets are masked in
display output.`,
		examples: []string{
			"/credential list",
			"/credential add my-binance binance APIKEY123 APISECRET456",
			"/credential remove my-binance",
			"/cred list",
		},
		seeAlso: []string{"config", "workflow"},
	},
	"workflow": {
		name:     "workflow - manage automation workflows",
		synopsis: "/workflow <list|create|run|stop|show|remove|edit> [args...]",
		desc: `Manage multi-node automation workflows for trading strategies.

Subcommands:
  list                 Show all workflows with status and run count
  create <path.json>   Create workflow from a JSON definition file
  run <name>           Start a workflow (simulated execution)
  stop <name>          Stop a running workflow
  show <name>          Show workflow details with node list
  remove <name>        Delete a workflow
  edit <name>          Edit workflow (hint: use :e in COMMAND mode)

Alias: /wf

Node types: trigger, schedule, price_feed, data, analysis, llm,
condition, filter, trade, execution, notification, webhook.

Workflows are stored at ~/.nickai/workflows.json. See the
examples/ directory for sample workflow definitions.`,
		examples: []string{
			"/workflow list",
			"/workflow create examples/btc-momentum.json",
			"/workflow run btc-momentum",
			"/workflow show btc-momentum",
			"/workflow stop btc-momentum",
			"/wf list",
		},
		seeAlso: []string{"logs", "credential", "agents"},
	},
	"logs": {
		name:     "logs - workflow execution logs",
		synopsis: "/logs <workflow-name>",
		desc: `Display execution logs for a workflow. If the workflow is
currently running, shows simulated live node execution states.
If stopped, shows the summary from the last run.

Alias: /log`,
		examples: []string{
			"/logs btc-momentum",
			"/log polymarket-scanner",
		},
		seeAlso: []string{"workflow"},
	},
	"man": {
		name:     "man - manual pages",
		synopsis: "/man [command]",
		desc: `Display detailed unix-style manual page for a command. When
called without arguments, shows an index of all available
commands.

Alias: /manual

Each manual page includes NAME, SYNOPSIS, DESCRIPTION, EXAMPLES,
and SEE ALSO sections.`,
		examples: []string{
			"/man",
			"/man buy",
			"/man workflow",
			"/manual credential",
		},
		seeAlso: []string{"help"},
	},
	"snapshot": {
		name:     "snapshot - combined portfolio dashboard",
		synopsis: "/snapshot",
		desc: `Display a combined dashboard view with portfolio positions,
market prices, agent status, and recent trades in a single
screen. Pulls live data from the PaperNick API for portfolio
and market sections.

Agent data is currently mock data. Recent trades are limited
to the 5 most recent orders.

Alias: /snap`,
		examples: []string{"/snapshot", "/snap"},
		seeAlso:  []string{"status", "market", "pnl"},
	},
	"market": {
		name:     "market - full market overview",
		synopsis: "/market",
		desc: `Display prices for the top 10 cryptocurrencies in a formatted
table: BTC, ETH, SOL, DOGE, ADA, AVAX, LINK, DOT, MATIC, XRP.

Shows current price for each asset with a last-updated timestamp.

Requires a configured API key.`,
		examples: []string{"/market"},
		seeAlso:  []string{"price", "watch", "snapshot"},
	},
	"pnl": {
		name:     "pnl - profit and loss summary",
		synopsis: "/pnl",
		desc: `Calculate and display profit and loss from your trading history.
Shows starting balance ($100K), current portfolio value, total
P&L with percentage, a visual progress bar, and trade statistics
including win rate, best trade, and worst trade.

P&L is calculated from order history and current portfolio value.

Requires a configured API key.`,
		examples: []string{"/pnl"},
		seeAlso:  []string{"history", "status", "orders"},
	},
	"history": {
		name:     "history - trade journal",
		synopsis: "/history",
		desc: `Display a numbered trade journal showing all orders with
timestamps, side (BUY/SELL), asset, amount, and status.
Includes a count of total trades and trades made today.

Alias: /journal

Requires a configured API key.`,
		examples: []string{"/history", "/journal"},
		seeAlso:  []string{"orders", "pnl", "status"},
	},
	"watch": {
		name:     "watch - live price monitor",
		synopsis: "/watch <SYMBOL> [SYMBOL...]",
		desc: `Display a compact price monitoring dashboard for one or more
symbols. Shows current prices in a styled ticker box with a
LIVE indicator. Similar to /price but formatted as a monitoring
dashboard.

Requires a configured API key.`,
		examples: []string{
			"/watch BTC",
			"/watch BTC ETH SOL",
		},
		seeAlso: []string{"price", "status"},
	},
}

// RenderManPage renders a unix-style manual page for the given command.
func RenderManPage(command string) string {
	command = strings.TrimPrefix(command, "/")
	command = strings.ToLower(command)

	// Resolve aliases.
	switch command {
	case "cred":
		command = "credential"
	case "wf":
		command = "workflow"
	case "log":
		command = "logs"
	case "manual":
		command = "man"
	case "exit":
		command = "quit"
	case "snap":
		command = "snapshot"
	case "journal":
		command = "history"
	}

	page, ok := manPages[command]
	if !ok {
		return ErrorStyle.Render("  No manual entry for: ") + command +
			"\n" + DimStyle.Render("  Type /man for a list of all commands.")
	}

	var lines []string

	// NAME
	lines = append(lines, "")
	lines = append(lines, "  "+manHeader.Render("NAME"))
	lines = append(lines, "       "+manBody.Render(page.name))

	// SYNOPSIS
	lines = append(lines, "")
	lines = append(lines, "  "+manHeader.Render("SYNOPSIS"))
	lines = append(lines, "       "+CommandStyle.Render(page.synopsis))

	// DESCRIPTION
	lines = append(lines, "")
	lines = append(lines, "  "+manHeader.Render("DESCRIPTION"))
	for _, line := range strings.Split(page.desc, "\n") {
		lines = append(lines, "       "+manBody.Render(line))
	}

	// EXAMPLES
	if len(page.examples) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+manHeader.Render("EXAMPLES"))
		for _, ex := range page.examples {
			lines = append(lines, "       "+manExample.Render(ex))
		}
	}

	// SEE ALSO
	if len(page.seeAlso) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+manHeader.Render("SEE ALSO"))
		var refs []string
		for _, ref := range page.seeAlso {
			refs = append(refs, CommandStyle.Render("/"+ref))
		}
		lines = append(lines, "       "+strings.Join(refs, ", "))
	}

	lines = append(lines, "")

	title := SecondaryStyle.Render("  NICKAI(1)") +
		DimStyle.Render("                    NickAI Manual")
	return title + "\n" + strings.Join(lines, "\n")
}

// RenderManIndex renders an index of all available manual pages.
func RenderManIndex() string {
	title := SecondaryStyle.Render("  NickAI Manual Pages\n")

	type entry struct {
		cmd  string
		desc string
	}
	entries := []entry{
		{"help", "Display available commands"},
		{"status", "Portfolio and platform status"},
		{"price", "Live price quotes"},
		{"buy", "Place a buy order"},
		{"sell", "Place a sell order"},
		{"orders", "List recent orders"},
		{"agents", "List trading agents"},
		{"templates", "Browse marketplace"},
		{"snapshot", "Combined portfolio dashboard"},
		{"market", "Full market overview"},
		{"pnl", "Profit and loss summary"},
		{"history", "Trade journal"},
		{"config", "Manage configuration"},
		{"credential", "Manage exchange API keys"},
		{"workflow", "Manage automation workflows"},
		{"logs", "Workflow execution logs"},
		{"watch", "Live price monitor"},
		{"man", "Manual pages"},
		{"clear", "Clear chat history"},
		{"quit", "Exit NickAI"},
	}

	var lines []string
	lines = append(lines, title)
	for _, e := range entries {
		line := CommandStyle.Render(padRight("/man "+e.cmd, 28)) +
			DimStyle.Render(e.desc)
		lines = append(lines, "  "+line)
	}
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("  Type /man <command> for detailed documentation."))
	return strings.Join(lines, "\n")
}
