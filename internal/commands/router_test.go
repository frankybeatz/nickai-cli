package commands

import (
	"strings"
	"testing"
)

func TestRouteEmpty(t *testing.T) {
	r := Route("")
	if r.Type != TypeChat || r.Input != "" {
		t.Errorf("empty input: got Type=%d Input=%q", r.Type, r.Input)
	}
}

func TestRouteWhitespace(t *testing.T) {
	r := Route("   ")
	if r.Type != TypeChat || r.Input != "" {
		t.Errorf("whitespace input: got Type=%d Input=%q", r.Type, r.Input)
	}
}

func TestRouteChatMessage(t *testing.T) {
	r := Route("what is the price of BTC?")
	if r.Type != TypeChat {
		t.Errorf("expected TypeChat, got %d", r.Type)
	}
	if r.IsCommand {
		t.Error("expected IsCommand=false for chat")
	}
}

func TestRouteBasicCommands(t *testing.T) {
	tests := []struct {
		input    string
		expected CommandType
	}{
		{"/help", TypeHelp},
		{"/buy BTC 0.1", TypeBuy},
		{"/sell ETH 1.0", TypeSell},
		{"/price BTC", TypePrice},
		{"/status", TypeStatus},
		{"/orders", TypeOrders},
		{"/clear", TypeClear},
		{"/quit", TypeQuit},
		{"/pnl", TypePnl},
		{"/history", TypeHistory},
		{"/chart BTC", TypeChart},
		{"/alert BTC > 100000", TypeAlert},
		{"/analyze BTC", TypeAnalyze},
		{"/backtest presets", TypeBacktest},
		{"/memory", TypeMemory},
		{"/consensus BTC", TypeConsensus},
		{"/auto list", TypeAuto},
		{"/risk show", TypeRisk},
		{"/strategy list", TypeStrategy},
		{"/trigger list", TypeTrigger},
		{"/notify show", TypeNotify},
		{"/analytics", TypeAnalytics},
		{"/polymarket scan", TypePolymarket},
		{"/guide", TypeGuide},
		{"/config", TypeConfig},
		{"/model", TypeModel},
		{"/theme cyberpunk", TypeTheme},
		{"/mcp list", TypeMCP},
		{"/credential list", TypeCredential},
		{"/workflow list", TypeWorkflow},
		{"/logs test-wf", TypeLogs},
		{"/man buy", TypeMan},
		{"/watch BTC ETH", TypeWatch},
		{"/snapshot", TypeSnapshot},
		{"/market", TypeMarket},
	}

	for _, tt := range tests {
		r := Route(tt.input)
		if r.Type != tt.expected {
			t.Errorf("Route(%q): got Type=%d, want %d", tt.input, r.Type, tt.expected)
		}
		if !r.IsCommand {
			t.Errorf("Route(%q): expected IsCommand=true", tt.input)
		}
	}
}

func TestRouteMultiVertical(t *testing.T) {
	tests := []struct {
		input    string
		expected CommandType
	}{
		{"/connect binance", TypeConnect},
		{"/balances", TypeBalances},
		{"/positions", TypePositions},
		{"/markets", TypeMarkets},
		{"/bet yes 100", TypeBet},
		{"/wallet balance 0xabc", TypeWallet},
		{"/swap SOL USDC 10", TypeSwap},
		{"/gas", TypeGas},
		{"/stock AAPL", TypeStock},
		{"/screen tech", TypeScreen},
		{"/odds Lakers", TypeOdds},
		{"/lines NFL", TypeLines},
		{"/funding", TypeFunding},
	}

	for _, tt := range tests {
		r := Route(tt.input)
		if r.Type != tt.expected {
			t.Errorf("Route(%q): got Type=%d, want %d", tt.input, r.Type, tt.expected)
		}
	}
}

func TestRouteAliases(t *testing.T) {
	aliases := []struct {
		alias    string
		canonical string
		expected  CommandType
	}{
		{"/p BTC", "/price BTC", TypePrice},
		{"/b BTC 0.1", "/buy BTC 0.1", TypeBuy},
		{"/s ETH 1", "/sell ETH 1", TypeSell},
		{"/bt presets", "/backtest presets", TypeBacktest},
		{"/pm scan", "/polymarket scan", TypePolymarket},
		{"/mem", "/memory", TypeMemory},
		{"/con BTC", "/consensus BTC", TypeConsensus},
		{"/bal", "/balances", TypeBalances},
		{"/pos", "/positions", TypePositions},
		{"/wf list", "/workflow list", TypeWorkflow},
		{"/cred list", "/credential list", TypeCredential},
		{"/trig list", "/trigger list", TypeTrigger},
		{"/strat list", "/strategy list", TypeStrategy},
		{"/snap", "/snapshot", TypeSnapshot},
		{"/exit", "/quit", TypeQuit},
		{"/log test", "/logs test", TypeLogs},
		{"/journal", "/history", TypeHistory},
		{"/manual buy", "/man buy", TypeMan},
		{"/models", "/model", TypeModel},
		{"/automation list", "/auto list", TypeAuto},
	}

	for _, tt := range aliases {
		r := Route(tt.alias)
		if r.Type != tt.expected {
			t.Errorf("Route(%q) alias: got Type=%d, want %d (canonical: %s)", tt.alias, r.Type, tt.expected, tt.canonical)
		}
	}
}

func TestRouteCaseInsensitive(t *testing.T) {
	r := Route("/BUY BTC 0.1")
	if r.Type != TypeBuy {
		t.Errorf("uppercase command: got Type=%d, want TypeBuy", r.Type)
	}

	r = Route("/Price BTC")
	if r.Type != TypePrice {
		t.Errorf("mixed case command: got Type=%d, want TypePrice", r.Type)
	}
}

func TestRouteArgs(t *testing.T) {
	r := Route("/buy BTC 0.1 limit 65000")
	if r.Type != TypeBuy {
		t.Fatalf("expected TypeBuy, got %d", r.Type)
	}
	if len(r.Args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(r.Args), r.Args)
	}
	if r.Args[0] != "BTC" || r.Args[1] != "0.1" || r.Args[2] != "limit" || r.Args[3] != "65000" {
		t.Errorf("args mismatch: %v", r.Args)
	}
}

func TestRouteSubcommands(t *testing.T) {
	// Connect uses subcommand parsing.
	r := Route("/connect binance key123")
	if r.Type != TypeConnect {
		t.Fatalf("expected TypeConnect, got %d", r.Type)
	}
	if r.SubCommand != "binance" {
		t.Errorf("SubCommand: got %q, want %q", r.SubCommand, "binance")
	}
	if len(r.Args) != 1 || r.Args[0] != "key123" {
		t.Errorf("Args: got %v, want [key123]", r.Args)
	}

	// Wallet uses subcommand parsing.
	r = Route("/wallet balance 0xabc")
	if r.SubCommand != "balance" {
		t.Errorf("SubCommand: got %q, want %q", r.SubCommand, "balance")
	}

	// Markets uses subcommand parsing.
	r = Route("/markets trending")
	if r.SubCommand != "trending" {
		t.Errorf("SubCommand: got %q, want %q", r.SubCommand, "trending")
	}
}

func TestRouteNoSubcommandForLegacy(t *testing.T) {
	// Backtest does NOT use subcommand parsing (handled internally).
	r := Route("/backtest presets")
	if r.SubCommand != "" {
		t.Errorf("backtest should not have SubCommand, got %q", r.SubCommand)
	}
	if len(r.Args) != 1 || r.Args[0] != "presets" {
		t.Errorf("Args: got %v, want [presets]", r.Args)
	}
}

func TestRouteUnknownCommand(t *testing.T) {
	r := Route("/foobar")
	if r.Type != TypeUnknown {
		t.Errorf("expected TypeUnknown, got %d", r.Type)
	}
	if !r.IsCommand {
		t.Error("expected IsCommand=true for unknown command")
	}
}

func TestRouteFuzzySuggestion(t *testing.T) {
	r := Route("/pri")
	if r.Type != TypeUnknown {
		t.Fatalf("expected TypeUnknown, got %d", r.Type)
	}
	if len(r.Args) == 0 {
		t.Fatal("expected fuzzy suggestion in Args")
	}
	if !strings.Contains(r.Args[0], "Did you mean") {
		t.Errorf("expected suggestion, got %q", r.Args[0])
	}
	if !strings.Contains(r.Args[0], "/price") {
		t.Errorf("expected /price in suggestion, got %q", r.Args[0])
	}
}

func TestRoutePreservesInput(t *testing.T) {
	r := Route("/buy BTC 0.1")
	if r.Input != "/buy BTC 0.1" {
		t.Errorf("Input: got %q, want %q", r.Input, "/buy BTC 0.1")
	}
}
