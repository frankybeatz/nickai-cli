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
// isConfigured indicates whether the user has an API key set.
func RenderWelcome(width int, isConfigured bool) string {
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

	version := DimStyle.Render("v" + Version)

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

	punchlines := []string{
		"No suit required.",
		"Wall Street hates this one terminal.",
		"Pants optional, alpha required.",
		"Hedge fund energy, hoodie dress code.",
		"Faster than your broker's lunch break.",
		"Your Bloomberg at home, in a terminal.",
		"Because alt-tabbing to Binance is so 2024.",
		"While your broker is still on hold.",
		"Less Bloomberg, more brrr.",
		"Suits not included. Returns may vary.",
	}
	bodyText := "AI-powered trading terminal. Build agents, execute trades,\nand manage your portfolio — " +
		punchlines[rand.Intn(len(punchlines))]
	body := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA")).
		Render(bodyText)

	var ctaLine, helpHint, tip string

	num := func(n string) string {
		return lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(n)
	}

	if !isConfigured {
		// First-time user: step-by-step guide.
		ctaLine = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Bold(true).
			Render("Quick start:")

		helpHint = num("1. ") + CommandStyle.Render("/config init") +
			DimStyle.Render("            — create paper trading account") +
			"\n" + num("2. ") + CommandStyle.Render("/config set anthropic_key") +
			DimStyle.Render(" — connect your AI") +
			"\n" + num("3. ") + CommandStyle.Render("/mcp quick") +
			DimStyle.Render("              — install free market data servers") +
			"\n" + num("4. ") + DimStyle.Render("Ask me anything — ") +
			lipgloss.NewStyle().Foreground(ColorWhite).Render("\"buy 0.5 ETH\"") +
			DimStyle.Render(", ") +
			lipgloss.NewStyle().Foreground(ColorWhite).Render("\"show my portfolio\"")

		tip = lipgloss.NewStyle().Foreground(ColorPrimary).Render("● ") +
			DimStyle.Render("$100k fake money to experiment. Zero risk.")
	} else {
		// Returning user: normal CTA + random tip.
		ctaLine = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Render("Tell me what you want to build, or try a command.")

		helpHint = DimStyle.Render("Type ") +
			CommandStyle.Render("/help") +
			DimStyle.Render(" for commands, or ") +
			CommandStyle.Render("F1") +
			DimStyle.Render(" for keyboard shortcuts.")

		tips := []string{
			"Tab completes commands and symbols",
			"Press Esc for vim NORMAL mode, j/k to scroll",
			"/chart BTC shows a sparkline chart",
			"/snapshot shows a combined dashboard",
			"/alert BTC > 100000 sets a background alert",
			"/mcp search to browse trading integrations",
			"/model to switch between AI providers",
			"/theme to change the color scheme",
			"Up/Down arrow cycles through command history",
			"/man <command> shows detailed manual pages",
			"/backtest presets to see ready-made strategies",
			"/backtest run rsi-reversal BTC to run a preset",
			"Ask Nick to backtest any strategy in plain English",
			"/guide to learn the ropes interactively",
			"/polymarket scan for prediction market analysis",
		}
		tip = lipgloss.NewStyle().Foreground(ColorPrimary).Render("● ") +
			DimStyle.Render("Tip: "+tips[rand.Intn(len(tips))])
	}

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
	inner = append(inner, tip)
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
