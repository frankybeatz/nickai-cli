package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/guidance"
	"github.com/nickai/cli/internal/personality"
)

// sessionSeed is a stable random value picked once at startup to prevent flicker.
var sessionSeed = time.Now().UnixNano()

// sessionTagline is picked once at startup to prevent flicker on re-renders.
var sessionTagline = startupTaglines[sessionSeed%int64(len(startupTaglines))]

var startupTaglines = []string{
	"Your agents don't sleep. Neither should your portfolio.",
	"10 AI models. One consensus. Zero emotion.",
	"Built different. Trades different.",
	"The terminal is the trading floor.",
	"While you sleep, your agents compound.",
	"Deploy an agent. Touch grass. Check back later.",
	"Ape responsibly.",
	"Not financial advice. Definitely financial vibes.",
	"Your broker could never.",
	"Ctrl+C is the only stop-loss you need.",
	"WAGMI (with proper risk management).",
	"Less panic selling. More autonomous building.",
	"From /buy to Lambo. Results may vary.",
	"Your strategies backtest themselves. You just vibe.",
	"The degen terminal for disciplined degens.",
	"ser, the chart is talking to me.",
	"Fueled by caffeine and RSI divergences.",
}

// RenderWelcome returns the NickAI branded welcome screen.
// Now uses the guidance system for dynamic action cards.
func RenderWelcome(width int, stage guidance.Stage, actions []guidance.ActionCard, vibeID string, memCount ...int) string {
	memoryCount := 0
	mcpCount := 0
	if len(memCount) > 0 {
		memoryCount = memCount[0]
	}
	if len(memCount) > 1 {
		mcpCount = memCount[1]
	}

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
		Render("Your degen co-pilot with discipline")

	version := DimStyle.Render("v" + Version)

	// Decorative divider.
	pulseUnit := "─ · "
	repeats := (boxInner - 4) / len(pulseUnit)
	if repeats < 1 {
		repeats = 1
	}
	divider := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Render(strings.Repeat(pulseUnit, repeats))

	var inner []string
	inner = append(inner, "")
	inner = append(inner, styledLogo)
	inner = append(inner, "")
	inner = append(inner, tagline+"   "+version)
	quip := DimStyle.Render("\"" + sessionTagline + "\"")
	inner = append(inner, quip)
	inner = append(inner, "")
	inner = append(inner, divider)
	inner = append(inner, "")

	// Dynamic content based on stage.
	if stage == guidance.StageFresh {
		greeting := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render("gm. Let's get you set up.")
		inner = append(inner, greeting)
		inner = append(inner, "")
	} else {
		vibe := personality.GetVibe(vibeID)
		greetings := vibe.Greetings
		greeting := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(greetings[sessionSeed%int64(len(greetings))])
		inner = append(inner, greeting)
		inner = append(inner, "")
	}

	// Action cards.
	if len(actions) > 0 {
		header := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("  What's next:")
		inner = append(inner, header)
		inner = append(inner, "")
		for _, card := range actions {
			title := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).
				Render("  ▸ " + card.Title)
			desc := DimStyle.Render("    " + card.Desc)
			cmd := CommandStyle.Render("    " + card.Command)
			inner = append(inner, title)
			inner = append(inner, desc)
			inner = append(inner, cmd)
			inner = append(inner, "")
		}
	}

	// Context indicators for returning users.
	if memoryCount > 0 || mcpCount > 0 {
		var ctx []string
		if memoryCount > 0 {
			ctx = append(ctx, fmt.Sprintf("%d memories loaded", memoryCount))
		}
		if mcpCount > 0 {
			ctx = append(ctx, fmt.Sprintf("%d MCP servers", mcpCount))
		}
		inner = append(inner, DimStyle.Render("  "+strings.Join(ctx, "  ·  ")))
		inner = append(inner, "")
	}

	// Keyboard hint.
	inner = append(inner, DimStyle.Render("  ")+
		CommandStyle.Render("/help")+
		DimStyle.Render(" commands  ·  ")+
		CommandStyle.Render("F1")+
		DimStyle.Render(" shortcuts  ·  ")+
		CommandStyle.Render("Esc")+
		DimStyle.Render(" vim mode"))
	inner = append(inner, "")

	content := strings.Join(inner, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Width(boxInner + 4).
		Render(content)

	centered := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(box)

	return "\n" + centered + "\n"
}
