package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/automation"
	"github.com/nickai/cli/internal/notify"
	"github.com/nickai/cli/internal/risk"
	"github.com/nickai/cli/internal/strategy"
	"github.com/nickai/cli/internal/trigger"
)

// --- Trigger rendering ---

// RenderTriggerList shows all active triggers in a formatted list.
func RenderTriggerList(triggers []trigger.Trigger) string {
	var lines []string
	lines = append(lines, SectionHeader("Active Triggers"))
	for _, t := range triggers {
		condition := BrandStyle.Render(t.Symbol) + " " + t.Operator + " " + formatPrice(t.Target)
		action := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(
			strings.ToUpper(t.Action.Side) + " " + fmt.Sprintf("%g", t.Action.Quantity) + " " + t.Symbol)
		id := DimStyle.Render("  [" + t.ID[:6] + "]")
		lines = append(lines, "  "+StatusIndicator("running")+
			DimStyle.Render("if ")+condition+DimStyle.Render(" → ")+action+id)
	}
	return strings.Join(lines, "\n")
}

// RenderTriggerConfirm renders a confirmation card for a fired trigger.
func RenderTriggerConfirm(t trigger.Trigger, currentPrice float64) string {
	return WarningStyle.Render("  TRIGGER FIRED ") + "\n" +
		"  " + BrandStyle.Render(t.Symbol) + " hit " + formatPrice(currentPrice) +
		DimStyle.Render(fmt.Sprintf("  (condition: %s %s)", t.Operator, formatPrice(t.Target))) + "\n" +
		"  " + lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(
			strings.ToUpper(t.Action.Side)+" "+fmt.Sprintf("%g", t.Action.Quantity)+" "+t.Symbol+" ("+t.Action.Type+")") + "\n" +
		DimStyle.Render("  Press y to execute, n to skip")
}

// --- Risk rendering ---

// RenderRiskLimits displays current risk limits.
func RenderRiskLimits(limits *risk.RiskLimits) string {
	if limits == nil || limits.IsEmpty() {
		return EmptyState("No risk limits set.") + "\n" +
			DimStyle.Render("  Set with ") + CommandStyle.Render("/risk set max-order 5000")
	}

	var lines []string
	lines = append(lines, SectionHeader("Risk Guardrails"))

	if limits.MaxOrderValue > 0 {
		lines = append(lines, "  "+StatusIndicator("running")+
			DimStyle.Render("Max Order Value:    ")+
			BrandStyle.Render(fmt.Sprintf("$%.0f", limits.MaxOrderValue)))
	}
	if limits.MaxPositionPct > 0 {
		lines = append(lines, "  "+StatusIndicator("running")+
			DimStyle.Render("Max Position:       ")+
			BrandStyle.Render(fmt.Sprintf("%.0f%%", limits.MaxPositionPct)))
	}
	if limits.DailyLossPct > 0 {
		lines = append(lines, "  "+StatusIndicator("running")+
			DimStyle.Render("Daily Loss Limit:   ")+
			BrandStyle.Render(fmt.Sprintf("%.0f%%", limits.DailyLossPct)))
	}
	return strings.Join(lines, "\n")
}

// RenderRiskHelp shows /risk usage.
func RenderRiskHelp() string {
	header := SectionHeader("/risk — portfolio risk guardrails")
	lines := []string{
		header,
		"  " + CommandStyle.Render("/risk show") + DimStyle.Render("                     — view current limits"),
		"  " + CommandStyle.Render("/risk set max-order <$>") + DimStyle.Render("        — max single order value"),
		"  " + CommandStyle.Render("/risk set max-position <%>") + DimStyle.Render("     — max position as % of portfolio"),
		"  " + CommandStyle.Render("/risk set daily-loss <%>") + DimStyle.Render("       — daily loss limit %"),
		"  " + CommandStyle.Render("/risk clear") + DimStyle.Render("                    — remove all limits"),
	}
	return strings.Join(lines, "\n")
}

// --- Strategy rendering ---

// RenderStrategyList shows all TWAP strategies.
func RenderStrategyList(strategies []strategy.TWAPStrategy) string {
	if len(strategies) == 0 {
		return EmptyState("No strategies.") + "\n" +
			DimStyle.Render("  Create one with ") + CommandStyle.Render("/strategy twap ETH buy $2000 4h")
	}

	var lines []string
	lines = append(lines, SectionHeader("TWAP Strategies"))

	for _, s := range strategies {
		statusStr := "running"
		if s.Status != "active" {
			statusStr = "stopped"
		}
		status := StatusIndicator(statusStr)

		progress := fmt.Sprintf("%d/%d", s.Executed, s.SliceCount)
		side := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(strings.ToUpper(s.Side))
		if s.Side == "sell" {
			side = lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render("SELL")
		}

		lines = append(lines, "  "+status+
			side+" "+BrandStyle.Render(s.Symbol)+
			DimStyle.Render(fmt.Sprintf("  $%.0f over %s  [%s]  ", s.TotalValue, s.Duration, progress))+
			renderStrategyStatus(s.Status)+
			DimStyle.Render("  ["+s.ID[:6]+"]"))

		if s.Status == "active" {
			nextStr := s.NextSliceAt.Format("3:04 PM")
			lines = append(lines, "    "+DimStyle.Render("Next slice: ")+nextStr+
				DimStyle.Render(fmt.Sprintf("  ($%.2f/slice)", s.SliceValue)))
		}
	}
	return strings.Join(lines, "\n")
}

func renderStrategyStatus(s string) string {
	switch s {
	case "active":
		return BrandStyle.Render(s)
	case "completed":
		return lipgloss.NewStyle().Foreground(ColorPrimary).Render(s)
	case "cancelled":
		return DimStyle.Render(s)
	default:
		return DimStyle.Render(s)
	}
}

// RenderStrategySliceConfirm renders a confirmation card for a TWAP slice.
func RenderStrategySliceConfirm(s strategy.TWAPStrategy, price float64) string {
	qty := s.SliceValue / price
	side := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(strings.ToUpper(s.Side))
	if s.Side == "sell" {
		side = lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render("SELL")
	}

	return WarningStyle.Render("  TWAP SLICE ") +
		DimStyle.Render(fmt.Sprintf(" (%d/%d)", s.Executed+1, s.SliceCount)) + "\n" +
		"  " + side + " " + BrandStyle.Render(s.Symbol) +
		DimStyle.Render(fmt.Sprintf("  %.4f @ %s ≈ $%.2f", qty, formatPrice(price), s.SliceValue)) + "\n" +
		DimStyle.Render("  Press y to execute, n to skip")
}

// RenderStrategyHelp shows /strategy usage.
func RenderStrategyHelp() string {
	header := SectionHeader("/strategy — TWAP execution strategies")
	lines := []string{
		header,
		"  " + CommandStyle.Render("/strategy twap <SYM> <buy|sell> $<VALUE> <DURATION>") + DimStyle.Render(" — create TWAP"),
		"  " + CommandStyle.Render("/strategy list") + DimStyle.Render("                                      — show strategies"),
		"  " + CommandStyle.Render("/strategy cancel <id>") + DimStyle.Render("                               — cancel strategy"),
		"",
		DimStyle.Render("  Duration examples: 4h, 1h, 30m, 2h30m"),
	}
	return strings.Join(lines, "\n")
}

// --- Notification rendering ---

// RenderNotifyConfig displays the current notification configuration.
func RenderNotifyConfig(cfg *notify.Config) string {
	if cfg == nil || cfg.IsEmpty() {
		return EmptyState("No notification channels configured.") + "\n" +
			DimStyle.Render("  Set with ") + CommandStyle.Render("/notify set desktop on")
	}

	var lines []string
	lines = append(lines, SectionHeader("Notification Settings"))

	indicator := func(enabled bool) string {
		if enabled {
			return StatusIndicator("running")
		}
		return StatusIndicator("stopped")
	}

	lines = append(lines, "  "+indicator(cfg.Desktop)+
		DimStyle.Render("Desktop:  ")+boolStr(cfg.Desktop))
	lines = append(lines, "  "+indicator(cfg.Sound)+
		DimStyle.Render("Sound:    ")+boolStr(cfg.Sound))

	webhookStr := DimStyle.Render("(none)")
	if cfg.WebhookURL != "" {
		webhookStr = BrandStyle.Render(cfg.WebhookURL)
	}
	lines = append(lines, "  "+indicator(cfg.WebhookURL != "")+
		DimStyle.Render("Webhook:  ")+webhookStr)

	return strings.Join(lines, "\n")
}

// RenderNotifyHelp shows /notify usage.
func RenderNotifyHelp() string {
	header := SectionHeader("/notify — desktop & webhook notifications")
	lines := []string{
		header,
		"  " + CommandStyle.Render("/notify show") + DimStyle.Render("                      — view settings"),
		"  " + CommandStyle.Render("/notify set desktop on|off") + DimStyle.Render("        — toggle desktop alerts"),
		"  " + CommandStyle.Render("/notify set sound on|off") + DimStyle.Render("          — toggle sound"),
		"  " + CommandStyle.Render("/notify set webhook <url>") + DimStyle.Render("         — set webhook URL"),
		"  " + CommandStyle.Render("/notify clear") + DimStyle.Render("                     — reset all settings"),
		"  " + CommandStyle.Render("/notify test") + DimStyle.Render("                      — send a test notification"),
	}
	return strings.Join(lines, "\n")
}

// --- Automation rendering ---

// RenderAutoList shows all automation rules.
func RenderAutoList(rules []automation.AutoRule) string {
	if len(rules) == 0 {
		return EmptyState("No automation rules.") + "\n" +
			DimStyle.Render("  Ask the AI to create one, e.g. ") +
			CommandStyle.Render("\"buy $100 of BTC every day\"")
	}

	var lines []string
	lines = append(lines, SectionHeader("Automation Rules"))

	for _, r := range rules {
		statusStr := "running"
		if r.Status != "active" {
			statusStr = "stopped"
		}
		status := StatusIndicator(statusStr)

		typeTag := DimStyle.Render("[" + string(r.Type) + "]")
		desc := lipgloss.NewStyle().Foreground(ColorWhite).Render(r.Description)
		idStr := DimStyle.Render("  [" + r.ID[:6] + "]")

		lines = append(lines, "  "+status+desc+idStr)

		// Details line.
		details := "    " + typeTag
		if r.Schedule != "" {
			details += DimStyle.Render("  every ") + r.Schedule
		}
		action := fmt.Sprintf("  %s %s $%.0f", strings.ToUpper(r.Action), r.ActionSymbol, r.ActionValue)
		details += lipgloss.NewStyle().Foreground(ColorSecondary).Render(action)
		if r.FireCount > 0 {
			details += DimStyle.Render(fmt.Sprintf("  (%d fires)", r.FireCount))
		}
		lines = append(lines, details)
	}
	return strings.Join(lines, "\n") + NextSteps("/auto add", "/trigger list")
}

// RenderAutoConfirm renders a confirmation card for an automation fire.
func RenderAutoConfirm(rule automation.AutoRule, price float64) string {
	side := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(strings.ToUpper(rule.Action))
	if rule.Action == "sell" || rule.Action == "sell_all" {
		side = lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render(strings.ToUpper(rule.Action))
	}

	qty := 0.0
	if price > 0 {
		qty = rule.ActionValue / price
	}

	return WarningStyle.Render("  AUTOMATION FIRED ") + "\n" +
		"  " + DimStyle.Render(rule.Description) + "\n" +
		"  " + side + " " + BrandStyle.Render(rule.ActionSymbol) +
		DimStyle.Render(fmt.Sprintf("  %.4f @ %s ≈ $%.0f", qty, formatPrice(price), rule.ActionValue)) + "\n" +
		DimStyle.Render("  Press y to execute, n to skip")
}

// RenderAutoHelp shows /auto usage.
func RenderAutoHelp() string {
	header := SectionHeader("/auto — natural language automation")
	lines := []string{
		header,
		"  " + CommandStyle.Render("/auto list") + DimStyle.Render("                    — view all rules"),
		"  " + CommandStyle.Render("/auto pause <id>") + DimStyle.Render("              — pause a rule"),
		"  " + CommandStyle.Render("/auto resume <id>") + DimStyle.Render("             — resume a rule"),
		"  " + CommandStyle.Render("/auto remove <id>") + DimStyle.Render("             — delete a rule"),
		"",
		DimStyle.Render("  Create rules by asking the AI:"),
		"  " + CommandStyle.Render("\"buy $100 of BTC every day\""),
		"  " + CommandStyle.Render("\"sell ETH if it goes above 5000\""),
	}
	return strings.Join(lines, "\n")
}
