package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nickai/cli/internal/ai"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/config"
	"github.com/nickai/cli/internal/tools"
)

// Run checks os.Args for a CLI subcommand and executes it without launching the TUI.
// Returns true if a CLI command was handled (caller should exit), false to fall through to TUI.
func Run(version string) bool {
	if len(os.Args) < 2 {
		return false
	}

	cmd := strings.ToLower(os.Args[1])
	args := os.Args[2:]

	switch cmd {
	case "price", "prices", "p":
		return runPrice(args)
	case "portfolio", "status":
		return runPortfolio()
	case "orders":
		return runOrders()
	case "ask":
		return runAsk(args)
	case "analyze":
		return runAnalyze(args)
	case "consensus":
		return runConsensus(args)
	default:
		return false
	}
}

// CLICommands returns help text for CLI mode commands.
func CLICommands() string {
	return `  CLI Commands (non-interactive):
    nickai price BTC ETH SOL     Print live prices
    nickai portfolio             Print portfolio summary
    nickai orders                Print recent orders
    nickai ask "question"        Ask Nick a question
    nickai analyze BTC           Run technical analysis
    nickai consensus BTC         Multi-model consensus`
}

func loadClient() (*config.Config, *api.PapernickClient) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}
	return cfg, api.NewClient(cfg)
}

func runPrice(args []string) bool {
	if len(args) == 0 {
		args = []string{"BTC", "ETH", "SOL"}
	}

	_, client := loadClient()
	if !client.IsConfigured() {
		fmt.Fprintln(os.Stderr, "No API key configured. Run: nickai (TUI) then /config init")
		os.Exit(1)
	}

	// Check if user wants JSON output.
	jsonOutput := false
	var symbols []string
	for _, a := range args {
		if a == "--json" || a == "-j" {
			jsonOutput = true
		} else {
			symbols = append(symbols, strings.ToUpper(a))
		}
	}

	prices, err := client.GetPrices(symbols)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching prices: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(prices)
		return true
	}

	for _, p := range prices {
		sym := strings.TrimSuffix(p.Symbol, "USDT")
		fmt.Printf("%-6s %s\n", sym, fmtPrice(p.Price))
	}
	return true
}

func runPortfolio() bool {
	_, client := loadClient()
	if !client.IsConfigured() {
		fmt.Fprintln(os.Stderr, "No API key configured. Run: nickai (TUI) then /config init")
		os.Exit(1)
	}

	// Check for --json flag.
	jsonOutput := false
	for _, a := range os.Args[2:] {
		if a == "--json" || a == "-j" {
			jsonOutput = true
		}
	}

	portfolio, err := client.GetPortfolio()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(portfolio)
		return true
	}

	fmt.Printf("Portfolio Value: $%.2f\n", portfolio.TotalValue)
	fmt.Printf("Cash: $%.2f\n", portfolio.AvailableCash)
	if len(portfolio.Assets) > 0 {
		fmt.Println("\nPositions:")
		for _, pos := range portfolio.Assets {
			sym := strings.TrimSuffix(pos.Symbol, "USDT")
			fmt.Printf("  %-6s  qty: %.4f  val: $%.2f\n", sym, pos.Quantity, pos.Value)
		}
	}
	return true
}

func runOrders() bool {
	_, client := loadClient()
	if !client.IsConfigured() {
		fmt.Fprintln(os.Stderr, "No API key configured. Run: nickai (TUI) then /config init")
		os.Exit(1)
	}

	// Check for --json flag.
	jsonOutput := false
	for _, a := range os.Args[2:] {
		if a == "--json" || a == "-j" {
			jsonOutput = true
		}
	}

	orders, err := client.GetOrders()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(orders)
		return true
	}

	if len(orders) == 0 {
		fmt.Println("No orders.")
		return true
	}

	for _, o := range orders {
		sym := strings.TrimSuffix(o.Symbol, "USDT")
		price := fmtPrice(o.FilledPrice)
		if o.FilledPrice == 0 {
			price = fmtPrice(o.Price)
		}
		fmt.Printf("%-6s %-4s  qty: %.4f  price: %s  status: %s\n",
			sym, strings.ToUpper(o.Side), o.Quantity, price, o.Status)
	}
	return true
}

func runAsk(args []string) bool {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: nickai ask \"your question\"")
		os.Exit(1)
	}

	question := strings.Join(args, " ")
	cfg, client := loadClient()

	anthKey := cfg.AnthropicKeyOrEnv()
	if anthKey == "" {
		fmt.Fprintln(os.Stderr, "No AI key configured. Set ANTHROPIC_API_KEY or run /config set anthropic_key in TUI.")
		os.Exit(1)
	}

	registry := tools.NewRegistry()
	tools.RegisterBuiltins(registry, client, nil)
	agent := ai.NewAgent(client, anthKey, registry, cfg.Vibe)
	if cfg.Model != "" {
		_ = agent.SetModel(cfg.Model)
	}

	fmt.Fprint(os.Stderr, "Thinking...")
	resp, err := agent.Chat(context.Background(),question)
	fmt.Fprint(os.Stderr, "\r            \r") // clear "Thinking..."
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(resp)
	return true
}

func runAnalyze(args []string) bool {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: nickai analyze <symbol>")
		os.Exit(1)
	}

	symbol := strings.ToUpper(args[0])
	cfg, client := loadClient()

	anthKey := cfg.AnthropicKeyOrEnv()
	if anthKey == "" {
		fmt.Fprintln(os.Stderr, "No AI key configured. Set ANTHROPIC_API_KEY or run /config set anthropic_key in TUI.")
		os.Exit(1)
	}

	registry := tools.NewRegistry()
	tools.RegisterBuiltins(registry, client, nil)
	agent := ai.NewAgent(client, anthKey, registry, cfg.Vibe)
	if cfg.Model != "" {
		_ = agent.SetModel(cfg.Model)
	}

	fmt.Fprint(os.Stderr, "Analyzing "+symbol+"...")
	resp, err := agent.Chat(context.Background(),fmt.Sprintf("Analyze %s. Give me price, RSI, MACD, support/resistance, and your recommendation.", symbol))
	fmt.Fprint(os.Stderr, "\r"+strings.Repeat(" ", 30)+"\r")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(resp)
	return true
}

func runConsensus(args []string) bool {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: nickai consensus <symbol>")
		os.Exit(1)
	}

	symbol := strings.ToUpper(args[0])
	cfg, client := loadClient()

	anthKey := cfg.AnthropicKeyOrEnv()
	if anthKey == "" {
		fmt.Fprintln(os.Stderr, "No AI key configured. Set ANTHROPIC_API_KEY or run /config set anthropic_key in TUI.")
		os.Exit(1)
	}

	registry := tools.NewRegistry()
	tools.RegisterBuiltins(registry, client, nil)
	agent := ai.NewAgent(client, anthKey, registry, cfg.Vibe)
	if cfg.Model != "" {
		_ = agent.SetModel(cfg.Model)
	}

	fmt.Fprint(os.Stderr, "Running consensus on "+symbol+"...")
	resp, err := agent.Chat(context.Background(),fmt.Sprintf("Run a full consensus analysis on %s. Use get_prices first, then give me your verdict with specific levels.", symbol))
	fmt.Fprint(os.Stderr, "\r"+strings.Repeat(" ", 40)+"\r")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(resp)
	return true
}

func fmtPrice(v float64) string {
	if v >= 1 {
		return fmt.Sprintf("$%.2f", v)
	}
	return fmt.Sprintf("$%.6f", v)
}
