package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nickai/cli/internal/tools"
)

// MCPConnection represents a live connection to an external MCP server.
type MCPConnection struct {
	Name   string
	Client *mcpclient.Client
	Tools  []mcp.Tool
}

// ClientManager manages connections to multiple external MCP servers.
type ClientManager struct {
	connections []*MCPConnection
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
			continue
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

	ctx := context.Background()
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

// RegisterTools adds all discovered MCP tools into the shared registry.
// Each tool's executor proxies calls back to the originating MCP server.
func (cm *ClientManager) RegisterTools(registry *tools.Registry) {
	for _, conn := range cm.connections {
		for _, tool := range conn.Tools {
			entry := tools.ToolEntry{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: marshalSchema(tool.InputSchema),
				Execute:     makeProxyExecutor(conn.Client, tool.Name),
				Source:      conn.Name,
			}
			registry.Register(entry)
		}
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
