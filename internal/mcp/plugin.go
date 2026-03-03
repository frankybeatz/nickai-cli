package mcp

import (
	"fmt"
	"os"
	"strings"

	"github.com/nickai/cli/internal/safefile"
)

// PluginEntry describes an installable MCP server plugin.
type PluginEntry struct {
	Name        string            // short name used with /plugin install
	Description string            // one-liner
	Command     string            // command to run (e.g. "npx")
	Args        []string          // default args
	Env         map[string]string // required env vars with human-readable hints
	Tags        []string          // searchable tags
	RequiresNpx bool              // true if this plugin requires npx
}

// PluginRegistry is the curated list of installable MCP server plugins.
// It includes official Model Context Protocol servers plus any servers
// already present in the CuratedRegistry.
var PluginRegistry []PluginEntry

func init() {
	// Seed from the official MCP servers not already in CuratedRegistry.
	officialPlugins := []PluginEntry{
		{
			Name:        "filesystem",
			Description: "Read, write, and manage files on your local filesystem",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
			Tags:        []string{"files", "filesystem", "local", "read", "write"},
			RequiresNpx: true,
		},
		{
			Name:        "github",
			Description: "GitHub repos, issues, PRs, and code search",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-github"},
			Env:         map[string]string{"GITHUB_TOKEN": "personal access token from github.com/settings/tokens"},
			Tags:        []string{"github", "git", "repos", "issues", "code"},
			RequiresNpx: true,
		},
		{
			Name:        "sqlite",
			Description: "Query and manage SQLite databases",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-sqlite"},
			Tags:        []string{"sqlite", "database", "sql", "query"},
			RequiresNpx: true,
		},
		{
			Name:        "memory",
			Description: "Persistent memory and knowledge graph for AI conversations",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-memory"},
			Tags:        []string{"memory", "knowledge", "graph", "persistent", "context"},
			RequiresNpx: true,
		},
		{
			Name:        "fetch",
			Description: "Fetch and extract content from web URLs",
			Command:     "npx",
			Args:        []string{"-y", "@modelcontextprotocol/server-fetch"},
			Tags:        []string{"fetch", "web", "url", "scrape", "http"},
			RequiresNpx: true,
		},
	}

	// Start with official plugins.
	PluginRegistry = append(PluginRegistry, officialPlugins...)

	// Add entries from the CuratedRegistry, converting them to PluginEntry.
	for _, entry := range CuratedRegistry {
		// Skip if already added (check by name).
		if GetPlugin(entry.Name) != nil {
			continue
		}
		env := map[string]string{}
		for _, key := range entry.EnvKeys {
			hint := key
			if entry.EnvHints != nil {
				if h, ok := entry.EnvHints[key]; ok {
					hint = h
				}
			}
			env[key] = hint
		}
		pe := PluginEntry{
			Name:        entry.Name,
			Description: entry.Description,
			Command:     entry.Command,
			Args:        entry.Args,
			Env:         env,
			Tags:        entry.Tags,
			RequiresNpx: entry.Command == "npx",
		}
		PluginRegistry = append(PluginRegistry, pe)
	}
}

// GetPlugin returns a plugin entry by name, or nil if not found.
func GetPlugin(name string) *PluginEntry {
	for i := range PluginRegistry {
		if PluginRegistry[i].Name == name {
			return &PluginRegistry[i]
		}
	}
	return nil
}

// SearchPlugins returns plugins matching the query string against
// name, description, and tags. Empty query returns all entries.
func SearchPlugins(query string) []PluginEntry {
	if query == "" {
		return PluginRegistry
	}
	var results []PluginEntry
	q := strings.ToLower(query)
	for _, p := range PluginRegistry {
		if pluginMatchesQuery(p, q) {
			results = append(results, p)
		}
	}
	return results
}

func pluginMatchesQuery(p PluginEntry, q string) bool {
	if strings.Contains(strings.ToLower(p.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(p.Description), q) {
		return true
	}
	for _, tag := range p.Tags {
		if strings.Contains(tag, q) {
			return true
		}
	}
	return false
}

// ListInstalled returns names of currently configured MCP servers.
func ListInstalled() []string {
	cfg, err := LoadMCPConfig()
	if err != nil {
		return nil
	}
	var names []string
	for name := range cfg.MCPServers {
		names = append(names, name)
	}
	return names
}

// IsInstalled returns true if the named plugin is currently in mcp.json.
func IsInstalled(name string) bool {
	cfg, err := LoadMCPConfig()
	if err != nil {
		return false
	}
	_, ok := cfg.MCPServers[name]
	return ok
}

// InstallPlugin adds a plugin's server config to ~/.nickai/mcp.json.
// extraEnv contains KEY=VALUE pairs provided by the user; these take
// priority over os.Getenv lookups.
func InstallPlugin(name string, extraEnv map[string]string) error {
	plugin := GetPlugin(name)
	if plugin == nil {
		return fmt.Errorf("unknown plugin %q — use /plugin search to browse available plugins", name)
	}

	// Check if already installed.
	if IsInstalled(name) {
		return fmt.Errorf("plugin %q is already installed", name)
	}

	// Build env map, resolving values.
	env := map[string]string{}
	for key := range plugin.Env {
		if v, ok := extraEnv[key]; ok {
			env[key] = v
		} else if val := os.Getenv(key); val != "" {
			env[key] = val
		}
	}

	// Also try the curated registry path (AddServerToConfig) if the entry
	// exists there, for consistency.
	if entry := GetEntry(name); entry != nil {
		return AddServerToConfig(entry, extraEnv)
	}

	// Direct install for plugins not in the curated registry.
	path, _ := mcpConfigPath()
	mu := safefile.Lock(path)
	mu.Lock()
	defer mu.Unlock()

	cfg, err := LoadMCPConfig()
	if err != nil {
		return err
	}
	cfg.MCPServers[name] = MCPServerConfig{
		Command: plugin.Command,
		Args:    plugin.Args,
		Env:     env,
	}
	return saveMCPConfig(cfg)
}

// RemovePlugin removes a plugin from ~/.nickai/mcp.json.
func RemovePlugin(name string) error {
	return RemoveServerFromConfig(name)
}

// MissingEnvKeys returns the env keys that the plugin requires but
// are not provided in extraEnv or the current environment.
func MissingEnvKeys(name string, extraEnv map[string]string) []string {
	plugin := GetPlugin(name)
	if plugin == nil {
		return nil
	}
	var missing []string
	for key := range plugin.Env {
		if _, ok := extraEnv[key]; ok {
			continue
		}
		if os.Getenv(key) != "" {
			continue
		}
		missing = append(missing, key)
	}
	return missing
}
