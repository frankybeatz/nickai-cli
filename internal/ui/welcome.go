package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderWelcome returns the NickAI branded welcome screen.
func RenderWelcome(width int) string {
	// Constrain the hero box to a comfortable reading width.
	boxInner := min(width-6, 66)
	if boxInner < 40 {
		boxInner = 40
	}

	logo := ` ███╗   ██╗██╗ ██████╗██╗  ██╗ █████╗ ██╗
 ████╗  ██║██║██╔════╝██║ ██╔╝██╔══██╗██║
 ██╔██╗ ██║██║██║     █████╔╝ ███████║██║
 ██║╚██╗██║██║██║     ██╔═██╗ ██╔══██║██║
 ██║ ╚████║██║╚██████╗██║  ██╗██║  ██║██║
 ╚═╝  ╚═══╝╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝`

	styledLogo := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render(logo)

	tagline := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true).
		Render("The agentic OS for autonomous finance")

	version := DimStyle.Render("v0.1.0")

	// Decorative divider — alternating dashes give an "energy pulse" look.
	pulseUnit := "─ · "
	repeats := (boxInner - 4) / len(pulseUnit)
	if repeats < 1 {
		repeats = 1
	}
	divider := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Render(strings.Repeat(pulseUnit, repeats))

	// Welcome copy.
	greeting := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Bold(true).
		Render("Welcome.") +
		" " +
		lipgloss.NewStyle().
			Foreground(ColorWhite).
			Render("I'm your NickAI agent builder.")

	body := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA")).
		Render("I can create trading agents, deploy them, check their\nperformance, and manage your portfolio — all from here.")

	ctaLine := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Render("To get started, tell me what you want to build.")

	helpHint := DimStyle.Render("Or type ") +
		CommandStyle.Render("/help") +
		DimStyle.Render(" for commands.")

	// Assemble inner content.
	var inner []string
	inner = append(inner, "")
	inner = append(inner, styledLogo)
	inner = append(inner, "")
	inner = append(inner, tagline+"   "+version)
	inner = append(inner, "")
	inner = append(inner, divider)
	inner = append(inner, "")
	inner = append(inner, greeting)
	inner = append(inner, body)
	inner = append(inner, "")
	inner = append(inner, ctaLine)
	inner = append(inner, helpHint)
	inner = append(inner, "")

	content := strings.Join(inner, "\n")

	// Outer bordered box in NickAI green.
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Width(boxInner + 4). // padding adds 4 chars
		Render(content)

	// Center the box in the terminal.
	centered := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(box)

	return "\n" + centered + "\n"
}
