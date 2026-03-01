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
	"chart": {
		name:     "chart - ASCII sparkline chart",
		synopsis: "/chart <SYMBOL>",
		desc: `Display a sparkline chart for a cryptocurrency symbol. Fetches
the current price and generates a simulated 50-point price
history to render a visual sparkline using block characters.

Shows current price, high/low range, and trend direction
(green for up, red for down).

Requires a configured API key.`,
		examples: []string{"/chart BTC", "/chart ETH", "/chart SOL"},
		seeAlso:  []string{"price", "watch", "market"},
	},
	"alert": {
		name:     "alert - set price alerts",
		synopsis: "/alert <SYMBOL> > <PRICE>  or  /alert <SYMBOL> < <PRICE>",
		desc: `Set a background price alert that triggers when a symbol
crosses a target price. Alerts are checked every 30 seconds.

When triggered, a styled notification appears in chat with
a terminal bell.

Subcommands:
  list       Show all active alerts
  clear      Remove all alerts

Operators: > (above), < (below).`,
		examples: []string{
			"/alert BTC > 100000",
			"/alert ETH < 2000",
			"/alert list",
			"/alert clear",
		},
		seeAlso: []string{"price", "watch", "chart"},
	},
	"model": {
		name:     "model - switch AI model",
		synopsis: "/model [id]",
		desc: `Switch the AI language model used for chat. When called without
arguments, shows all available models with the current selection.

Available models:
  claude-sonnet    Claude Sonnet 4 (Anthropic, requires API key)
  claude-haiku     Claude Haiku 4.5 (Anthropic, requires API key)
  minimax          MiniMax abab6.5s (free tier, requires API key)

Model preference is saved to config and persists across sessions.
Switching models clears conversation history.

Alias: /models`,
		examples: []string{
			"/model",
			"/model claude-haiku",
			"/model minimax",
		},
		seeAlso: []string{"config"},
	},
	"theme": {
		name:     "theme - switch color theme",
		synopsis: "/theme [name]",
		desc: `Switch the terminal color scheme. When called without arguments,
shows all available themes with the current selection marked.

Available themes:
  default     NickAI brand colors (green/purple)
  cyberpunk   Neon magenta and cyan
  bloomberg   Orange terminal style
  minimal     Muted grayscale
  matrix      Green-on-black hacker aesthetic

Theme preference is saved to config and persists across sessions.`,
		examples: []string{
			"/theme",
			"/theme cyberpunk",
			"/theme bloomberg",
			"/theme default",
		},
		seeAlso: []string{"config"},
	},
	"mcp": {
		name:     "mcp - manage MCP server connections",
		synopsis: "/mcp <subcommand> [args]",
		desc: `Manage Model Context Protocol (MCP) servers. MCP lets NickAI
connect to external tools — trading APIs, blockchain data,
DeFi protocols, and more — as plug-and-play extensions.

Subcommands:
  list              Show connected servers and their tools
  search [query]    Browse the curated server directory
  info <name>       Detailed info on a specific server
  add <name>        Install a server from the registry
  remove <name>     Disconnect a server

Servers are stored in ~/.nickai/mcp.json and loaded on startup.
Each server runs as a subprocess communicating over stdio.`,
		examples: []string{
			"/mcp list",
			"/mcp search trading",
			"/mcp search defi",
			"/mcp info ccxt",
			"/mcp add ccxt",
			"/mcp remove ccxt",
		},
		seeAlso: []string{"config", "credential"},
	},
	"trigger": {
		name:     "trigger - conditional trading rules",
		synopsis: "/trigger <subcommand> [args]",
		desc: `Create conditional trading rules that fire when price conditions
are met. Triggers persist across restarts and are checked every
30 seconds alongside price alerts.

When a trigger fires, you get a confirmation prompt before the
trade executes — you always stay in control.

Subcommands:
  list                          Show active triggers
  add <SYM> < or > <PRICE> <buy|sell> <QTY>  Create a trigger
  remove <id>                   Remove by ID prefix
  clear                         Remove all triggers`,
		examples: []string{
			"/trigger list",
			"/trigger add BTC < 60000 sell 0.5",
			"/trigger add ETH > 4000 buy 2 market",
			"/trigger remove a1b2c3",
			"/trigger clear",
		},
		seeAlso: []string{"alert", "buy", "sell"},
	},
	"risk": {
		name:     "risk - portfolio risk guardrails",
		synopsis: "/risk <show|set|clear> [args...]",
		desc: `Manage risk guardrails that protect your portfolio from
oversized trades and excessive losses. Risk limits are checked
before every trade — AI, manual, trigger, and strategy trades.

Subcommands:
  show                              Display current risk limits
  set max-order <amount>            Max single order value ($)
  set max-position <pct>            Max position % of portfolio
  set daily-loss <pct>              Max daily loss % (from $100K)
  clear                             Remove all risk limits

When a risk limit blocks a trade, the AI receives an error
explaining why, so it can adjust its approach.

Limits are stored at ~/.nickai/risk.json and persist across
sessions.`,
		examples: []string{
			"/risk show",
			"/risk set max-order 5000",
			"/risk set max-position 10",
			"/risk set daily-loss 5",
			"/risk clear",
		},
		seeAlso: []string{"buy", "sell", "strategy", "trigger"},
	},
	"strategy": {
		name:     "strategy - TWAP time-weighted strategies",
		synopsis: "/strategy <twap|list|cancel> [args...]",
		desc: `Create and manage TWAP (Time-Weighted Average Price) strategies
that split large orders into smaller slices executed at regular
intervals.

Subcommands:
  twap <SYM> <buy|sell> <$VALUE> <DURATION>   Create TWAP strategy
  list                                         Show all strategies
  cancel <id>                                  Cancel by ID prefix

Duration format: "4h", "1h", "30m", etc. Slices are calculated
automatically (minimum 4 slices, one every 15 minutes minimum).

Each slice triggers a confirmation prompt before execution.
Risk guardrails apply to each slice individually.

Alias: /strat

Strategies persist at ~/.nickai/strategies.json.`,
		examples: []string{
			"/strategy twap ETH buy $2000 4h",
			"/strategy twap SOL sell $500 1h",
			"/strategy list",
			"/strategy cancel a1b2",
			"/strat list",
		},
		seeAlso: []string{"risk", "buy", "sell", "trigger"},
	},
	"notify": {
		name:     "notify - desktop & webhook notifications",
		synopsis: "/notify <show|set|clear|test> [args...]",
		desc: `Manage notification channels for alerts, trades, and strategy events.
Notifications fire when alerts trigger, trades execute, and TWAP
slices complete — even when you're not watching the terminal.

Subcommands:
  show                           Display current settings
  set desktop on|off             Toggle macOS desktop notifications
  set sound on|off               Toggle sound with notifications
  set webhook <url>              Set a webhook URL for POST notifications
  clear                          Reset all notification settings
  test                           Send a test notification

Desktop notifications use macOS native display notification.
Webhook sends JSON POST with {title, body, timestamp}.

Settings are stored at ~/.nickai/notify.json.`,
		examples: []string{
			"/notify show",
			"/notify set desktop on",
			"/notify set webhook https://hooks.example.com/notify",
			"/notify test",
			"/notify clear",
		},
		seeAlso: []string{"alert", "strategy", "auto"},
	},
	"analytics": {
		name:     "analytics - portfolio analytics dashboard",
		synopsis: "/analytics",
		desc: `Display advanced portfolio analytics including Sharpe ratio,
max drawdown, win rate, profit factor, allocation breakdown,
and trade statistics.

Metrics are calculated from your trade journal and current
portfolio positions. Requires at least one trade in the journal.

The AI can also access analytics via the get_analytics tool —
try asking "how am I performing?"`,
		examples: []string{"/analytics"},
		seeAlso:  []string{"pnl", "history", "status"},
	},
	"analyze": {
		name:     "analyze - AI market analysis with technical indicators",
		synopsis: "/analyze <SYMBOL>",
		desc: `Run technical analysis on a cryptocurrency symbol. Computes
RSI (14), MACD (12/26/9), Bollinger Bands (20), SMA 20/50,
trend direction, and fetches the Fear & Greed Index.

Each indicator shows a signal (bullish/bearish/neutral) and
a summary recommendation is generated from the combined signals.

The AI can also run analysis via the analyze_market tool —
try asking "should I buy ETH?"

Price history uses synthetic data for paper trading. Connect
a TradingView MCP server for real historical data.`,
		examples: []string{
			"/analyze BTC",
			"/analyze ETH",
			"/analyze SOL",
		},
		seeAlso: []string{"price", "chart", "analytics"},
	},
	"auto": {
		name:     "auto - natural language automation rules",
		synopsis: "/auto <list|pause|resume|remove> [args...]",
		desc: `Manage automation rules created by the AI. Rules can be:

  schedule   - Fire on a time schedule (daily, hourly, every 4h)
  condition  - Fire when a price condition is met (BTC > 100000)
  portfolio  - Fire on portfolio metrics (drawdown > 5%)

Rules are created by asking the AI in natural language:
  "buy $100 of BTC every day"
  "sell all ETH if it goes above 5000"
  "alert me if portfolio drops 5%"

Each fire requires user confirmation before executing.
Risk guardrails apply to all automation trades.

Subcommands:
  list                   Show all automation rules
  pause <id>             Pause a rule (prefix match)
  resume <id>            Resume a paused rule
  remove <id>            Delete a rule

Alias: /automation

Rules are stored at ~/.nickai/automations.json.`,
		examples: []string{
			"/auto list",
			"/auto pause a1b2c3",
			"/auto resume a1b2c3",
			"/auto remove a1b2c3",
		},
		seeAlso: []string{"trigger", "strategy", "risk", "notify"},
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
	"connect": {
		name:     "connect - connect an exchange",
		synopsis: "/connect <exchange>",
		desc: `Connect to a centralized exchange via MCP servers. Maps exchange
names to the appropriate MCP server and provides setup hints.

Use /connect list to see currently connected exchanges.

Supported exchanges: binance, coinbase, hyperliquid, kraken, bybit, alpaca.`,
		examples: []string{
			"/connect binance",
			"/connect list",
			"/connect alpaca",
		},
		seeAlso: []string{"mcp", "balances", "credential"},
	},
	"balances": {
		name:     "balances - unified balance view",
		synopsis: "/balances",
		desc: `Show a unified view of balances across paper trading and any
connected exchanges. Displays cash, available cash, and total
portfolio value for each connected account.`,
		examples: []string{"/balances", "/bal"},
		seeAlso: []string{"status", "positions", "connect"},
	},
	"positions": {
		name:     "positions - unified positions view",
		synopsis: "/positions",
		desc: `Show all open positions across paper trading and connected
exchanges. Displays symbol, quantity, and value for each position.`,
		examples: []string{"/positions", "/pos"},
		seeAlso: []string{"status", "balances", "orders"},
	},
	"markets": {
		name:     "markets - prediction market search",
		synopsis: "/markets [query]",
		desc: `Search or browse prediction markets. Without arguments, shows
trending markets with highest volume. With a query, searches
for matching markets.

Requires the polymarket MCP server for live data.`,
		examples: []string{
			"/markets",
			"/markets US election",
			"/markets crypto ETF",
		},
		seeAlso: []string{"bet", "polymarket"},
	},
	"bet": {
		name:     "bet - place a prediction market bet",
		synopsis: "/bet <market> <yes|no> <amount>",
		desc: `Place a bet on a prediction market event. Requires a market
identifier, the side (yes or no), and a dollar amount.

Routes through the AI for execution via polymarket MCP tools.
Requires trade confirmation before execution.`,
		examples: []string{
			"/bet \"Trump wins\" yes 50",
			"/bet \"BTC 100k\" no 25",
		},
		seeAlso: []string{"markets", "polymarket", "positions"},
	},
	"wallet": {
		name:     "wallet - onchain wallet operations",
		synopsis: "/wallet <subcommand> [args]",
		desc: `Interact with onchain wallets. Check token balances for any
address across supported chains.

Requires onchain/web3 MCP server for live data.`,
		examples: []string{
			"/wallet balance 0xabc...",
		},
		seeAlso: []string{"swap", "gas", "connect"},
	},
	"swap": {
		name:     "swap - token swap on DEX",
		synopsis: "/swap <from> <to> <amount>",
		desc: `Swap tokens on a decentralized exchange. Routes to Jupiter
(Solana) or LiFi (cross-chain) MCP servers for execution.

Requires trade confirmation before execution.`,
		examples: []string{
			"/swap SOL USDC 10",
			"/swap ETH USDT 0.5",
		},
		seeAlso: []string{"wallet", "gas", "buy"},
	},
	"gas": {
		name:     "gas - blockchain gas prices",
		synopsis: "/gas [chain]",
		desc: `Fetch current gas price estimates for a blockchain. Shows fast,
standard, and slow estimates. Defaults to Ethereum if no chain
is specified.

Uses onchain MCP tools when available.`,
		examples: []string{
			"/gas",
			"/gas ethereum",
			"/gas solana",
		},
		seeAlso: []string{"wallet", "swap"},
	},
	"stock": {
		name:     "stock - stock analysis",
		synopsis: "/stock <ticker>",
		desc: `Analyze a stock by ticker symbol. Shows current price, key
fundamentals (P/E ratio, market cap, revenue), and recent news.

Uses Alpaca MCP server when connected for live data, otherwise
relies on AI knowledge.`,
		examples: []string{
			"/stock AAPL",
			"/stock TSLA",
			"/stock NVDA",
		},
		seeAlso: []string{"screen", "connect"},
	},
	"screen": {
		name:     "screen - stock screener",
		synopsis: "/screen <filters>",
		desc: `Screen stocks matching natural language criteria. The AI interprets
your filters and returns matching stocks with ticker, price,
and why they match.`,
		examples: []string{
			"/screen high dividend tech stocks under $50",
			"/screen small cap biotech",
			"/screen low P/E ratio energy stocks",
		},
		seeAlso: []string{"stock", "analyze"},
	},
	"odds": {
		name:     "odds - betting odds lookup",
		synopsis: "/odds <event>",
		desc: `Find current betting odds for a sporting event or matchup. Shows
moneyline, spread, and over/under from major sportsbooks.

Uses brave-search MCP or web tools for live odds data.`,
		examples: []string{
			"/odds Lakers vs Celtics",
			"/odds Super Bowl winner",
			"/odds UFC 300 main event",
		},
		seeAlso: []string{"lines"},
	},
	"lines": {
		name:     "lines - betting line movement",
		synopsis: "/lines <event>",
		desc: `Show line movement and betting line history for a sporting event.
Highlights significant shifts in odds that may indicate sharp
money movement.

Uses brave-search MCP or web tools for historical line data.`,
		examples: []string{
			"/lines Super Bowl",
			"/lines Lakers vs Celtics",
		},
		seeAlso: []string{"odds"},
	},
	"funding": {
		name:     "funding - perpetual funding rates",
		synopsis: "/funding [symbol]",
		desc: `Show current funding rates for major perpetual futures contracts.
Includes annualized rates and direction (longs paying shorts or
vice versa). Useful for Hyperliquid and other perp DEX traders.`,
		examples: []string{
			"/funding",
			"/funding BTC ETH",
		},
		seeAlso: []string{"connect", "balances"},
	},
	"memory": {
		name:     "memory - AI memory management",
		synopsis: "/memory [clear|remove <id>]",
		desc: `View, clear, or manage NickAI's persistent memory. Memories are
insights, preferences, and context that Nick remembers across
sessions to provide more personalized responses.`,
		examples: []string{
			"/memory",
			"/memory clear",
			"/memory remove abc123",
		},
		seeAlso: []string{"config"},
	},
	"consensus": {
		name:     "consensus - multi-LLM trading consensus",
		synopsis: "/consensus [all|budget] <symbol>",
		desc: `Query multiple frontier LLMs in parallel for a BUY/SELL/HOLD
verdict on any asset. Models vote independently and the consensus
is determined by agreement threshold (67%).

Three tiers available:
  Tier 1 (default): 4 frontier models
  Tier 2 (all):     All 10 models across all tiers
  Tier 3 (budget):  Free/near-free models only

Requires an OpenRouter API key.`,
		examples: []string{
			"/consensus BTC",
			"/consensus all ETH",
			"/consensus budget SOL",
			"/consensus models",
		},
		seeAlso: []string{"analyze", "price"},
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
	case "models":
		command = "model"
	case "trig":
		command = "trigger"
	case "strat":
		command = "strategy"
	case "automation":
		command = "auto"
	case "bal":
		command = "balances"
	case "pos":
		command = "positions"
	case "mem":
		command = "memory"
	case "con":
		command = "consensus"
	case "bt":
		command = "backtest"
	case "pm":
		command = "polymarket"
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
		{"chart", "ASCII sparkline chart"},
		{"alert", "Set price alerts"},
		{"trigger", "Conditional trading rules"},
		{"risk", "Portfolio risk guardrails"},
		{"strategy", "TWAP time-weighted strategies"},
		{"notify", "Desktop & webhook notifications"},
		{"analytics", "Portfolio analytics dashboard"},
		{"analyze", "AI market analysis with indicators"},
		{"auto", "Natural language automation rules"},
		{"model", "Switch AI model"},
		{"theme", "Switch color theme"},
		{"config", "Manage configuration"},
		{"mcp", "MCP server integrations"},
		{"credential", "Manage exchange API keys"},
		{"workflow", "Manage automation workflows"},
		{"logs", "Workflow execution logs"},
		{"watch", "Live price monitor"},
		{"memory", "AI memory management"},
		{"consensus", "Multi-LLM trading consensus"},
		{"connect", "Connect an exchange"},
		{"balances", "Unified balance view"},
		{"positions", "Open positions view"},
		{"markets", "Prediction market search"},
		{"bet", "Place a prediction bet"},
		{"wallet", "Onchain wallet operations"},
		{"swap", "Token swap on DEX"},
		{"gas", "Blockchain gas prices"},
		{"stock", "Stock analysis"},
		{"screen", "Stock screener"},
		{"odds", "Betting odds lookup"},
		{"lines", "Betting line movement"},
		{"funding", "Perpetual funding rates"},
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
