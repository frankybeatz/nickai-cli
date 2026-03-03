package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/ai"
	"github.com/nickai/cli/internal/commands"
)

// DialogType identifies which overlay dialog is active.
type DialogType int

const (
	DialogNone    DialogType = iota
	DialogHelp               // ? or Ctrl+?
	DialogPalette            // Ctrl+K
	DialogTheme              // Ctrl+T
	DialogModel              // Ctrl+O
)

// DialogState holds the interactive state for overlay dialogs.
type DialogState struct {
	Active       DialogType
	Cursor       int
	ScrollOffset int      // first visible index in scrollable lists
	Filter       string
	FilteredList []string // cached filtered items for palette
}

// overlayFrame renders a floating dialog frame with a drop shadow.
// content is the inner body. width and height are the outer dialog dimensions.
// screenW and screenH are the full terminal dimensions for centering.
func overlayFrame(content string, width, height, screenW, screenH int) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(1, 2).
		Width(width).
		Height(height).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true)

	rendered := box.Render(content)

	// Drop shadow using ░ character.
	shadowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
	renderedLines := strings.Split(rendered, "\n")

	var shadowed []string
	for _, line := range renderedLines {
		shadowed = append(shadowed, line+shadowStyle.Render("░░"))
	}
	// Bottom shadow.
	boxWidth := lipgloss.Width(renderedLines[0])
	shadowed = append(shadowed, strings.Repeat(" ", 2)+shadowStyle.Render(strings.Repeat("░", boxWidth)))
	rendered = strings.Join(shadowed, "\n")

	// Center horizontally and vertically.
	dialogHeight := len(shadowed)
	topPad := (screenH - dialogHeight) / 2
	if topPad < 1 {
		topPad = 1
	}

	centered := lipgloss.NewStyle().
		Width(screenW).
		Align(lipgloss.Center).
		PaddingTop(topPad).
		Render(rendered)

	return centered
}

// compositeOverlay takes the base terminal output and overlays a dialog on top.
// It replaces lines in the base output with dialog lines.
func compositeOverlay(base string, dialog string, screenW, screenH int) string {
	baseLines := strings.Split(base, "\n")
	dialogLines := strings.Split(dialog, "\n")

	// Ensure base has enough lines.
	for len(baseLines) < screenH {
		baseLines = append(baseLines, "")
	}

	// Overlay dialog lines onto base.
	for i, dLine := range dialogLines {
		if i >= len(baseLines) {
			break
		}
		if strings.TrimSpace(dLine) != "" {
			baseLines[i] = dLine
		}
	}

	// Trim to screen height.
	if len(baseLines) > screenH {
		baseLines = baseLines[:screenH]
	}

	return strings.Join(baseLines, "\n")
}

// ── Help Dialog ──

func renderHelpDialog(screenW, screenH int) string {
	col1Header := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("Navigation")
	col2Header := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("Trading")
	col3Header := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("Tools & AI")
	col4Header := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("Multi-Vertical")

	dim := DimStyle.Render
	key := func(k string) string {
		return lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(k)
	}

	col1 := []string{
		col1Header,
		"",
		key("Esc") + dim("    Normal mode"),
		key("i/a/o") + dim("  Insert mode"),
		key(":") + dim("      Command mode"),
		key("/") + dim("      Search mode"),
		key("j/k") + dim("    Scroll ↑/↓"),
		key("d/u") + dim("    Half page"),
		key("gg/G") + dim("   Top / bottom"),
		key("Tab") + dim("    Autocomplete"),
		key("↑/↓") + dim("    History"),
		key("Ctrl+K") + dim(" Palette"),
		key("Ctrl+T") + dim(" Theme picker"),
		key("Ctrl+O") + dim(" Model picker"),
		key("F1") + dim("     This help"),
	}

	col2 := []string{
		col2Header,
		"",
		key("/buy") + dim("      Market buy"),
		key("/sell") + dim("     Market sell"),
		key("/price") + dim("    Quotes"),
		key("/watch") + dim("    Live ticker"),
		key("/chart") + dim("    Sparkline"),
		key("/alert") + dim("    Price alert"),
		key("/status") + dim("   Portfolio"),
		key("/orders") + dim("   Recent trades"),
		key("/pnl") + dim("      P&L summary"),
		key("/snapshot") + dim(" Dashboard"),
		key("/analyze") + dim("  Analysis"),
		key("/backtest") + dim(" Backtest"),
	}

	col3 := []string{
		col3Header,
		"",
		key("/memory") + dim("    Memories"),
		key("/consensus") + dim(" Consensus"),
		key("/polymarket") + dim("Predictions"),
		key("/analytics") + dim(" Stats"),
		key("/risk") + dim("      Guardrails"),
		key("/auto") + dim("      Automation"),
		key("/trigger") + dim("   Triggers"),
		key("/strategy") + dim("  Strategies"),
		key("/config") + dim("    Settings"),
		key("/mcp") + dim("       MCP servers"),
		key("/guide") + dim("     Guide"),
		key("/man") + dim("       Manual"),
	}

	col4 := []string{
		col4Header,
		"",
		key("/connect") + dim("  Exchanges"),
		key("/balances") + dim(" Balances"),
		key("/positions") + dim("Positions"),
		key("/markets") + dim("  Markets"),
		key("/stock") + dim("    Equities"),
		key("/screen") + dim("   Screener"),
		key("/wallet") + dim("   Onchain"),
		key("/swap") + dim("     DEX swap"),
		key("/gas") + dim("      Gas prices"),
		key("/odds") + dim("     Odds"),
		key("/lines") + dim("    Lines"),
		key("/funding") + dim("  Funding"),
	}

	colWidth := 23
	maxRows := max(len(col1), max(len(col2), max(len(col3), len(col4))))
	var rows []string
	for i := 0; i < maxRows; i++ {
		c1, c2, c3, c4 := "", "", "", ""
		if i < len(col1) {
			c1 = col1[i]
		}
		if i < len(col2) {
			c2 = col2[i]
		}
		if i < len(col3) {
			c3 = col3[i]
		}
		if i < len(col4) {
			c4 = col4[i]
		}

		c1s := lipgloss.NewStyle().Width(colWidth).MaxWidth(colWidth).Render(c1)
		c2s := lipgloss.NewStyle().Width(colWidth).MaxWidth(colWidth).Render(c2)
		c3s := lipgloss.NewStyle().Width(colWidth).MaxWidth(colWidth).Render(c3)
		c4s := lipgloss.NewStyle().Width(colWidth).MaxWidth(colWidth).Render(c4)

		rows = append(rows, c1s+" "+c2s+" "+c3s+" "+c4s)
	}

	title := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("Help") + "\n"
	footer := "\n" + DimStyle.Render("Press any key to close")

	content := title + strings.Join(rows, "\n") + footer

	dialogW := min(screenW-4, 100)
	dialogH := maxRows + 7
	return overlayFrame(content, dialogW, dialogH, screenW, screenH)
}

// ── Theme Picker Dialog ──

const maxVisibleDialogItems = 12

func renderThemeDialog(cursor, scrollOffset int, screenW, screenH int) string {
	names := sortedThemeNames()

	endIdx := scrollOffset + maxVisibleDialogItems
	if endIdx > len(names) {
		endIdx = len(names)
	}

	var rows []string
	for i := scrollOffset; i < endIdx; i++ {
		name := names[i]
		t := Themes[name]
		prefix := "  "
		if i == cursor {
			prefix = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("▸ ")
		}

		swatch := lipgloss.NewStyle().Foreground(t.Primary).Render("██")
		swatch2 := lipgloss.NewStyle().Foreground(t.Secondary).Render("██")
		swatch3 := lipgloss.NewStyle().Foreground(t.Error).Render("██")

		nameStyle := lipgloss.NewStyle().Foreground(ColorWhite)
		if i == cursor {
			nameStyle = nameStyle.Bold(true)
		}

		row := prefix + nameStyle.Render(fmt.Sprintf("%-14s", name)) + " " + swatch + swatch2 + swatch3
		rows = append(rows, row)
	}

	if len(names) > maxVisibleDialogItems {
		remaining := len(names) - endIdx
		above := scrollOffset
		var scrollHint string
		if above > 0 && remaining > 0 {
			scrollHint = fmt.Sprintf("↑ %d above  ↓ %d below", above, remaining)
		} else if remaining > 0 {
			scrollHint = fmt.Sprintf("↓ %d more", remaining)
		} else if above > 0 {
			scrollHint = fmt.Sprintf("↑ %d above", above)
		}
		if scrollHint != "" {
			rows = append(rows, DimStyle.Render("  "+scrollHint))
		}
	}

	footer := "\n" + DimStyle.Render("↑/↓ navigate  Enter select  Esc cancel")

	title := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("Theme") + "\n"
	content := title + strings.Join(rows, "\n") + footer

	visibleCount := endIdx - scrollOffset
	dialogW := min(screenW-4, 44)
	dialogH := visibleCount + 7
	if len(names) > maxVisibleDialogItems {
		dialogH++
	}
	return overlayFrame(content, dialogW, dialogH, screenW, screenH)
}

// ── Model Picker Dialog ──

func renderModelDialog(cursor, scrollOffset int, agent *ai.Agent, screenW, screenH int) string {
	models := ai.AvailableModels

	endIdx := scrollOffset + maxVisibleDialogItems
	if endIdx > len(models) {
		endIdx = len(models)
	}

	var rows []string
	for i := scrollOffset; i < endIdx; i++ {
		m := models[i]
		prefix := "  "
		if i == cursor {
			prefix = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("▸ ")
		}

		nameStyle := lipgloss.NewStyle().Foreground(ColorWhite)
		if i == cursor {
			nameStyle = nameStyle.Bold(true)
		}

		current := ""
		if agent != nil && agent.ModelID() == m.ID {
			current = lipgloss.NewStyle().Foreground(ColorPrimary).Render(" ●")
		}

		freeTag := ""
		if m.Free {
			freeTag = lipgloss.NewStyle().Foreground(ColorPrimary).Render(" [free]")
		}

		row := prefix + nameStyle.Render(fmt.Sprintf("%-18s", m.ID)) +
			DimStyle.Render(fmt.Sprintf("%-12s", string(m.Provider))) + freeTag + current
		rows = append(rows, row)
	}

	if len(models) > maxVisibleDialogItems {
		remaining := len(models) - endIdx
		above := scrollOffset
		var scrollHint string
		if above > 0 && remaining > 0 {
			scrollHint = fmt.Sprintf("↑ %d above  ↓ %d below", above, remaining)
		} else if remaining > 0 {
			scrollHint = fmt.Sprintf("↓ %d more", remaining)
		} else if above > 0 {
			scrollHint = fmt.Sprintf("↑ %d above", above)
		}
		if scrollHint != "" {
			rows = append(rows, DimStyle.Render("  "+scrollHint))
		}
	}

	footer := "\n" + DimStyle.Render("↑/↓ navigate  Enter select  Esc cancel")

	title := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("Model") + "\n"
	content := title + strings.Join(rows, "\n") + footer

	visibleCount := endIdx - scrollOffset
	dialogW := min(screenW-4, 58)
	dialogH := visibleCount + 7
	if len(models) > maxVisibleDialogItems {
		dialogH++
	}
	return overlayFrame(content, dialogW, dialogH, screenW, screenH)
}

// ── Command Palette Dialog ──

func renderPaletteDialog(cursor, scrollOffset int, filter string, filtered []string, screenW, screenH int) string {
	// Search input.
	searchIcon := lipgloss.NewStyle().Foreground(ColorSecondary).Render("⌕ ")
	filterDisplay := filter
	if filterDisplay == "" {
		filterDisplay = DimStyle.Render("Type to filter...")
	} else {
		filterDisplay = lipgloss.NewStyle().Foreground(ColorWhite).Render(filterDisplay)
	}
	cursorChar := lipgloss.NewStyle().Foreground(ColorWhite).Render("█")
	searchLine := searchIcon + filterDisplay + cursorChar

	var rows []string
	rows = append(rows, searchLine)
	rows = append(rows, Divider(36))

	maxVisible := 12
	endIdx := scrollOffset + maxVisible
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}

	for i := scrollOffset; i < endIdx; i++ {
		entry := filtered[i]
		prefix := "  "
		if i == cursor {
			prefix = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("▸ ")
		}

		parts := strings.SplitN(entry, "|", 2)
		cmd := parts[0]
		desc := ""
		if len(parts) > 1 {
			desc = parts[1]
		}

		cmdStyle := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
		if i == cursor {
			cmdStyle = cmdStyle.Foreground(ColorPrimary)
		}
		row := prefix + cmdStyle.Render(fmt.Sprintf("%-16s", cmd)) + DimStyle.Render(desc)
		rows = append(rows, row)
	}

	if len(filtered) == 0 {
		rows = append(rows, DimStyle.Render("  No matching commands"))
	}

	// Show count indicator when there are more items.
	visibleCount := endIdx - scrollOffset
	if len(filtered) > maxVisible {
		remaining := len(filtered) - endIdx
		above := scrollOffset
		var scrollHint string
		if above > 0 && remaining > 0 {
			scrollHint = fmt.Sprintf("↑ %d above  ↓ %d below", above, remaining)
		} else if remaining > 0 {
			scrollHint = fmt.Sprintf("↓ %d more", remaining)
		} else if above > 0 {
			scrollHint = fmt.Sprintf("↑ %d above", above)
		}
		rows = append(rows, DimStyle.Render("  "+scrollHint))
	}

	title := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("Commands") + "\n"
	footer := "\n" + DimStyle.Render("↑/↓ navigate  Enter run  Esc cancel")
	content := title + strings.Join(rows, "\n") + footer

	dialogW := min(screenW-4, 52)
	dialogH := visibleCount + 8
	if len(filtered) > maxVisible {
		dialogH++ // extra line for scroll hint
	}
	return overlayFrame(content, dialogW, dialogH, screenW, screenH)
}

// paletteCommands is the full list for the command palette: "cmd|description".
// paletteCommands is generated from the unified command registry.
var paletteCommands = commands.PaletteEntries()

// filterSuggestions returns palette commands matching a prefix (for the / menu).
func filterSuggestions(prefix string) []string {
	prefix = strings.ToLower(prefix)
	var results []string
	for _, entry := range paletteCommands {
		cmd := strings.SplitN(entry, "|", 2)[0]
		if strings.HasPrefix(strings.ToLower(cmd), prefix) {
			results = append(results, entry)
		}
	}
	return results
}

// renderSuggestionsBox renders a compact suggestion menu for the / command picker.
func renderSuggestionsBox(candidates []string, cursor, scrollOffset, boxWidth int) string {
	maxVisible := 10
	endIdx := scrollOffset + maxVisible
	if endIdx > len(candidates) {
		endIdx = len(candidates)
	}

	var rows []string
	for i := scrollOffset; i < endIdx; i++ {
		entry := candidates[i]
		prefix := "  "
		if i == cursor {
			prefix = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("▸ ")
		}

		parts := strings.SplitN(entry, "|", 2)
		cmd := parts[0]
		desc := ""
		if len(parts) > 1 {
			desc = parts[1]
		}

		cmdStyle := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
		if i == cursor {
			cmdStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
		}
		row := prefix + cmdStyle.Render(padRight(cmd, 16)) + DimStyle.Render(desc)
		rows = append(rows, row)
	}

	if len(candidates) > maxVisible {
		remaining := len(candidates) - endIdx
		above := scrollOffset
		if above > 0 && remaining > 0 {
			rows = append(rows, DimStyle.Render(fmt.Sprintf("  ↑%d  ↓%d more", above, remaining)))
		} else if remaining > 0 {
			rows = append(rows, DimStyle.Render(fmt.Sprintf("  ↓ %d more", remaining)))
		} else if above > 0 {
			rows = append(rows, DimStyle.Render(fmt.Sprintf("  ↑ %d above", above)))
		}
	}

	content := strings.Join(rows, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorDim).
		Padding(0, 1).
		Width(boxWidth).
		Render(content)
}

// filterPaletteCommands filters the command list by a search string.
func filterPaletteCommands(filter string) []string {
	if filter == "" {
		return paletteCommands
	}
	filter = strings.ToLower(filter)
	var results []string
	for _, entry := range paletteCommands {
		if strings.Contains(strings.ToLower(entry), filter) {
			results = append(results, entry)
		}
	}
	return results
}

// sortedThemeNames returns theme names in stable sorted order.
func sortedThemeNames() []string {
	names := make([]string, 0, len(Themes))
	for name := range Themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
