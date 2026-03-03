package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/automation"
	"github.com/nickai/cli/internal/notify"
	"github.com/nickai/cli/internal/risk"
	"github.com/nickai/cli/internal/strategy"
	"github.com/nickai/cli/internal/trigger"
)

// handleTrigger processes /trigger subcommands.
func (m *Model) handleTrigger(args []string) string {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/trigger add BTC < 60000 sell 0.5") + "\n" +
			DimStyle.Render("  Subcommands: /trigger list, /trigger add, /trigger remove <id>, /trigger clear")
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "list":
		if len(m.triggers) == 0 {
			return DimStyle.Render("  No active triggers. Create one with ") +
				CommandStyle.Render("/trigger add BTC < 60000 sell 0.5")
		}
		return RenderTriggerList(m.triggers)

	case "clear":
		count := len(m.triggers)
		m.triggers = nil
		_ = trigger.Clear()
		return BotMsgStyle.Render("nick: ") +
			fmt.Sprintf("Cleared %d trigger(s).", count)

	case "remove", "rm", "delete":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/trigger remove <id>")
		}
		idPrefix := args[1]
		found := false
		for i, t := range m.triggers {
			if len(t.ID) >= len(idPrefix) && t.ID[:len(idPrefix)] == idPrefix {
				m.triggers = append(m.triggers[:i], m.triggers[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			return ErrorStyle.Render("  No trigger found with ID prefix: ") + idPrefix
		}
		_ = trigger.Remove(idPrefix)
		return BotMsgStyle.Render("nick: ") + "Trigger removed."

	case "add":
		// /trigger add BTC < 60000 sell 0.5 [market]
		if len(args) < 6 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/trigger add BTC < 60000 sell 0.5") + "\n" +
				DimStyle.Render("  Format: /trigger add <symbol> <> or <> <price> <buy|sell> <qty> [market|limit]")
		}
		symbol := strings.ToUpper(args[1])
		op := args[2]
		if op != ">" && op != "<" {
			return ErrorStyle.Render("  Operator must be ") +
				CommandStyle.Render(">") + " or " + CommandStyle.Render("<")
		}
		target, err := strconv.ParseFloat(args[3], 64)
		if err != nil || target <= 0 {
			return ErrorStyle.Render("  Invalid target price: ") + args[3]
		}
		side := strings.ToLower(args[4])
		if side != "buy" && side != "sell" {
			return ErrorStyle.Render("  Side must be ") +
				CommandStyle.Render("buy") + " or " + CommandStyle.Render("sell")
		}
		qty, err := strconv.ParseFloat(args[5], 64)
		if err != nil || qty <= 0 {
			return ErrorStyle.Render("  Invalid quantity: ") + args[5]
		}
		orderType := "market"
		if len(args) > 6 {
			orderType = strings.ToLower(args[6])
		}

		t := trigger.Trigger{
			ID:        randomID(8),
			Symbol:    symbol,
			Operator:  op,
			Target:    target,
			Action:    trigger.Action{Side: side, Quantity: qty, Type: orderType},
			CreatedAt: time.Now(),
		}
		m.triggers = append(m.triggers, t)
		_ = trigger.Add(t)

		return BotMsgStyle.Render("nick: ") + "Trigger set: " +
			DimStyle.Render("if ") + BrandStyle.Render(symbol) + " " + op + " " + formatPrice(target) +
			DimStyle.Render(" → ") +
			lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(
				strings.ToUpper(side)+" "+strconv.FormatFloat(qty, 'f', -1, 64)+" "+symbol) +
			DimStyle.Render("  (ID: "+t.ID+")")
	}

	return ErrorStyle.Render("  Unknown subcommand: ") + sub + "\n" +
		DimStyle.Render("  Try: list, add, remove, clear")
}

// handleNotify processes /notify subcommands.
func (m *Model) handleNotify(args []string) string {
	if len(args) == 0 {
		return RenderNotifyHelp()
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "show":
		return RenderNotifyConfig(m.notifyConfig)

	case "set":
		if len(args) < 3 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/notify set desktop on|off")
		}
		key := strings.ToLower(args[1])
		value := strings.ToLower(args[2])

		switch key {
		case "desktop":
			m.notifyConfig.Desktop = (value == "on" || value == "true" || value == "1")
		case "sound":
			m.notifyConfig.Sound = (value == "on" || value == "true" || value == "1")
		case "webhook":
			m.notifyConfig.WebhookURL = args[2] // preserve case for URL
		default:
			return ErrorStyle.Render("  Unknown setting: ") + key +
				"\n" + DimStyle.Render("  Valid keys: desktop, sound, webhook")
		}

		_ = notify.Save(m.notifyConfig)
		return BotMsgStyle.Render("nick: ") + "Notification setting updated." +
			"\n" + RenderNotifyConfig(m.notifyConfig)

	case "clear":
		m.notifyConfig = &notify.Config{}
		_ = notify.Save(m.notifyConfig)
		return BotMsgStyle.Render("nick: ") + "Notification settings cleared."

	case "test":
		if m.notifyConfig.IsEmpty() {
			return ErrorStyle.Render("  No notification channels configured.") + "\n" +
				DimStyle.Render("  Set one first with ") + CommandStyle.Render("/notify set desktop on")
		}
		notify.Send(m.notifyConfig, "NickAI Test", "Notifications are working!")
		return BotMsgStyle.Render("nick: ") + "Test notification sent."
	}

	return ErrorStyle.Render("  Unknown subcommand: ") + sub + "\n" +
		DimStyle.Render("  Try: show, set, clear, test")
}

// handleAuto processes /auto subcommands.
func (m *Model) handleAuto(args []string) (string, bool) {
	if len(args) == 0 {
		return RenderAutoHelp(), false
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "list":
		m.automations, _ = automation.Load()
		return RenderAutoList(m.automations), false

	case "pause":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/auto pause <id>"), false
		}
		if err := automation.Pause(args[1]); err != nil {
			return ErrorStyle.Render("  " + err.Error()), false
		}
		m.automations, _ = automation.Load()
		return BotMsgStyle.Render("nick: ") + "Automation rule paused.", false

	case "resume":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/auto resume <id>"), false
		}
		if err := automation.Resume(args[1]); err != nil {
			return ErrorStyle.Render("  " + err.Error()), false
		}
		m.automations, _ = automation.Load()
		// May need to start tick.
		needTick := false
		for _, r := range m.automations {
			if r.Status == "active" {
				needTick = true
				break
			}
		}
		return BotMsgStyle.Render("nick: ") + "Automation rule resumed.", needTick

	case "remove", "rm", "delete":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/auto remove <id>"), false
		}
		if err := automation.Remove(args[1]); err != nil {
			return ErrorStyle.Render("  " + err.Error()), false
		}
		m.automations, _ = automation.Load()
		return BotMsgStyle.Render("nick: ") + "Automation rule removed.", false
	}

	return ErrorStyle.Render("  Unknown subcommand: ") + sub + "\n" +
		DimStyle.Render("  Try: list, pause, resume, remove"), false
}

// handleRisk processes /risk subcommands.
func (m *Model) handleRisk(args []string) string {
	if len(args) == 0 {
		return RenderRiskHelp()
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "show":
		return RenderRiskLimits(m.riskLimits)

	case "clear":
		m.riskLimits = &risk.RiskLimits{}
		_ = risk.Save(m.riskLimits)
		if m.agent != nil {
			m.agent.SetRiskInfo("")
		}
		return BotMsgStyle.Render("nick: ") + "All risk limits cleared."

	case "set":
		if len(args) < 3 {
			return RenderRiskHelp()
		}
		key := strings.ToLower(args[1])
		valueStr := strings.TrimPrefix(args[2], "$")
		valueStr = strings.TrimSuffix(valueStr, "%")
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil || value <= 0 {
			return ErrorStyle.Render("  Invalid value: ") + args[2]
		}

		if m.riskLimits == nil {
			m.riskLimits = &risk.RiskLimits{}
		}

		switch key {
		case "max-order", "maxorder":
			m.riskLimits.MaxOrderValue = value
		case "max-position", "maxposition":
			m.riskLimits.MaxPositionPct = value
		case "daily-loss", "dailyloss":
			m.riskLimits.DailyLossPct = value
		default:
			return ErrorStyle.Render("  Unknown limit: ") + key + "\n" +
				DimStyle.Render("  Valid: max-order, max-position, daily-loss")
		}

		_ = risk.Save(m.riskLimits)
		if m.agent != nil {
			m.agent.SetRiskInfo(riskPromptFromLimits(m.riskLimits))
		}
		return BotMsgStyle.Render("nick: ") + "Risk limit set.\n" + RenderRiskLimits(m.riskLimits)

	default:
		return RenderRiskHelp()
	}
}

// handleStrategy processes /strategy subcommands.
func (m *Model) handleStrategy(args []string) (string, bool) {
	if len(args) == 0 {
		return RenderStrategyHelp(), false
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "list", "ls":
		// Reload from disk to get latest state.
		all, _ := strategy.Load()
		m.strategies = all
		return RenderStrategyList(all), false

	case "cancel":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/strategy cancel <id>"), false
		}
		if err := strategy.Cancel(args[1]); err != nil {
			return ErrorStyle.Render("  " + err.Error()), false
		}
		// Reload.
		all, _ := strategy.Load()
		m.strategies = all
		return BotMsgStyle.Render("nick: ") + "Strategy cancelled.", false

	case "twap":
		// /strategy twap ETH buy $2000 4h
		if len(args) < 5 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/strategy twap <SYMBOL> <buy|sell> $<VALUE> <DURATION>"), false
		}
		symbol := strings.ToUpper(args[1])
		side := strings.ToLower(args[2])
		if side != "buy" && side != "sell" {
			return ErrorStyle.Render("  Side must be buy or sell"), false
		}
		valueStr := strings.TrimPrefix(args[3], "$")
		totalValue, err := strconv.ParseFloat(valueStr, 64)
		if err != nil || totalValue <= 0 {
			return ErrorStyle.Render("  Invalid value: ") + args[3], false
		}
		dur, err := strategy.ParseDuration(args[4])
		if err != nil {
			return ErrorStyle.Render("  " + err.Error()), false
		}

		sliceCount, intervalSec := strategy.CalcSlices(dur)
		sliceValue := totalValue / float64(sliceCount)

		s := strategy.TWAPStrategy{
			ID:          randomID(8),
			Symbol:      symbol,
			Side:        side,
			TotalValue:  totalValue,
			Duration:    args[4],
			IntervalSec: intervalSec,
			SliceCount:  sliceCount,
			SliceValue:  sliceValue,
			Executed:    0,
			Status:      "active",
			CreatedAt:   time.Now(),
			NextSliceAt: time.Now().Add(time.Duration(intervalSec) * time.Second),
		}

		if err := strategy.Add(s); err != nil {
			return ErrorStyle.Render("  Failed to save: ") + err.Error(), false
		}

		// Reload.
		all, _ := strategy.Load()
		m.strategies = all

		return BotMsgStyle.Render("nick: ") + "TWAP strategy created.\n" +
			"  " + DimStyle.Render("Symbol: ") + BrandStyle.Render(symbol) +
			DimStyle.Render("  Side: ") + strings.ToUpper(side) +
			DimStyle.Render(fmt.Sprintf("  Value: $%.0f  Duration: %s", totalValue, args[4])) + "\n" +
			"  " + DimStyle.Render(fmt.Sprintf("%d slices × $%.2f every %dm", sliceCount, sliceValue, intervalSec/60)) + "\n" +
			"  " + DimStyle.Render("ID: ") + s.ID, true

	default:
		return RenderStrategyHelp(), false
	}
}

