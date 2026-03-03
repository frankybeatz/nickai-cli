package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nickai/cli/internal/alert"
	"github.com/nickai/cli/internal/api"
)

// handleAlert processes /alert subcommands.
func (m *Model) handleAlert(args []string) string {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/alert BTC > 100000") + "\n" +
			DimStyle.Render("  Subcommands: /alert list, /alert clear")
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "list":
		if len(m.alerts) == 0 {
			return DimStyle.Render("  No active alerts. Create one with ") +
				CommandStyle.Render("/alert BTC > 100000")
		}
		var lines []string
		lines = append(lines, SecondaryStyle.Render("  Active Alerts\n"))
		for _, a := range m.alerts {
			lines = append(lines, "  "+StatusIndicator("running")+
				BrandStyle.Render(a.Symbol)+" "+a.Operator+" "+formatPrice(a.Target))
		}
		return strings.Join(lines, "\n")

	case "clear":
		count := len(m.alerts)
		m.alerts = nil
		_ = alert.Clear()
		return BotMsgStyle.Render("nick: ") +
			fmt.Sprintf("Cleared %d alert(s).", count)
	}

	// Parse: /alert BTC > 100000
	if len(args) < 3 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/alert BTC > 100000")
	}
	symbol := strings.ToUpper(args[0])
	op := args[1]
	if op != ">" && op != "<" {
		return ErrorStyle.Render("  Operator must be ") +
			CommandStyle.Render(">") + " or " + CommandStyle.Render("<")
	}
	target, err := strconv.ParseFloat(args[2], 64)
	if err != nil || target <= 0 {
		return ErrorStyle.Render("  Invalid target price: ") + args[2]
	}

	a := alert.Alert{
		Symbol:   symbol,
		Operator: op,
		Target:   target,
		Created:  time.Now(),
	}
	m.alerts = append(m.alerts, a)
	_ = alert.Add(a)

	return BotMsgStyle.Render("nick: ") + "Alert set: " +
		BrandStyle.Render(symbol) + " " + op + " " + formatPrice(target) +
		DimStyle.Render("  (checking every 30s, persists across restarts)")
}

// handleTrade processes /buy and /sell commands.
func (m *Model) handleTrade(side string, args []string) string {
	if !m.client.IsConfigured() {
		return connectPrompt()
	}

	if len(args) < 2 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render(fmt.Sprintf("/%s BTC 0.1", side)) +
			DimStyle.Render("  or  ") +
			CommandStyle.Render(fmt.Sprintf("/%s BTC 0.1 limit 65000", side))
	}

	symbol := NormalizeSymbol(args[0])

	qty, err := strconv.ParseFloat(args[1], 64)
	if err != nil || qty <= 0 {
		return ErrorStyle.Render("  Invalid quantity: ") + args[1] +
			"\n" + DimStyle.Render("  Must be a positive number, e.g. 0.1")
	}

	orderType := "market"
	var limitPrice float64

	if len(args) >= 4 && strings.ToLower(args[2]) == "limit" {
		orderType = "limit"
		limitPrice, err = strconv.ParseFloat(args[3], 64)
		if err != nil || limitPrice <= 0 {
			return ErrorStyle.Render("  Invalid limit price: ") + args[3] +
				"\n" + DimStyle.Render("  Must be a positive number, e.g. 65000")
		}
	}

	req := api.PlaceOrderRequest{
		Symbol:   symbol,
		Quantity: qty,
		Side:     side,
		Type:     orderType,
		Price:    limitPrice,
	}

	// Store pending trade and enter confirmation mode.
	m.pendingTrade = &req
	m.vimMode = ModeConfirm
	m.textInput.Blur()

	return RenderTradeConfirmCard(&req, m.width)
}

// handleFunding processes /funding command.
func (m *Model) handleFunding(args []string) (string, tea.Cmd) {
	if msg := m.requireAgent(); msg != "" {
		return msg, nil
	}
	prompt := "Show current funding rates for major perpetual contracts (BTC, ETH, SOL, and any other notable rates). Include annualized rates and direction. Use exchange MCP tools if available."
	if len(args) > 0 {
		prompt = "Show funding rates for: " + strings.Join(args, " ")
	}
	return m.streamToAI(prompt, "Fetching funding rates...", "funding")
}
