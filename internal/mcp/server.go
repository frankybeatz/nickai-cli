package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/config"
	"github.com/nickai/cli/internal/tools"
)

// ServeStdio starts the MCP server on stdin/stdout. This blocks until the
// client disconnects. No TUI is started — this is designed for use by
// Claude Desktop, Cursor, VS Code, or any MCP-compatible client.
func ServeStdio(cfg *config.Config, version string) error {
	client := api.NewClient(cfg)
	registry := tools.NewRegistry()
	tools.RegisterBuiltins(registry, client, nil)

	s := mcpserver.NewMCPServer(
		"nickai",
		version,
		mcpserver.WithToolCapabilities(false),
	)

	// Convert each registry entry to an MCP tool and register it.
	for _, entry := range registry.All() {
		mcpTool := mcp.NewToolWithRawSchema(entry.Name, entry.Description, entry.InputSchema)
		s.AddTool(mcpTool, makeHandler(entry))
	}

	return mcpserver.ServeStdio(s)
}

// makeHandler bridges an MCP CallToolRequest to the internal ToolFunc.
func makeHandler(entry *tools.ToolEntry) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		rawArgs, err := json.Marshal(req.GetArguments())
		if err != nil {
			return mcp.NewToolResultError("invalid arguments: " + err.Error()), nil
		}
		result, execErr := entry.Execute(ctx, rawArgs)
		if execErr != nil {
			return mcp.NewToolResultError(execErr.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	}
}
