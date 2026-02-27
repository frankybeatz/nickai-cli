package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nickai/cli/internal/config"
	"github.com/nickai/cli/internal/mcp"
	"github.com/nickai/cli/internal/ui"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("nickai v%s\n", version)
			return
		case "--help", "-h", "help":
			fmt.Println("NickAI CLI — Conversational trading terminal")
			fmt.Println()
			fmt.Printf("  Usage: nickai [command] [flags]\n\n")
			fmt.Println("  Commands:")
			fmt.Println("    mcp serve        Start MCP server (for Claude Desktop / Cursor / VS Code)")
			fmt.Println()
			fmt.Println("  Flags:")
			fmt.Println("    --help, -h       Show this help message")
			fmt.Println("    --version, -v    Print version")
			fmt.Println()
			fmt.Println("  Inside the terminal:")
			fmt.Println("    /help            List all commands")
			fmt.Println("    /config          Manage API keys")
			fmt.Println("    /man <command>   Detailed manual pages")
			fmt.Println("    Esc              Enter vim NORMAL mode")
			fmt.Println()
			fmt.Printf("  v%s  |  https://github.com/frankybeatz/nickai-cli\n", version)
			return
		case "mcp":
			if len(os.Args) > 2 && os.Args[2] == "serve" {
				cfg, err := config.Load()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
					os.Exit(1)
				}
				if err := mcp.ServeStdio(cfg, version); err != nil {
					fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
					os.Exit(1)
				}
				return
			}
			fmt.Fprintln(os.Stderr, "Usage: nickai mcp serve")
			os.Exit(1)
		}
	}

	ui.Version = version

	p := tea.NewProgram(
		ui.New(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running NickAI: %v\n", err)
		os.Exit(1)
	}
}
