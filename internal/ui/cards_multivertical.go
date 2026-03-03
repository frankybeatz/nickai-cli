package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/api"
)

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
	case "connect":
		return renderGuideConnect()
	case "onchain":
		return renderGuideOnchain()
	case "stocks":
		return renderGuideStocks()
	case "betting":
		return renderGuideBetting()
	default:
		return ErrorStyle.Render("  Unknown guide section: ") + section + "\n" +
			DimStyle.Render("  Available: start, trading, analysis, backtest, ai, mcp, risk, polymarket, connect, onchain, stocks, betting")
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
		"  " + CommandStyle.Render("/guide connect") + DimStyle.Render("      — connecting exchanges"),
		"  " + CommandStyle.Render("/guide onchain") + DimStyle.Render("      — wallet + DeFi commands"),
		"  " + CommandStyle.Render("/guide stocks") + DimStyle.Render("       — equities commands"),
		"  " + CommandStyle.Render("/guide betting") + DimStyle.Render("      — sports betting commands"),
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

func renderGuideConnect() string {
	lines := []string{
		SecondaryStyle.Render("  Connecting Exchanges\n"),
		DimStyle.Render("  NickAI connects to exchanges via MCP servers:\n"),
		"  " + CommandStyle.Render("/connect binance") + DimStyle.Render("    — connect Binance"),
		"  " + CommandStyle.Render("/connect coinbase") + DimStyle.Render("   — connect Coinbase (via CCXT)"),
		"  " + CommandStyle.Render("/connect alpaca") + DimStyle.Render("     — connect Alpaca (stocks)"),
		"  " + CommandStyle.Render("/connect list") + DimStyle.Render("       — view connections"),
		"",
		"  " + CommandStyle.Render("/balances") + DimStyle.Render("            — unified balance view"),
		"  " + CommandStyle.Render("/positions") + DimStyle.Render("           — positions across all exchanges"),
		"  " + CommandStyle.Render("/funding") + DimStyle.Render("             — perpetual funding rates"),
		"",
		DimStyle.Render("  Next: ") + CommandStyle.Render("/guide onchain"),
	}
	return strings.Join(lines, "\n")
}

func renderGuideOnchain() string {
	lines := []string{
		SecondaryStyle.Render("  Onchain / DeFi Commands\n"),
		DimStyle.Render("  Interact with wallets and DeFi protocols:\n"),
		"  " + CommandStyle.Render("/wallet balance <addr>") + DimStyle.Render("  — check token balances"),
		"  " + CommandStyle.Render("/swap SOL USDC 10") + DimStyle.Render("      — swap on DEX (Jupiter/LiFi)"),
		"  " + CommandStyle.Render("/gas") + DimStyle.Render("                    — gas price estimates"),
		"",
		DimStyle.Render("  Requires MCP servers:"),
		"  " + CommandStyle.Render("/mcp add web3") + DimStyle.Render("   or   ") + CommandStyle.Render("/mcp add jupiter"),
		"",
		DimStyle.Render("  Next: ") + CommandStyle.Render("/guide stocks"),
	}
	return strings.Join(lines, "\n")
}

func renderGuideStocks() string {
	lines := []string{
		SecondaryStyle.Render("  Stocks & Equities\n"),
		DimStyle.Render("  Analyze and screen stocks:\n"),
		"  " + CommandStyle.Render("/stock AAPL") + DimStyle.Render("          — full stock analysis"),
		"  " + CommandStyle.Render("/screen tech < $50") + DimStyle.Render("   — stock screener"),
		"",
		DimStyle.Render("  For live data, connect Alpaca:"),
		"  " + CommandStyle.Render("/connect alpaca"),
		"",
		DimStyle.Render("  Next: ") + CommandStyle.Render("/guide betting"),
	}
	return strings.Join(lines, "\n")
}

func renderGuideBetting() string {
	lines := []string{
		SecondaryStyle.Render("  Sports Betting\n"),
		DimStyle.Render("  Find odds and track lines:\n"),
		"  " + CommandStyle.Render("/odds Lakers vs Celtics") + DimStyle.Render("  — moneyline, spread, O/U"),
		"  " + CommandStyle.Render("/lines Super Bowl") + DimStyle.Render("        — line movement history"),
		"",
		DimStyle.Render("  Uses brave-search MCP for odds data:"),
		"  " + CommandStyle.Render("/mcp add brave-search"),
		"",
		DimStyle.Render("  That covers all verticals! Type anything to start."),
	}
	return strings.Join(lines, "\n")
}

// RenderConnectHelp renders the /connect command help.
func RenderConnectHelp() string {
	header := lipgloss.NewStyle().
		Foreground(ColorPrimary).Bold(true).
		Render("  Exchange Connectivity")
	divider := "  " + Divider(50)

	return strings.Join([]string{
		"", header, divider, "",
		"  " + CommandStyle.Render("/connect <exchange>") + DimStyle.Render("   Connect an exchange via MCP"),
		"  " + CommandStyle.Render("/connect list") + DimStyle.Render("         Show connected exchanges"),
		"",
		lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("  Available exchanges:"),
		"  " + DimStyle.Render("binance, coinbase, hyperliquid, kraken, bybit, alpaca"),
		"",
		DimStyle.Render("  Example: /connect binance"),
		"",
	}, "\n") + NextSteps("/connect <exchange>")
}

// RenderWalletHelp renders the /wallet command help.
func RenderWalletHelp() string {
	header := lipgloss.NewStyle().
		Foreground(ColorPrimary).Bold(true).
		Render("  Onchain Wallet")
	divider := "  " + Divider(50)

	return strings.Join([]string{
		"", header, divider, "",
		"  " + CommandStyle.Render("/wallet balance <address>") + DimStyle.Render("  Check token balances"),
		"",
		DimStyle.Render("  Requires onchain/web3 MCP server:"),
		"  " + CommandStyle.Render("/mcp add web3"),
		"",
	}, "\n") + NextSteps("/wallet balance", "/swap")
}

// RenderBalances renders a unified balance view across paper trading and exchanges.
func RenderBalances(client *api.PapernickClient) string {
	if !client.IsConfigured() {
		return BotMsgStyle.Render("nick: ") +
			"Connect a paper trading account first with " +
			CommandStyle.Render("/config init")
	}

	portfolio, err := client.GetPortfolio()
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch portfolio: ") + err.Error()
	}

	header := lipgloss.NewStyle().
		Foreground(ColorPrimary).Bold(true).
		Render("  Unified Balances")
	divider := "  " + Divider(50)

	var rows []string
	rows = append(rows, "", header, divider, "")

	// Paper trading.
	rows = append(rows, lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("  PaperNick"))
	rows = append(rows, fmt.Sprintf("    Cash:        $%s", formatMoney(portfolio.Cash)))
	rows = append(rows, fmt.Sprintf("    Available:   $%s", formatMoney(portfolio.AvailableCash)))
	rows = append(rows, fmt.Sprintf("    Total Value: $%s", formatMoney(portfolio.TotalValue)))

	rows = append(rows, "")
	rows = append(rows, DimStyle.Render("  Connect exchanges for more: ")+CommandStyle.Render("/connect"))
	rows = append(rows, "")

	return strings.Join(rows, "\n") + NextSteps("/positions", "/pnl")
}

// RenderPositions renders a unified positions view.
func RenderPositions(client *api.PapernickClient) string {
	if !client.IsConfigured() {
		return BotMsgStyle.Render("nick: ") +
			"Connect a paper trading account first with " +
			CommandStyle.Render("/config init")
	}

	portfolio, err := client.GetPortfolio()
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch portfolio: ") + err.Error()
	}

	header := lipgloss.NewStyle().
		Foreground(ColorPrimary).Bold(true).
		Render("  Open Positions")
	divider := "  " + Divider(50)

	var rows []string
	rows = append(rows, "", header, divider, "")

	if len(portfolio.Assets) == 0 {
		rows = append(rows, DimStyle.Render("  No open positions."))
	} else {
		for _, pos := range portfolio.Assets {
			symbol := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).
				Render(fmt.Sprintf("  %-12s", pos.Symbol))
			qty := fmt.Sprintf("%.6f", pos.Quantity)
			val := fmt.Sprintf("$%s", formatMoney(pos.Value))
			rows = append(rows, symbol+DimStyle.Render(qty)+"  "+val)
		}
	}

	rows = append(rows, "")
	return strings.Join(rows, "\n") + NextSteps("/sell", "/pnl")
}
