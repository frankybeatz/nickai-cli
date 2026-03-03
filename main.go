package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nickai/cli/internal/cli"
	"github.com/nickai/cli/internal/config"
	"github.com/nickai/cli/internal/logging"
	"github.com/nickai/cli/internal/mcp"
	"github.com/nickai/cli/internal/telemetry"
	"github.com/nickai/cli/internal/ui"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	// Check for --debug flag (can appear anywhere in args).
	debug := os.Getenv("NICKAI_DEBUG") == "1"
	for _, arg := range os.Args[1:] {
		if arg == "--debug" {
			debug = true
			break
		}
	}
	logging.Init(debug)
	if home, err := os.UserHomeDir(); err == nil {
		telemetry.Init(filepath.Join(home, ".nickai"))
	}
	defer telemetry.Flush()
	if debug {
		logging.Info("nickai starting", "version", version, "args", os.Args)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("nickai v%s\n", version)
			return
		case "--help", "-h", "help":
			fmt.Println("NickAI CLI — Conversational trading terminal")
			fmt.Println()
			fmt.Printf("  Usage: nickai [command] [flags]\n\n")
			fmt.Println(cli.CLICommands())
			fmt.Println()
			fmt.Println("  Other Commands:")
			fmt.Println("    mcp serve        Start MCP server (for Claude Desktop / Cursor / VS Code)")
			fmt.Println()
			fmt.Println("  Flags:")
			fmt.Println("    --help, -h       Show this help message")
			fmt.Println("    --version, -v    Print version")
			fmt.Println("    --json, -j       JSON output (for price, portfolio, orders)")
			fmt.Println("    --debug          Enable debug logging to ~/.nickai/debug.log")
			fmt.Println()
			fmt.Println("  Interactive mode:")
			fmt.Println("    nickai           Launch the TUI")
			fmt.Println("    /help            List all TUI commands")
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

	// Try CLI mode first — handles non-interactive commands like "nickai price BTC".
	if cli.Run(version) {
		return
	}

	ui.Version = version

	// Prevent Bubbletea from querying terminal background color (OSC 11).
	// Some terminals leak the response as keyboard input ("rgb:...").
	os.Setenv("TERM_PROGRAM", "nickai")

	p := tea.NewProgram(
		ui.New(),
		tea.WithAltScreen(),
	)

	// Force-quit safety net: second SIGINT kills immediately.
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh // First signal — let Bubbletea handle it.
		<-sigCh // Second signal — force exit.
		os.Exit(1)
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running NickAI: %v\n", err)
		os.Exit(1)
	}
}
