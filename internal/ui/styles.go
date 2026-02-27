package ui

import "github.com/charmbracelet/lipgloss"

// Version is set from main at startup.
var Version = "dev"

// Theme defines a color scheme.
type Theme struct {
	Name      string
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Dim       lipgloss.Color
	White     lipgloss.Color
	Error     lipgloss.Color
	Warning   lipgloss.Color
	Bg        lipgloss.Color
	CardBg    lipgloss.Color
}

// Available themes.
var Themes = map[string]Theme{
	"default": {
		Name: "default", Primary: "#00D4AA", Secondary: "#6C63FF",
		Dim: "#666666", White: "#FFFFFF", Error: "#FF6B6B",
		Warning: "#FFD93D", Bg: "#1A1A2E", CardBg: "#16213E",
	},
	"cyberpunk": {
		Name: "cyberpunk", Primary: "#FF00FF", Secondary: "#00FFFF",
		Dim: "#555555", White: "#FFFFFF", Error: "#FF0055",
		Warning: "#FFE100", Bg: "#0A0A1A", CardBg: "#1A0A2E",
	},
	"bloomberg": {
		Name: "bloomberg", Primary: "#FF8800", Secondary: "#4488FF",
		Dim: "#777777", White: "#FFFFFF", Error: "#FF4444",
		Warning: "#FFAA00", Bg: "#000000", CardBg: "#111111",
	},
	"minimal": {
		Name: "minimal", Primary: "#AAAAAA", Secondary: "#888888",
		Dim: "#555555", White: "#DDDDDD", Error: "#CC6666",
		Warning: "#CCAA66", Bg: "#111111", CardBg: "#1A1A1A",
	},
	"matrix": {
		Name: "matrix", Primary: "#00FF00", Secondary: "#00CC00",
		Dim: "#005500", White: "#00FF00", Error: "#FF0000",
		Warning: "#FFFF00", Bg: "#000000", CardBg: "#001100",
	},
	"tokyonight": {
		Name: "tokyonight", Primary: "#7AA2F7", Secondary: "#BB9AF7",
		Dim: "#565F89", White: "#C0CAF5", Error: "#F7768E",
		Warning: "#E0AF68", Bg: "#1A1B26", CardBg: "#24283B",
	},
	"dracula": {
		Name: "dracula", Primary: "#BD93F9", Secondary: "#FF79C6",
		Dim: "#6272A4", White: "#F8F8F2", Error: "#FF5555",
		Warning: "#F1FA8C", Bg: "#282A36", CardBg: "#44475A",
	},
	"catppuccin": {
		Name: "catppuccin", Primary: "#CBA6F7", Secondary: "#F5C2E7",
		Dim: "#6C7086", White: "#CDD6F4", Error: "#F38BA8",
		Warning: "#F9E2AF", Bg: "#1E1E2E", CardBg: "#313244",
	},
	"nord": {
		Name: "nord", Primary: "#88C0D0", Secondary: "#81A1C1",
		Dim: "#4C566A", White: "#ECEFF4", Error: "#BF616A",
		Warning: "#EBCB8B", Bg: "#2E3440", CardBg: "#3B4252",
	},
	"gruvbox": {
		Name: "gruvbox", Primary: "#B8BB26", Secondary: "#83A598",
		Dim: "#928374", White: "#EBDBB2", Error: "#FB4934",
		Warning: "#FABD2F", Bg: "#282828", CardBg: "#3C3836",
	},
}

// NickAI brand colors (active theme).
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

// ApplyTheme sets the active color variables from a theme.
func ApplyTheme(t Theme) {
	ColorPrimary = t.Primary
	ColorSecondary = t.Secondary
	ColorDim = t.Dim
	ColorWhite = t.White
	ColorError = t.Error
	ColorWarning = t.Warning
	ColorBg = t.Bg
	ColorCardBg = t.CardBg
	rebuildStyles()
}

// Text styles.
var (
	BrandStyle     lipgloss.Style
	SecondaryStyle lipgloss.Style
	DimStyle       lipgloss.Style
	ErrorStyle     lipgloss.Style
	WarningStyle   lipgloss.Style
	UserMsgStyle   lipgloss.Style
	BotMsgStyle    lipgloss.Style
	CommandStyle   lipgloss.Style
)

func init() {
	rebuildStyles()
}

func rebuildStyles() {
	BrandStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	SecondaryStyle = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	DimStyle = lipgloss.NewStyle().Foreground(ColorDim)
	ErrorStyle = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
	WarningStyle = lipgloss.NewStyle().Foreground(ColorWarning)
	UserMsgStyle = lipgloss.NewStyle().Foreground(ColorWhite).Bold(true)
	BotMsgStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
	CommandStyle = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
}

// UserMsgBar renders a message with a colored left border accent (user messages).
func UserMsgBar(content string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(ColorSecondary).
		PaddingLeft(1).
		Render(content)
}

// BotMsgBar renders a message with a colored left border accent (bot messages).
func BotMsgBar(content string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(ColorPrimary).
		PaddingLeft(1).
		Render(content)
}

// SystemMsgBar renders a message with a muted left border (system messages).
func SystemMsgBar(content string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(ColorDim).
		PaddingLeft(1).
		Render(content)
}

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
