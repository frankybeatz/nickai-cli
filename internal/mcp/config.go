package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nickai/cli/internal/safefile"
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

// mcpConfigPath returns the path to ~/.nickai/mcp.json.
func mcpConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nickai", "mcp.json"), nil
}

// saveMCPConfig writes the config to ~/.nickai/mcp.json.
func saveMCPConfig(cfg *MCPConfig) error {
	path, err := mcpConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return safefile.AtomicWrite(path, data, 0o600)
}

// AddServerToConfig adds a registry entry to ~/.nickai/mcp.json.
// extraEnv contains KEY=VALUE pairs provided inline by the user;
// these take priority over os.Getenv lookups.
func AddServerToConfig(entry *RegistryEntry, extraEnv map[string]string) error {
	path, _ := mcpConfigPath()
	mu := safefile.Lock(path)
	mu.Lock()
	defer mu.Unlock()
	cfg, err := LoadMCPConfig()
	if err != nil {
		return err
	}
	env := map[string]string{}
	for _, key := range entry.EnvKeys {
		if v, ok := extraEnv[key]; ok {
			env[key] = v
		} else if val := os.Getenv(key); val != "" {
			env[key] = val
		}
	}
	cfg.MCPServers[entry.Name] = MCPServerConfig{
		Command: entry.Command,
		Args:    entry.Args,
		Env:     env,
	}
	return saveMCPConfig(cfg)
}

// RemoveServerFromConfig removes a server from ~/.nickai/mcp.json.
func RemoveServerFromConfig(name string) error {
	path, _ := mcpConfigPath()
	mu := safefile.Lock(path)
	mu.Lock()
	defer mu.Unlock()
	cfg, err := LoadMCPConfig()
	if err != nil {
		return err
	}
	if _, ok := cfg.MCPServers[name]; !ok {
		return fmt.Errorf("Server %q not found in config.", name)
	}
	delete(cfg.MCPServers, name)
	return saveMCPConfig(cfg)
}
