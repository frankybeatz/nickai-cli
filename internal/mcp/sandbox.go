package mcp

import (
	"fmt"
	"path/filepath"
	"strings"
)

// allowedCommands is the set of commands that MCP servers are permitted to run.
var allowedCommands = map[string]bool{
	"npx":     true,
	"node":    true,
	"python":  true,
	"python3": true,
	"uvx":     true,
}

// defaultAllowedEnv is the set of environment variable names passed through by default.
var defaultAllowedEnv = map[string]bool{
	"PATH":       true,
	"HOME":       true,
	"USER":       true,
	"TERM":       true,
	"LANG":       true,
	"LC_ALL":     true,
	"NODE_PATH":  true,
	"PYTHONPATH": true,
}

// ValidateCommand checks if an MCP server command is in the allowlist.
// Returns error if the command is not allowed.
func ValidateCommand(command string) error {
	// Extract base command (strip path like /usr/bin/node -> node).
	base := filepath.Base(command)
	if allowedCommands[base] {
		return nil
	}
	return fmt.Errorf("MCP command %q is not in the allowlist (allowed: %s)",
		command, strings.Join(allowedCommandList(), ", "))
}

// allowedCommandList returns a sorted list of allowed commands for error messages.
func allowedCommandList() []string {
	list := make([]string, 0, len(allowedCommands))
	for cmd := range allowedCommands {
		list = append(list, cmd)
	}
	return list
}

// SanitizeEnv filters environment variables for MCP subprocess.
// Only passes through variables in the default allowed set plus any
// explicitly allowed vars. This prevents leaking secrets like API keys.
func SanitizeEnv(env []string, allowed map[string]bool) []string {
	var filtered []string
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if defaultAllowedEnv[key] || allowed[key] {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
