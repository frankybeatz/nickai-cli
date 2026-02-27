package commands

import (
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
	TypeUnknown
)

// Result holds the parsed result of user input.
type Result struct {
	Type      CommandType
	Input     string   // original input
	Args      []string // arguments after the command
	IsCommand bool
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

	switch cmd {
	case "/help":
		return Result{Type: TypeHelp, Input: trimmed, Args: args, IsCommand: true}
	case "/agents":
		return Result{Type: TypeAgents, Input: trimmed, Args: args, IsCommand: true}
	case "/templates":
		return Result{Type: TypeTemplates, Input: trimmed, Args: args, IsCommand: true}
	case "/status":
		return Result{Type: TypeStatus, Input: trimmed, Args: args, IsCommand: true}
	case "/orders":
		return Result{Type: TypeOrders, Input: trimmed, Args: args, IsCommand: true}
	case "/price":
		return Result{Type: TypePrice, Input: trimmed, Args: args, IsCommand: true}
	case "/buy":
		return Result{Type: TypeBuy, Input: trimmed, Args: args, IsCommand: true}
	case "/sell":
		return Result{Type: TypeSell, Input: trimmed, Args: args, IsCommand: true}
	case "/config":
		return Result{Type: TypeConfig, Input: trimmed, Args: args, IsCommand: true}
	case "/clear":
		return Result{Type: TypeClear, Input: trimmed, Args: args, IsCommand: true}
	case "/quit", "/exit":
		return Result{Type: TypeQuit, Input: trimmed, Args: args, IsCommand: true}
	case "/credential", "/cred":
		return Result{Type: TypeCredential, Input: trimmed, Args: args, IsCommand: true}
	case "/workflow", "/wf":
		return Result{Type: TypeWorkflow, Input: trimmed, Args: args, IsCommand: true}
	case "/logs", "/log":
		return Result{Type: TypeLogs, Input: trimmed, Args: args, IsCommand: true}
	case "/man", "/manual":
		return Result{Type: TypeMan, Input: trimmed, Args: args, IsCommand: true}
	case "/watch":
		return Result{Type: TypeWatch, Input: trimmed, Args: args, IsCommand: true}
	case "/snapshot", "/snap":
		return Result{Type: TypeSnapshot, Input: trimmed, Args: args, IsCommand: true}
	case "/market":
		return Result{Type: TypeMarket, Input: trimmed, Args: args, IsCommand: true}
	case "/pnl":
		return Result{Type: TypePnl, Input: trimmed, Args: args, IsCommand: true}
	case "/history", "/journal":
		return Result{Type: TypeHistory, Input: trimmed, Args: args, IsCommand: true}
	case "/alert":
		return Result{Type: TypeAlert, Input: trimmed, Args: args, IsCommand: true}
	case "/chart":
		return Result{Type: TypeChart, Input: trimmed, Args: args, IsCommand: true}
	case "/theme":
		return Result{Type: TypeTheme, Input: trimmed, Args: args, IsCommand: true}
	case "/model", "/models":
		return Result{Type: TypeModel, Input: trimmed, Args: args, IsCommand: true}
	case "/mcp":
		return Result{Type: TypeMCP, Input: trimmed, Args: args, IsCommand: true}
	default:
		return Result{Type: TypeUnknown, Input: cmd, IsCommand: true}
	}
}
