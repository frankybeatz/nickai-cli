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
func RenderWelcome(width int, stage guidance.Stage, actions []guidance.ActionCard, vibeID string, ctx guidance.StageContext, memCount ...int) string {
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

	stageOrdinal := guidance.StageOrdinal(stage)
	stageLabel := guidance.StageLabel(stage)
	journey := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true).
		Render(fmt.Sprintf("  Journey %d/7 · %s", stageOrdinal, stageLabel))
	inner = append(inner, journey)
	inner = append(inner, "  "+renderProgressBar(stageOrdinal, 7, 26))
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

	done, total := guidance.JourneyProgress(ctx)
	checklist := guidance.OnboardingChecklist(ctx)
	inner = append(inner, renderMissionBlock(stage, checklist, done, total)...)

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

func renderProgressBar(current int, total int, width int) string {
	if total <= 0 {
		total = 1
	}
	if width < total {
		width = total
	}
	filled := current * width / total
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("=", filled) + strings.Repeat("-", width-filled) + "]"
}

func renderMissionBlock(stage guidance.Stage, checklist []guidance.ChecklistItem, done int, total int) []string {
	var lines []string
	if total > 0 {
		lines = append(lines, DimStyle.Render(fmt.Sprintf("  Mission progress: %d/%d complete", done, total)))
		lines = append(lines, "")
	}

	var pending []guidance.ChecklistItem
	for _, item := range checklist {
		if !item.Done {
			pending = append(pending, item)
		}
	}
	if len(pending) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("  You're fully equipped."))
		lines = append(lines, CommandStyle.Render("  Try: scan market opportunities and propose 2 setups with risk limits"))
		lines = append(lines, "")
		return lines
	}

	stepsToShow := min(3, len(pending))
	title := "  5-minute mission:"
	if stage == guidance.StageAdvanced {
		title = "  Next mission:"
	}
	lines = append(lines, lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(title))
	for i := range stepsToShow {
		step := pending[i]
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(fmt.Sprintf("  %d) %s", i+1, step.Label)))
		lines = append(lines, CommandStyle.Render("     "+step.Command))
	}
	lines = append(lines, "")

	lines = append(lines, DimStyle.Render("  Talk naturally with Nick:"))
	lines = append(lines, CommandStyle.Render("  \"Nick, scan BTC/ETH/SOL and give me one setup with clear risk.\""))
	lines = append(lines, CommandStyle.Render("  \"Guide me step by step like I'm new and keep me disciplined.\""))
	lines = append(lines, "")
	return lines
}
