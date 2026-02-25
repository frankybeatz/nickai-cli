package ui

import "github.com/charmbracelet/lipgloss"

// NickAI brand colors.
var (
	ColorPrimary   = lipgloss.Color("#00D4AA")
	ColorSecondary = lipgloss.Color("#6C63FF")
	ColorDim       = lipgloss.Color("#666666")
	ColorWhite     = lipgloss.Color("#FFFFFF")
	ColorError     = lipgloss.Color("#FF6B6B")
	ColorWarning   = lipgloss.Color("#FFD93D")
	ColorBg        = lipgloss.Color("#1A1A2E")
	ColorCardBg    = lipgloss.Color("#16213E")
)

// Text styles.
var (
	BrandStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	SecondaryStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	UserMsgStyle = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Bold(true)

	BotMsgStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	CommandStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)
)

// Card renders a bordered card at the given width.
func Card(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(0, 1).
		Width(width)
}

// AgentCard renders a card for agent display.
func AgentCard(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Width(width)
}

// StatusIndicator returns a styled status dot.
func StatusIndicator(status string) string {
	switch status {
	case "running":
		return lipgloss.NewStyle().Foreground(ColorPrimary).Render("● ")
	case "stopped":
		return lipgloss.NewStyle().Foreground(ColorDim).Render("○ ")
	case "error":
		return lipgloss.NewStyle().Foreground(ColorError).Render("✖ ")
	default:
		return lipgloss.NewStyle().Foreground(ColorDim).Render("? ")
	}
}

// InputPrompt is the styled prompt prefix.
var InputPrompt = lipgloss.NewStyle().
	Foreground(ColorPrimary).
	Bold(true).
	Render("nick → ")
