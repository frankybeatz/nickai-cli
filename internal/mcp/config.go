package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// MCPConfig holds the full ~/.nickai/mcp.json configuration.
type MCPConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPServerConfig defines how to launch and connect to one MCP server.
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// LoadMCPConfig reads MCP server configuration from ~/.nickai/mcp.json.
// Returns an empty config (no servers) if the file doesn't exist.
func LoadMCPConfig() (*MCPConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return &MCPConfig{MCPServers: map[string]MCPServerConfig{}}, nil
	}
	path := filepath.Join(home, ".nickai", "mcp.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &MCPConfig{MCPServers: map[string]MCPServerConfig{}}, nil
		}
		return nil, err
	}
	var cfg MCPConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.MCPServers == nil {
		cfg.MCPServers = map[string]MCPServerConfig{}
	}
	return &cfg, nil
}
