package ui

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

// renderMarkdown renders markdown content using glamour for terminal display.
func renderMarkdown(content string, width int) string {
	if content == "" {
		return content
	}
	if width < 20 {
		width = 20
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
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
