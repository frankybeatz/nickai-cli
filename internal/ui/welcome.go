package ui

import (
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var startupTaglines = []string{
	"Your agents don't sleep. Neither should your portfolio.",
	"Multi-LLM consensus. Zero human emotion.",
	"Built different. Trades different.",
	"The terminal is the trading floor.",
	"Ctrl+C is the only stop-loss you need.",
	"While you sleep, your agents compound.",
	"Autonomous alpha, one prompt at a time.",
	"Less panic selling. More autonomous building.",
	"Your portfolio called. It wants an upgrade.",
	"Agents go brrr.",
	"Wall Street runs on coffee. NickAI runs on tokens.",
	"Deploy an agent. Touch grass. Check back later.",
	"No Bloomberg terminal needed. Just vibes and workflows.",
	"Sentiment analysis at 3am so you don't have to be.",
	"The only leverage here is AI leverage.",
	"From /buy to Lambo. Results may vary.",
}

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

	version := DimStyle.Render("v0.3.0")

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
	quip := DimStyle.Render("\"" + startupTaglines[rand.Intn(len(startupTaglines))] + "\"")
	inner = append(inner, quip)
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
