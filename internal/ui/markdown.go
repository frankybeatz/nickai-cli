package ui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/muesli/termenv"
)

// nickStyle returns a glamour dark style with no background colors on headings
// so rendered text blends cleanly with the TUI. Using a fixed dark profile
// avoids OSC terminal queries (which leak escape sequences into Bubbletea's
// input buffer and cause inconsistent rendering).
func nickStyle() glamour.TermRendererOption {
	s := styles.DarkStyleConfig
	// Remove background color on h1 — it renders as a coloured highlight
	// band that clashes with our card backgrounds.
	s.H1.BackgroundColor = nil
	return glamour.WithStyles(s)
}

// renderMarkdown renders markdown content using glamour for terminal display.
func renderMarkdown(content string, width int) string {
	if content == "" {
		return content
	}
	if width < 20 {
		width = 20
	}

	r, err := glamour.NewTermRenderer(
		nickStyle(),
		glamour.WithWordWrap(width),
		glamour.WithColorProfile(termenv.TrueColor),
	)
	if err != nil {
		return content
	}

	rendered, err := r.Render(content)
	if err != nil {
		return content
	}

	return strings.TrimSpace(rendered)
}
