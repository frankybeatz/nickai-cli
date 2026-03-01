package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nickai/cli/internal/risk"
	"github.com/nickai/cli/internal/tools"
)

// MCPConnection represents a live connection to an external MCP server.
type MCPConnection struct {
	Name         string
	Client       *mcpclient.Client
	Tools        []mcp.Tool
	Capabilities []Capability
}

// FailedConnection records a server that failed to start.
type FailedConnection struct {
	Name  string
	Error string
}

// ClientManager manages connections to multiple external MCP servers.
type ClientManager struct {
	connections []*MCPConnection
	failed      []FailedConnection
}

// NewClientManager creates an empty client manager.
func NewClientManager() *ClientManager {
	return &ClientManager{}
}

// ConnectAll reads the MCP config and starts all configured servers.
// Errors on individual servers are logged but do not block others.
func (cm *ClientManager) ConnectAll(cfg *MCPConfig) {
	for name, serverCfg := range cfg.MCPServers {
		conn, err := cm.connect(name, serverCfg)
		if err != nil {
			log.Printf("MCP: failed to connect to %s: %v", name, err)
			cm.failed = append(cm.failed, FailedConnection{Name: name, Error: err.Error()})
			continue
		}
		// Attach capabilities from curated registry if available.
		if entry := GetEntry(name); entry != nil {
			conn.Capabilities = entry.Capabilities
		}
		cm.connections = append(cm.connections, conn)
	}
}

func (cm *ClientManager) connect(name string, cfg MCPServerConfig) (*MCPConnection, error) {
	// Build env slice from map, inheriting current env.
	env := os.Environ()
	for k, v := range cfg.Env {
		env = append(env, k+"="+v)
	}

	client, err := mcpclient.NewStdioMCPClient(cfg.Command, env, cfg.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to start %s: %w", name, err)
	}

	// Use a timeout so a hanging server doesn't block startup forever.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "nickai",
		Version: "0.4.0",
	}

	_, err = client.Initialize(ctx, initReq)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("initialize failed for %s: %w", name, err)
	}

	// Discover tools.
	toolsResult, err := client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("list tools failed for %s: %w", name, err)
	}

	return &MCPConnection{
		Name:   name,
		Client: client,
		Tools:  toolsResult.Tools,
	}, nil
}

// RiskLimitsFunc returns current risk limits for trade confirmation checks.
type RiskLimitsFunc func() *risk.RiskLimits

// RegisterTools adds all discovered MCP tools into the shared registry.
// Servers with trade or on-chain capabilities get a confirming proxy that
// asks for user approval before executing. riskFn may be nil (risk checks skipped).
func (cm *ClientManager) RegisterTools(registry *tools.Registry, riskFn RiskLimitsFunc) {
	for _, conn := range cm.connections {
		hasTrade := hasTradeCapability(conn.Capabilities)
		for _, tool := range conn.Tools {
			var executor tools.ToolFunc
			if hasTrade {
				executor = makeConfirmingProxyExecutor(registry, conn.Client, tool.Name, riskFn)
			} else {
				executor = makeProxyExecutor(conn.Client, tool.Name)
			}
			entry := tools.ToolEntry{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: marshalSchema(tool.InputSchema),
				Execute:     executor,
				Source:      conn.Name,
			}
			registry.Register(entry)
		}
	}
}

// hasTradeCapability returns true if the capability list includes CapTrade or CapOnChain.
func hasTradeCapability(caps []Capability) bool {
	for _, c := range caps {
		if c == CapTrade || c == CapOnChain {
			return true
		}
	}
	return false
}

// makeConfirmingProxyExecutor wraps the proxy executor with a best-effort risk
// check and a user confirmation prompt (same pattern as builtin place_order).
func makeConfirmingProxyExecutor(registry *tools.Registry, client *mcpclient.Client, toolName string, riskFn RiskLimitsFunc) tools.ToolFunc {
	direct := makeProxyExecutor(client, toolName)
	return func(ctx context.Context, rawInput json.RawMessage) (string, error) {
		// Best-effort: parse symbol/side/quantity/price from input JSON.
		var fields struct {
			Symbol   string  `json:"symbol"`
			Side     string  `json:"side"`
			Quantity float64 `json:"quantity"`
			Amount   float64 `json:"amount"`
			Price    float64 `json:"price"`
		}
		_ = json.Unmarshal(rawInput, &fields)

		// Best-effort risk check (portfolio-level checks skipped for MCP tools).
		if riskFn != nil && fields.Symbol != "" {
			limits := riskFn()
			if limits != nil && !limits.IsEmpty() {
				qty := fields.Quantity
				if qty == 0 {
					qty = fields.Amount
				}
				price := fields.Price
				side := strings.ToLower(fields.Side)
				if side == "" {
					side = "buy"
				}
				if qty > 0 && price > 0 {
					result := risk.CheckOrder(limits, nil, fields.Symbol, side, qty, price)
					if !result.Allowed {
						return tools.ErrorJSON("Risk limit: " + result.Reason), nil
					}
				}
			}
		}

		// Build display string for confirmation.
		display := fmt.Sprintf("MCP Tool: %s", toolName)
		if fields.Symbol != "" {
			display = fmt.Sprintf("%s %s %s", strings.ToUpper(fields.Side), fields.Symbol, toolName)
		}

		// Send confirmation request and block.
		registry.ConfirmCh <- tools.ConfirmRequest{
			ToolName: toolName,
			Input:    rawInput,
			Display:  display,
		}
		resp := <-registry.ResponseCh

		if !resp.Approved {
			return tools.ToJSON(map[string]string{
				"status": "cancelled",
				"reason": "User declined the action",
			}), nil
		}
		return direct(ctx, rawInput)
	}
}

// makeProxyExecutor returns a ToolFunc that forwards calls to an MCP server.
func makeProxyExecutor(client *mcpclient.Client, toolName string) tools.ToolFunc {
	return func(ctx context.Context, rawInput json.RawMessage) (string, error) {
		var args map[string]any
		if err := json.Unmarshal(rawInput, &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}

		callReq := mcp.CallToolRequest{}
		callReq.Params.Name = toolName
		callReq.Params.Arguments = args

		result, err := client.CallTool(ctx, callReq)
		if err != nil {
			return "", err
		}

		if result.IsError {
			return tools.ErrorJSON(getTextFromResult(result)), nil
		}
		return getTextFromResult(result), nil
	}
}

func marshalSchema(schema mcp.ToolInputSchema) json.RawMessage {
	data, err := json.Marshal(schema)
	if err != nil {
		return json.RawMessage(`{"type":"object"}`)
	}
	return data
}

func getTextFromResult(result *mcp.CallToolResult) string {
	for _, c := range result.Content {
		if tc, ok := mcp.AsTextContent(c); ok {
			return tc.Text
		}
	}
	return "{}"
}

// CloseAll shuts down all MCP server connections.
func (cm *ClientManager) CloseAll() {
	for _, conn := range cm.connections {
		conn.Client.Close()
	}
}

// ConnectionCount returns the number of active connections.
func (cm *ClientManager) ConnectionCount() int {
	return len(cm.connections)
}

// Connections returns all active connections (for status display).
func (cm *ClientManager) Connections() []*MCPConnection {
	return cm.connections
}

// Failed returns servers that failed to connect on startup.
func (cm *ClientManager) Failed() []FailedConnection {
	return cm.failed
}
