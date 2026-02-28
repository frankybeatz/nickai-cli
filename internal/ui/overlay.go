package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/ai"
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
// title is shown in the top border. content is the inner body.
// width and height are the outer dialog dimensions.
// screenW and screenH are the full terminal dimensions for centering.
func overlayFrame(title, content string, width, height, screenW, screenH int) string {
	titleRendered := ""
	if title != "" {
		titleRendered = lipgloss.NewStyle().
			Foreground(ColorPrimary).Bold(true).
			Render(" " + title + " ")
	}

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

	// Inject title into top border (replace center section).
	if titleRendered != "" {
		lines := strings.Split(rendered, "\n")
		if len(lines) > 0 {
			border := lines[0]
			titleWidth := lipgloss.Width(titleRendered)
			borderWidth := lipgloss.Width(border)
			if titleWidth+4 < borderWidth {
				// Place title 2 chars from the left edge of the border.
				insertAt := 3
				leftBorder := string([]rune(border)[:insertAt])
				rightStart := insertAt + titleWidth
				if rightStart < len([]rune(border)) {
					rightBorder := string([]rune(border)[rightStart:])
					lines[0] = leftBorder + titleRendered + rightBorder
				}
			}
		}
		rendered = strings.Join(lines, "\n")
	}

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
	col3Header := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("Setup & Tools")

	dim := DimStyle.Render
	key := func(k string) string {
		return lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(k)
	}

	col1 := []string{
		col1Header,
		"",
		key("Esc") + dim("     Normal mode"),
		key("i/a/o") + dim("   Insert mode"),
		key(":") + dim("       Command mode"),
		key("/") + dim("       Search mode"),
		key("j/k") + dim("     Scroll up/down"),
		key("d/u") + dim("     Half page ↓/↑"),
		key("gg/G") + dim("    Top / bottom"),
		key("Tab") + dim("     Complete cmd"),
		key("↑/↓") + dim("     History"),
		key("q") + dim("       Quit"),
	}

	col2 := []string{
		col2Header,
		"",
		key("/buy") + dim("       Market buy"),
		key("/sell") + dim("      Limit sell"),
		key("/price") + dim("     Price quotes"),
		key("/watch") + dim("     Live dashboard"),
		key("/chart") + dim("     Sparkline chart"),
		key("/alert") + dim("     Price alerts"),
		key("/status") + dim("    Portfolio"),
		key("/orders") + dim("    Recent trades"),
		key("/pnl") + dim("       Profit & loss"),
		key("/snapshot") + dim("  Full dashboard"),
	}

	col3 := []string{
		col3Header,
		"",
		key("/config") + dim("      Settings & API keys"),
		key("/mcp") + dim("         MCP integrations"),
		key("/credential") + dim("  Exchange API keys"),
		key("/workflow") + dim("    Automations"),
		key("/trigger") + dim("     Conditional trades"),
		key("/model") + dim("       Switch LLM"),
		"",
		key("Ctrl+K") + dim("  Command palette"),
		key("Ctrl+T") + dim("  Theme picker"),
		key("Ctrl+O") + dim("  Model selector"),
		key("/help") + dim("       All commands"),
	}

	colWidth := 28
	maxRows := max(len(col1), max(len(col2), len(col3)))
	var rows []string
	for i := 0; i < maxRows; i++ {
		c1, c2, c3 := "", "", ""
		if i < len(col1) {
			c1 = col1[i]
		}
		if i < len(col2) {
			c2 = col2[i]
		}
		if i < len(col3) {
			c3 = col3[i]
		}

		c1Styled := lipgloss.NewStyle().Width(colWidth).Render(c1)
		c2Styled := lipgloss.NewStyle().Width(colWidth).Render(c2)
		c3Styled := lipgloss.NewStyle().Width(colWidth).Render(c3)

		rows = append(rows, c1Styled+"  "+c2Styled+"  "+c3Styled)
	}

	footer := "\n" + DimStyle.Render("Press any key to close")

	content := strings.Join(rows, "\n") + footer

	dialogW := min(screenW-4, 92)
	dialogH := maxRows + 6
	return overlayFrame("Help", content, dialogW, dialogH, screenW, screenH)
}

// ── Theme Picker Dialog ──

func renderThemeDialog(cursor int, screenW, screenH int) string {
	// Get sorted theme names.
	names := sortedThemeNames()

	var rows []string
	for i, name := range names {
		t := Themes[name]
		prefix := "  "
		if i == cursor {
			prefix = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("▸ ")
		}

		// Show theme name with its own primary color swatch.
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

	footer := "\n" + DimStyle.Render("↑/↓ navigate  Enter select  Esc cancel")

	content := strings.Join(rows, "\n") + footer

	dialogW := min(screenW-4, 44)
	dialogH := len(names) + 6
	return overlayFrame("Theme", content, dialogW, dialogH, screenW, screenH)
}

// ── Model Picker Dialog ──

func renderModelDialog(cursor int, agent *ai.Agent, screenW, screenH int) string {
	models := ai.AvailableModels

	var rows []string
	for i, m := range models {
		prefix := "  "
		if i == cursor {
			prefix = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("▸ ")
		}

		nameStyle := lipgloss.NewStyle().Foreground(ColorWhite)
		if i == cursor {
			nameStyle = nameStyle.Bold(true)
		}

		// Current model indicator.
		current := ""
		if agent != nil && agent.ModelID() == m.ID {
			current = lipgloss.NewStyle().Foreground(ColorPrimary).Render(" ●")
		}

		freeTag := ""
		if m.Free {
			freeTag = lipgloss.NewStyle().Foreground(ColorPrimary).Render(" [free]")
		}

		row := prefix + nameStyle.Render(fmt.Sprintf("%-16s", m.ID)) +
			DimStyle.Render(string(m.Provider)) + freeTag + current
		rows = append(rows, row)
	}

	footer := "\n" + DimStyle.Render("↑/↓ navigate  Enter select  Esc cancel")

	content := strings.Join(rows, "\n") + footer

	dialogW := min(screenW-4, 48)
	dialogH := len(models) + 6
	return overlayFrame("Model", content, dialogW, dialogH, screenW, screenH)
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
	rows = append(rows, DimStyle.Render(strings.Repeat("─", 36)))

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

	footer := "\n" + DimStyle.Render("↑/↓ navigate  Enter run  Esc cancel")
	content := strings.Join(rows, "\n") + footer

	dialogW := min(screenW-4, 52)
	dialogH := visibleCount + 7
	if len(filtered) > maxVisible {
		dialogH++ // extra line for scroll hint
	}
	return overlayFrame("Commands", content, dialogW, dialogH, screenW, screenH)
}

// paletteCommands is the full list for the command palette: "cmd|description".
var paletteCommands = []string{
	"/help|Show all commands",
	"/status|Portfolio & positions",
	"/orders|Recent orders",
	"/price|Live price quotes",
	"/watch|Live price dashboard",
	"/snapshot|Combined dashboard",
	"/market|Full market overview",
	"/pnl|Profit & loss summary",
	"/history|Trade journal",
	"/buy|Market buy order",
	"/sell|Limit/market sell",
	"/agents|List trading agents",
	"/templates|Browse marketplace",
	"/workflow|Manage workflows",
	"/credential|Manage API keys",
	"/logs|Workflow logs",
	"/chart|ASCII sparkline chart",
	"/alert|Set price alerts",
	"/model|Switch AI model",
	"/theme|Switch color theme",
	"/mcp list|Connected MCP servers & tools",
	"/mcp search|Browse MCP server directory",
	"/mcp add|Install an MCP server",
	"/mcp info|Details on an MCP server",
	"/mcp remove|Disconnect an MCP server",
	"/mcp quick|Install all free MCP servers",
	"/trigger list|View active triggers",
	"/trigger add|Create conditional trade trigger",
	"/trigger clear|Remove all triggers",
	"/config|Manage settings",
	"/config init|Create account & API key",
	"/man|Manual pages",
	"/clear|Clear chat",
	"/quit|Exit NickAI",
}

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
