package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// exchangeMap maps common exchange names to MCP server names.
var exchangeMap = map[string]string{
	"binance":     "binance",
	"coinbase":    "ccxt",
	"hyperliquid": "ccxt",
	"kraken":      "ccxt",
	"bybit":       "ccxt",
	"alpaca":      "alpaca",
}

func (m *Model) handleConnect(args []string) string {
	if len(args) == 0 {
		return RenderConnectHelp()
	}

	sub := strings.ToLower(args[0])
	if sub == "list" {
		if m.mcpManager == nil || m.mcpManager.ConnectionCount() == 0 {
			return BotMsgStyle.Render("nick: ") + "No exchanges connected." + "\n" +
				DimStyle.Render("  Run /connect to see available exchanges.")
		}
		var rows []string
		rows = append(rows, BotMsgStyle.Render("nick: ")+"Connected exchanges:")
		for _, c := range m.mcpManager.Connections() {
			rows = append(rows, "  "+lipgloss.NewStyle().Foreground(ColorPrimary).Render("●")+" "+c.Name+
				DimStyle.Render(fmt.Sprintf(" (%d tools)", len(c.Tools))))
		}
		return strings.Join(rows, "\n")
	}

	// Map exchange name to MCP server.
	serverName, ok := exchangeMap[sub]
	if !ok {
		return ErrorStyle.Render("  Unknown exchange: ") + sub + "\n" +
			DimStyle.Render("  Available: binance, coinbase, hyperliquid, kraken, bybit, alpaca")
	}
	return BotMsgStyle.Render("nick: ") + "To connect " + sub + ", run:\n" +
		"  " + CommandStyle.Render("/mcp add "+serverName) + "\n" +
		DimStyle.Render("  This installs the MCP server for "+sub+".")
}

func (m *Model) handleMarkets(args []string) (string, tea.Cmd) {
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}

	prompt := "Show trending prediction markets with highest volume. Use available polymarket or prediction market tools."
	if len(args) > 0 {
		prompt = "Search prediction markets for: " + strings.Join(args, " ")
	}

	return m.streamToAI(prompt, "Searching markets...", "markets")
}

func (m *Model) handleBet(args []string) (string, tea.Cmd) {
	if len(args) < 3 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/bet <market> <yes|no> <amount>") + "\n" +
			DimStyle.Render("  Example: /bet \"Trump wins\" yes 50"), nil
	}
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	prompt := fmt.Sprintf("Place a prediction market bet: market=%s, side=%s, amount=$%s. Use the polymarket tools to execute.",
		args[0], args[1], args[2])
	return m.streamToAI(prompt, "Placing bet...", "bet")
}

func (m *Model) handleWallet(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return RenderWalletHelp(), nil
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "balance", "bal":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/wallet balance <address>"), nil
		}
		if msg := m.requireAgent(); msg != "" {
			return msg, nil
		}
		prompt := "Check the wallet balance for address: " + args[1] + ". Use onchain/web3 MCP tools if available."
		return m.streamToAI(prompt, "Checking wallet...", "wallet")

	default:
		if msg := m.requireAgent(); msg != "" {
			return msg, nil
		}
		prompt := "Wallet command: " + strings.Join(args, " ")
		return m.streamToAI(prompt, "Processing wallet request...", "wallet")
	}
}

func (m *Model) handleSwap(args []string) (string, tea.Cmd) {
	if len(args) < 3 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/swap <from> <to> <amount>") + "\n" +
			DimStyle.Render("  Example: /swap SOL USDC 10"), nil
	}
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}

	from := strings.ToUpper(args[0])
	to := strings.ToUpper(args[1])
	amount := args[2]
	prompt := fmt.Sprintf("Swap %s %s to %s using Jupiter (Solana) or LiFi (cross-chain) MCP servers. Confirm before executing.", amount, from, to)
	return m.streamToAI(prompt, fmt.Sprintf("Swapping %s %s → %s...", amount, from, to), "swap")
}

func (m *Model) handleGas(args []string) (string, tea.Cmd) {
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	chain := "ethereum"
	if len(args) > 0 {
		chain = strings.ToLower(args[0])
	}
	prompt := fmt.Sprintf("Fetch current gas prices for %s. Show fast, standard, and slow estimates. Use onchain MCP tools if available.", chain)
	return m.streamToAI(prompt, "Fetching gas prices...", "gas")
}

func (m *Model) handleStock(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/stock <ticker>") + "\n" +
			DimStyle.Render("  Example: /stock AAPL"), nil
	}
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	ticker := strings.ToUpper(args[0])
	prompt := fmt.Sprintf("Analyze stock %s — current price, key fundamentals (P/E, market cap, revenue), and recent news. Use Alpaca MCP if connected, otherwise use your knowledge.", ticker)
	return m.streamToAI(prompt, "Analyzing "+ticker+"...", "stock")
}

func (m *Model) handleScreen(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/screen <filters>") + "\n" +
			DimStyle.Render("  Example: /screen high dividend tech stocks under $50"), nil
	}
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	prompt := "Screen stocks matching these criteria: " + strings.Join(args, " ") + ". List top 10 matches with ticker, price, and why they match."
	return m.streamToAI(prompt, "Screening stocks...", "stock")
}

func (m *Model) handleOdds(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/odds <event>") + "\n" +
			DimStyle.Render("  Example: /odds Lakers vs Celtics"), nil
	}
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	prompt := "Find current betting odds for: " + strings.Join(args, " ") + ". Show moneyline, spread, and over/under from major sportsbooks. Use brave-search MCP or web tools if available."
	return m.streamToAI(prompt, "Finding odds...", "bet")
}

func (m *Model) handleLines(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/lines <event>") + "\n" +
			DimStyle.Render("  Example: /lines Super Bowl"), nil
	}
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	prompt := "Show line movement and betting line history for: " + strings.Join(args, " ") + ". Highlight any significant shifts. Use brave-search MCP or web tools if available."
	return m.streamToAI(prompt, "Checking line movement...", "bet")
}
