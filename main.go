package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nickai/cli/internal/ui"
)

const version = "0.3.0"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("nickai v%s\n", version)
			return
		case "--help", "-h", "help":
			fmt.Println("NickAI CLI — Conversational trading terminal")
			fmt.Println()
			fmt.Printf("  Usage: nickai [flags]\n\n")
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
		}
	}

	p := tea.NewProgram(
		ui.New(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running NickAI: %v\n", err)
		os.Exit(1)
	}
}
