package commands

import (
	"sort"
	"strings"
)

// CommandType identifies what kind of response was produced.
type CommandType int

const (
	TypeChat CommandType = iota
	TypeHelp
	TypeAgents
	TypeTemplates
	TypeStatus
	TypeOrders
	TypePrice
	TypeBuy
	TypeSell
	TypeConfig
	TypeClear
	TypeQuit
	TypeCredential
	TypeWorkflow
	TypeLogs
	TypeMan
	TypeWatch
	TypeSnapshot
	TypeMarket
	TypePnl
	TypeHistory
	TypeAlert
	TypeChart
	TypeTheme
	TypeModel
	TypeMCP
	TypeTrigger
	TypeRisk
	TypeStrategy
	TypeNotify
	TypeAnalytics
	TypeAnalyze
	TypeAuto
	TypeBacktest
	TypePolymarket
	TypeGuide
	TypeMemory
	TypeConsensus

	// v0.5.1 — multi-vertical commands.
	TypeConnect   // /connect [exchange]
	TypeBalances  // /balances, /bal
	TypePositions // /positions, /pos
	TypeMarkets   // /markets
	TypeBet       // /bet
	TypeWallet    // /wallet
	TypeSwap      // /swap
	TypeGas       // /gas
	TypeStock     // /stock
	TypeScreen    // /screen
	TypeOdds      // /odds
	TypeLines     // /lines
	TypeFunding   // /funding
	TypeDashboard // /dashboard, /dash
	TypeVibe      // /vibe
	TypeExport    // /export

	TypeUnknown
)

// Result holds the parsed result of user input.
type Result struct {
	Type       CommandType
	SubCommand string   // e.g. "scan", "analyze", "connect"
	Input      string   // original input
	Args       []string // arguments after the subcommand (or after the command if no sub)
	IsCommand  bool
}

// allCommands maps every command name (including aliases) to its type.
// Generated from the unified Registry in defs.go.
var allCommands = BuildCommandMap()

// subcommandCommands lists NEW command types that parse SubCommand from args.
// Existing commands (backtest, analyze, memory, etc.) parse subcommands
// internally in their handlers for backward compatibility.
var subcommandCommands = map[CommandType]bool{
	TypeConnect: true,
	TypeWallet:  true,
	TypeMarkets: true,
}

// Route parses raw user input and returns a Result.
// The UI layer is responsible for rendering.
func Route(input string) Result {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return Result{}
	}

	if !strings.HasPrefix(trimmed, "/") {
		return Result{Type: TypeChat, Input: trimmed, IsCommand: false}
	}

	parts := strings.Fields(trimmed)
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	if ct, ok := allCommands[cmd]; ok {
		r := Result{Type: ct, Input: trimmed, Args: args, IsCommand: true}

		// Parse subcommand for commands that use them.
		if subcommandCommands[ct] && len(args) > 0 {
			r.SubCommand = strings.ToLower(args[0])
			r.Args = args[1:]
		}

		return r
	}

	// Unknown command — try fuzzy suggestion.
	r := Result{Type: TypeUnknown, Input: cmd, IsCommand: true}
	if suggestions := fuzzySuggest(cmd, 2); len(suggestions) > 0 {
		r.Args = []string{"Did you mean " + strings.Join(suggestions, " or ") + "?"}
	}
	return r
}

// fuzzySuggest returns up to maxResults command names that start with the
// given prefix. Only considers canonical (non-alias) commands.
func fuzzySuggest(prefix string, maxResults int) []string {
	// De-duplicate: only suggest the longest form of each command type.
	seen := map[CommandType]bool{}
	var matches []string
	for cmd, ct := range allCommands {
		if seen[ct] {
			continue
		}
		if strings.HasPrefix(cmd, prefix) && cmd != prefix {
			matches = append(matches, cmd)
			seen[ct] = true
		}
	}
	sort.Strings(matches)
	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}
	return matches
}
