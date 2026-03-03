package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/api"
)

// --- Shared helpers used across card files ---

// connectPrompt renders the "no API key" message with setup instructions.
func connectPrompt() string {
	return DimStyle.Render("  No API key configured. ") +
		"Run " + CommandStyle.Render("/config init") +
		DimStyle.Render(" to create an account, or ") +
		CommandStyle.Render("/config set api_key <key>") +
		DimStyle.Render(" if you have one.")
}

// NormalizeSymbol delegates to api.NormalizeSymbol.
func NormalizeSymbol(s string) string {
	return api.NormalizeSymbol(s)
}

// maskKeyShort masks all but the first/last few chars of a key.
func maskKeyShort(k string) string {
	if k == "" {
		return "(not set)"
	}
	if len(k) <= 8 {
		return k[:2] + "***"
	}
	return k[:4] + "..." + k[len(k)-4:]
}

// --- Formatting helpers ---

func formatMoney(v float64) string {
	if v >= 1_000_000 || v <= -1_000_000 {
		return fmt.Sprintf("$%.2fM", v/1_000_000)
	}
	return fmt.Sprintf("$%.2f", v)
}

func formatPrice(v float64) string {
	if v >= 1 {
		return fmt.Sprintf("$%.2f", v)
	}
	return fmt.Sprintf("$%.6f", v)
}

func orderStatusToIndicator(s string) string {
	switch s {
	case "filled", "completed":
		return "running"
	case "cancelled", "rejected", "expired":
		return "stopped"
	case "failed":
		return "error"
	default:
		return "running" // pending, open, partial
	}
}

func renderOrderStatus(s string) string {
	switch s {
	case "filled", "completed":
		return BrandStyle.Render(s)
	case "cancelled", "rejected", "expired":
		return DimStyle.Render(s)
	case "failed":
		return ErrorStyle.Render(s)
	default:
		return WarningStyle.Render(s) // pending, open, partial
	}
}

func padRight(s string, n int) string {
	w := lipgloss.Width(s)
	if w >= n {
		return s
	}
	return s + strings.Repeat(" ", n-w)
}

func boolStr(v bool) string {
	if v {
		return BrandStyle.Render("on")
	}
	return DimStyle.Render("off")
}
