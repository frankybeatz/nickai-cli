package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/ai"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/automation"
	"github.com/nickai/cli/internal/commands"
	"github.com/nickai/cli/internal/journal"
	"github.com/nickai/cli/internal/notify"
	"github.com/nickai/cli/internal/risk"
	"github.com/nickai/cli/internal/strategy"
	"github.com/nickai/cli/internal/tools"
	"github.com/nickai/cli/internal/trigger"
)

// --- INSERT mode ---

func (m Model) updateInsertMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		if m.watchMode {
			m.watchMode = false
			m.watchSymbols = nil
			m.watchPrices = nil
			m.watchHistory = nil
			m.addBotMessage(BotMsgStyle.Render("nick: ") + "Watch mode ended.")
			m.updateViewport()
			return m, nil
		}
		if m.dashboardMode {
			m.dashboardMode = false
			m.addBotMessage(BotMsgStyle.Render("nick: ") + "Back to chat.")
			m.updateViewport()
			return m, nil
		}
		m.vimMode = ModeNormal
		m.normalKeyBuf = ""
		m.completionCandidates = nil
		m.textInput.Blur()
		return m, nil

	case tea.KeyTab:
		// If suggestions are visible, select the highlighted one.
		if len(m.completionCandidates) > 0 {
			return m.selectSuggestion()
		}
		return m.handleTabCompletion()

	case tea.KeyUp:
		// Navigate suggestions if visible.
		if len(m.completionCandidates) > 0 {
			if m.completionIndex > 0 {
				m.completionIndex--
				if m.completionIndex < m.completionScroll {
					m.completionScroll = m.completionIndex
				}
			}
			return m, nil
		}
		if len(m.inputHistory) == 0 {
			return m, nil
		}
		if m.historyIndex == -1 {
			m.historySaved = m.textInput.Value()
			m.historyIndex = len(m.inputHistory) - 1
		} else if m.historyIndex > 0 {
			m.historyIndex--
		}
		m.textInput.SetValue(m.inputHistory[m.historyIndex])
		m.textInput.CursorEnd()
		return m, nil

	case tea.KeyDown:
		// Navigate suggestions if visible.
		if len(m.completionCandidates) > 0 {
			max := len(m.completionCandidates) - 1
			if m.completionIndex < max {
				m.completionIndex++
				if m.completionIndex >= m.completionScroll+10 {
					m.completionScroll = m.completionIndex - 9
				}
			}
			return m, nil
		}
		if m.historyIndex == -1 {
			return m, nil
		}
		if m.historyIndex < len(m.inputHistory)-1 {
			m.historyIndex++
			m.textInput.SetValue(m.inputHistory[m.historyIndex])
		} else {
			m.historyIndex = -1
			m.textInput.SetValue(m.historySaved)
		}
		m.textInput.CursorEnd()
		return m, nil

	case tea.KeyEnter:
		// If suggestions are visible, select the highlighted one instead of submitting.
		if len(m.completionCandidates) > 0 {
			return m.selectSuggestion()
		}
		m.completionCandidates = nil
		m.historyIndex = -1
		input := strings.TrimSpace(oscLeakRe.ReplaceAllString(oscResponseRe.ReplaceAllString(m.textInput.Value(), ""), ""))
		if input == "" {
			return m, nil
		}
		m.textInput.SetValue("")

		// Save to command history (deduplicate consecutive).
		if len(m.inputHistory) == 0 || m.inputHistory[len(m.inputHistory)-1] != input {
			m.inputHistory = append(m.inputHistory, input)
			// Keep last 100 entries.
			if len(m.inputHistory) > 100 {
				m.inputHistory = m.inputHistory[len(m.inputHistory)-100:]
			}
		}

		m.messages = append(m.messages, message{
			content: UserMsgStyle.Render("you: ") + input,
			isUser:  true,
		})

		// Track recent commands for AI context.
		if strings.HasPrefix(input, "/") {
			m.trackRecentCommand(input)
		}

		result := commands.Route(input)

		return m.dispatchCommand(result)
	}

	// Forward to textInput.
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	// Strip leaked OSC color responses from the text buffer.
	if val := m.textInput.Value(); oscLeakRe.MatchString(val) {
		cleaned := oscLeakRe.ReplaceAllString(val, "")
		m.textInput.SetValue(cleaned)
		m.textInput.CursorEnd()
	}

	// Auto-show suggestions when typing a / command.
	input := m.textInput.Value()
	if strings.HasPrefix(input, "/") && !strings.Contains(input, " ") {
		m.completionCandidates = filterSuggestions(input)
		m.completionIndex = 0
		m.completionScroll = 0
	} else {
		m.completionCandidates = nil
		m.completionScroll = 0
	}

	return m, cmd
}

// --- NORMAL mode ---

func (m Model) updateNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle multi-key sequences.
	if m.normalKeyBuf == "g" {
		m.normalKeyBuf = ""
		if key == "g" {
			m.viewport.GotoTop()
			return m, nil
		}
		// Not a recognized sequence, fall through.
	}

	switch msg.Type {
	case tea.KeyDown:
		m.viewport.LineDown(1)
		return m, nil
	case tea.KeyUp:
		m.viewport.LineUp(1)
		return m, nil
	case tea.KeyCtrlD:
		m.viewport.HalfViewDown()
		return m, nil
	case tea.KeyCtrlU:
		m.viewport.HalfViewUp()
		return m, nil
	}

	switch key {
	// Navigation.
	case "j":
		m.viewport.LineDown(1)
		return m, nil
	case "k":
		m.viewport.LineUp(1)
		return m, nil
	case "d":
		m.viewport.HalfViewDown()
		return m, nil
	case "u":
		m.viewport.HalfViewUp()
		return m, nil
	case "G":
		m.viewport.GotoBottom()
		return m, nil
	case "g":
		m.normalKeyBuf = "g"
		return m, nil

	// Mode switches.
	case "i":
		m.vimMode = ModeInsert
		m.textInput.Focus()
		return m, textinput.Blink
	case "a":
		m.vimMode = ModeInsert
		m.textInput.Focus()
		// Append after current cursor position.
		m.textInput.SetCursor(m.textInput.Position() + 1)
		return m, textinput.Blink
	case "A":
		m.vimMode = ModeInsert
		m.textInput.Focus()
		m.textInput.CursorEnd()
		return m, textinput.Blink
	case "I":
		m.vimMode = ModeInsert
		m.textInput.Focus()
		m.textInput.CursorStart()
		return m, textinput.Blink
	case "o":
		m.vimMode = ModeInsert
		m.textInput.Focus()
		return m, textinput.Blink

	// Command mode.
	case ":":
		m.vimMode = ModeCommand
		m.commandBuffer = ""
		m.normalKeyBuf = ""
		return m, nil

	// Search mode.
	case "/":
		m.vimMode = ModeSearch
		m.searchBuffer = ""
		m.normalKeyBuf = ""
		return m, nil

	// Next search match.
	case "n":
		m.nextSearchMatch()
		return m, nil

	// Previous search match.
	case "N":
		m.prevSearchMatch()
		return m, nil

	// Help overlay.
	case "?":
		m.dialog = DialogState{Active: DialogHelp}
		return m, nil

	// Quit.
	case "q":
		m.cleanup()
		return m, tea.Quit
	}

	return m, nil
}

// --- COMMAND mode ---

func (m Model) updateCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.vimMode = ModeNormal
		m.commandBuffer = ""
		return m, nil

	case tea.KeyEnter:
		cmd := strings.TrimSpace(m.commandBuffer)
		m.commandBuffer = ""
		m.vimMode = ModeNormal
		return m.executeVimCommand(cmd)

	case tea.KeyBackspace:
		if len(m.commandBuffer) > 0 {
			m.commandBuffer = m.commandBuffer[:len(m.commandBuffer)-1]
		} else {
			// Empty buffer + backspace exits command mode (like vim).
			m.vimMode = ModeNormal
		}
		return m, nil
	}

	// Append printable characters.
	if msg.Type == tea.KeyRunes {
		m.commandBuffer += msg.String()
	} else if msg.Type == tea.KeySpace {
		m.commandBuffer += " "
	}

	return m, nil
}

// executeVimCommand handles : commands.
func (m Model) executeVimCommand(cmd string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return m, nil
	}

	base := parts[0]
	args := parts[1:]

	switch base {
	case "q", "q!":
		m.cleanup()
		return m, tea.Quit
	case "wq":
		m.cleanup()
		return m, tea.Quit
	case "w":
		m.addBotMessage(DimStyle.Render("  Nothing to save."))
		m.updateViewport()
		return m, nil

	case "help":
		m.dialog = DialogState{Active: DialogHelp}
		return m, nil

	case "man":
		if len(args) > 0 {
			m.addBotMessage(RenderManPage(args[0]))
		} else {
			m.addBotMessage(RenderManIndex())
		}
		m.updateViewport()
		return m, nil

	case "e":
		if len(args) == 0 {
			m.addBotMessage(ErrorStyle.Render("  Usage: ") + CommandStyle.Render(":e <file>"))
			m.updateViewport()
			return m, nil
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
		c := exec.Command(editor, args[0])
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			return editorFinishedMsg{err}
		})

	case "set":
		if len(args) == 0 {
			m.addBotMessage(ErrorStyle.Render("  Usage: ") + CommandStyle.Render(":set key=value"))
			m.updateViewport()
			return m, nil
		}
		kv := strings.SplitN(args[0], "=", 2)
		if len(kv) != 2 {
			m.addBotMessage(ErrorStyle.Render("  Usage: ") + CommandStyle.Render(":set key=value"))
			m.updateViewport()
			return m, nil
		}
		output := m.handleConfig([]string{"set", kv[0], kv[1]})
		m.addBotMessage(output)
		m.updateViewport()
		return m, nil

	case "clear":
		m.messages = nil
		m.viewport.SetContent(m.welcomeContent())
		m.viewport.GotoBottom()
		return m, nil

	case "status":
		result := commands.Route("/status")
		output := m.renderResult(result)
		if output != "" {
			m.addBotMessage(output)
		}
		m.updateViewport()
		return m, nil

	case "run":
		if len(args) == 0 {
			m.addBotMessage(ErrorStyle.Render("  Usage: ") + CommandStyle.Render(":run <workflow-name>"))
			m.updateViewport()
			return m, nil
		}
		output := m.handleWorkflow([]string{"run", args[0]})
		m.addBotMessage(output)
		m.updateViewport()
		return m, nil

	case "logs":
		if len(args) == 0 {
			m.addBotMessage(ErrorStyle.Render("  Usage: ") + CommandStyle.Render(":logs <workflow-name>"))
			m.updateViewport()
			return m, nil
		}
		output := m.handleLogs(args)
		m.addBotMessage(output)
		m.updateViewport()
		return m, nil

	case "cred":
		output := m.handleCredential(args)
		m.addBotMessage(output)
		m.updateViewport()
		return m, nil

	case "wf":
		output := m.handleWorkflow(args)
		m.addBotMessage(output)
		m.updateViewport()
		return m, nil

	default:
		m.addBotMessage(ErrorStyle.Render("  Unknown command: ") + ":" + cmd)
		m.updateViewport()
		return m, nil
	}
}

// --- SEARCH mode ---

func (m Model) updateSearchMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.vimMode = ModeNormal
		m.searchBuffer = ""
		return m, nil

	case tea.KeyEnter:
		m.searchPattern = m.searchBuffer
		m.searchBuffer = ""
		m.vimMode = ModeNormal
		m.executeSearch()
		return m, nil

	case tea.KeyBackspace:
		if len(m.searchBuffer) > 0 {
			m.searchBuffer = m.searchBuffer[:len(m.searchBuffer)-1]
		} else {
			// Empty buffer + backspace exits search mode (like vim).
			m.vimMode = ModeNormal
		}
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		m.searchBuffer += msg.String()
	} else if msg.Type == tea.KeySpace {
		m.searchBuffer += " "
	}

	return m, nil
}

func (m *Model) executeSearch() {
	m.searchMatches = nil
	m.searchCurrent = 0

	if m.searchPattern == "" {
		return
	}
	pattern := strings.ToLower(m.searchPattern)
	lines := strings.Split(m.viewContent, "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), pattern) {
			m.searchMatches = append(m.searchMatches, i)
		}
	}
	if len(m.searchMatches) > 0 {
		m.viewport.SetYOffset(m.searchMatches[0])
	}
}

// nextSearchMatch jumps to the next search match (n key in Normal mode).
func (m *Model) nextSearchMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	m.searchCurrent = (m.searchCurrent + 1) % len(m.searchMatches)
	m.viewport.SetYOffset(m.searchMatches[m.searchCurrent])
}

// prevSearchMatch jumps to the previous search match (N key in Normal mode).
func (m *Model) prevSearchMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	m.searchCurrent--
	if m.searchCurrent < 0 {
		m.searchCurrent = len(m.searchMatches) - 1
	}
	m.viewport.SetYOffset(m.searchMatches[m.searchCurrent])
}

// --- CONFIRM mode ---

func (m Model) updateConfirmMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// AI-initiated trade confirmation.
		if m.pendingAITrade != nil {
			confirmCh := m.toolRegistry.ConfirmCh
			m.toolRegistry.ResponseCh <- tools.ConfirmResponse{Approved: true}
			m.pendingAITrade = nil
			m.vimMode = ModeInsert
			m.textInput.Focus()
			m.addBotMessage(BotMsgStyle.Render("nick: ") + lipgloss.NewStyle().Foreground(ColorPrimary).Render("Trade approved — executing..."))
			m.updateViewport()
			// Re-listen for more confirmations (agent may place multiple trades).
			return m, tea.Batch(textinput.Blink, waitForConfirmation(confirmCh))
		}
		// Trigger-fired trade confirmation.
		if m.pendingTrigger != nil {
			t := *m.pendingTrigger
			m.pendingTrigger = nil
			m.vimMode = ModeInsert
			m.textInput.Focus()
			// Mark trigger as fired.
			_ = trigger.MarkFired(t.ID)
			for i := range m.triggers {
				if m.triggers[i].ID == t.ID {
					m.triggers = append(m.triggers[:i], m.triggers[i+1:]...)
					break
				}
			}
			// Execute the trade.
			symbol := strings.ToUpper(t.Symbol)
			if !strings.HasSuffix(symbol, "USDT") && !strings.HasSuffix(symbol, "USDC") && !strings.HasSuffix(symbol, "USD") {
				symbol += "USDT"
			}
			req := api.PlaceOrderRequest{
				Symbol:   symbol,
				Side:     t.Action.Side,
				Quantity: t.Action.Quantity,
				Type:     t.Action.Type,
				Price:    t.Action.Price,
			}

			// Risk check for trigger trades.
			if m.riskLimits != nil && !m.riskLimits.IsEmpty() {
				portfolio, _ := m.client.GetPortfolio()
				checkPrice := t.Action.Price
				if checkPrice == 0 {
					baseSymbol := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(symbol, "USDT"), "USDC"), "USD")
					if prices, err := m.client.GetPrices([]string{baseSymbol}); err == nil && len(prices) > 0 {
						checkPrice = prices[0].Price
					}
				}
				result := risk.CheckOrder(m.riskLimits, portfolio, symbol, t.Action.Side, t.Action.Quantity, checkPrice)
				if !result.Allowed {
					m.addBotMessage(ErrorStyle.Render("  Trigger blocked: ") + result.Reason)
					m.updateViewport()
					return m, textinput.Blink
				}
			}

			m.loading = true
			m.loadingFrame = 0
			m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " Executing triggered trade...")
			m.updateViewport()
			client := m.client
			width := m.width
			triggerID := t.ID
			triggerSymbol := t.Symbol
			return m, tea.Batch(
				textinput.Blink,
				tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
				func() tea.Msg {
					order, err := client.PlaceOrder(req)
					if err != nil {
						return apiResponseMsg{content: RenderOrderError(req.Side, req.Symbol, err)}
					}
					// Journal entry for trigger trades.
					filledPrice := order.FilledPrice
					if filledPrice == 0 {
						filledPrice = order.Price
					}
					_ = journal.Add(journal.JournalEntry{
						ID:        randomID(8),
						OrderID:   order.ID,
						Symbol:    order.Symbol,
						Side:      order.Side,
						Quantity:  order.Quantity,
						Price:     filledPrice,
						Source:    "trigger",
						Rationale: fmt.Sprintf("Trigger %s fired on %s", triggerID[:6], triggerSymbol),
						Timestamp: time.Now(),
					})
					return apiResponseMsg{content: RenderOrderConfirmation(order, width), isTrade: true}
				},
			)
		}
		// Automation rule trade confirmation.
		if m.pendingAutoRule != nil {
			r := *m.pendingAutoRule
			autoPrice := m.pendingAutoPrice
			m.pendingAutoRule = nil
			m.pendingAutoPrice = 0
			m.vimMode = ModeInsert
			m.textInput.Focus()

			qty := r.ActionValue / autoPrice
			symbol := strings.ToUpper(r.ActionSymbol)
			if !strings.HasSuffix(symbol, "USDT") && !strings.HasSuffix(symbol, "USDC") && !strings.HasSuffix(symbol, "USD") {
				symbol += "USDT"
			}

			side := r.Action
			if side == "sell_all" {
				side = "sell"
			}
			req := api.PlaceOrderRequest{
				Symbol:   symbol,
				Side:     side,
				Quantity: qty,
				Type:     r.ActionType,
			}

			// Risk check.
			if m.riskLimits != nil && !m.riskLimits.IsEmpty() {
				portfolio, _ := m.client.GetPortfolio()
				result := risk.CheckOrder(m.riskLimits, portfolio, symbol, side, qty, autoPrice)
				if !result.Allowed {
					m.addBotMessage(ErrorStyle.Render("  Automation blocked: ") + result.Reason)
					m.updateViewport()
					return m, textinput.Blink
				}
			}

			_ = automation.MarkFired(r.ID)
			// Reload automations.
			m.automations, _ = automation.Load()

			m.loading = true
			m.loadingFrame = 0
			m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " Executing automation trade...")
			m.updateViewport()
			client := m.client
			width := m.width
			ruleID := r.ID
			ruleDesc := r.Description
			notifyCfg := m.notifyConfig
			return m, tea.Batch(
				textinput.Blink,
				tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
				func() tea.Msg {
					order, err := client.PlaceOrder(req)
					if err != nil {
						return apiResponseMsg{content: RenderOrderError(req.Side, req.Symbol, err)}
					}
					filledPrice := order.FilledPrice
					if filledPrice == 0 {
						filledPrice = order.Price
					}
					_ = journal.Add(journal.JournalEntry{
						ID:        randomID(8),
						OrderID:   order.ID,
						Symbol:    order.Symbol,
						Side:      order.Side,
						Quantity:  order.Quantity,
						Price:     filledPrice,
						Source:    "automation",
						Rationale: fmt.Sprintf("Auto rule %s: %s", ruleID[:6], ruleDesc),
						Timestamp: time.Now(),
					})
					notify.Send(notifyCfg, "Trade Executed",
						fmt.Sprintf("%s %s — %s", strings.ToUpper(order.Side), order.Symbol, ruleDesc))
					return apiResponseMsg{content: RenderOrderConfirmation(order, width), isTrade: true}
				},
			)
		}
		// Strategy slice confirmation.
		if m.pendingStrategySlice != nil {
			s := *m.pendingStrategySlice
			slicePrice := m.pendingSlicePrice
			m.pendingStrategySlice = nil
			m.pendingSlicePrice = 0
			m.vimMode = ModeInsert
			m.textInput.Focus()
			// Calculate quantity from slice value.
			qty := s.SliceValue / slicePrice
			symbol := strings.ToUpper(s.Symbol)
			if !strings.HasSuffix(symbol, "USDT") && !strings.HasSuffix(symbol, "USDC") && !strings.HasSuffix(symbol, "USD") {
				symbol += "USDT"
			}
			req := api.PlaceOrderRequest{
				Symbol:   symbol,
				Side:     s.Side,
				Quantity: qty,
				Type:     "market",
			}

			// Risk check for strategy slices.
			if m.riskLimits != nil && !m.riskLimits.IsEmpty() {
				portfolio, _ := m.client.GetPortfolio()
				result := risk.CheckOrder(m.riskLimits, portfolio, symbol, s.Side, qty, slicePrice)
				if !result.Allowed {
					m.addBotMessage(ErrorStyle.Render("  Strategy slice blocked: ") + result.Reason)
					m.updateViewport()
					return m, textinput.Blink
				}
			}

			m.loading = true
			m.loadingFrame = 0
			m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " Executing TWAP slice...")
			m.updateViewport()
			client := m.client
			width := m.width
			stratID := s.ID
			return m, tea.Batch(
				textinput.Blink,
				tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
				func() tea.Msg {
					order, err := client.PlaceOrder(req)
					if err != nil {
						return apiResponseMsg{content: RenderOrderError(req.Side, req.Symbol, err)}
					}
					// Mark slice as executed.
					_ = strategy.MarkSliceExecuted(stratID, order.ID)
					// Journal entry for strategy trade.
					filledPrice := order.FilledPrice
					if filledPrice == 0 {
						filledPrice = order.Price
					}
					_ = journal.Add(journal.JournalEntry{
						ID:        randomID(8),
						OrderID:   order.ID,
						Symbol:    order.Symbol,
						Side:      order.Side,
						Quantity:  order.Quantity,
						Price:     filledPrice,
						Source:    "strategy",
						Rationale: fmt.Sprintf("TWAP slice %d/%d for %s", s.Executed+1, s.SliceCount, s.ID[:6]),
						Timestamp: time.Now(),
					})
					return apiResponseMsg{content: RenderOrderConfirmation(order, width), isTrade: true}
				},
			)
		}
		// Manual /buy /sell confirmation.
		if m.pendingTrade == nil {
			m.vimMode = ModeInsert
			m.textInput.Focus()
			return m, textinput.Blink
		}

		// Risk check for manual trades.
		if m.riskLimits != nil && !m.riskLimits.IsEmpty() && m.pendingTrade != nil {
			portfolio, _ := m.client.GetPortfolio()
			// Fetch price for risk check.
			checkPrice := m.pendingTrade.Price
			if checkPrice == 0 {
				baseSymbol := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(m.pendingTrade.Symbol, "USDT"), "USDC"), "USD")
				if prices, err := m.client.GetPrices([]string{baseSymbol}); err == nil && len(prices) > 0 {
					checkPrice = prices[0].Price
				}
			}
			result := risk.CheckOrder(m.riskLimits, portfolio, m.pendingTrade.Symbol, m.pendingTrade.Side, m.pendingTrade.Quantity, checkPrice)
			if !result.Allowed {
				m.pendingTrade = nil
				m.vimMode = ModeInsert
				m.textInput.Focus()
				m.addBotMessage(ErrorStyle.Render("  Risk limit: ") + result.Reason)
				m.updateViewport()
				return m, textinput.Blink
			}
		}

		m.loading = true
		m.loadingFrame = 0
		m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " Executing trade...")
		m.updateViewport()

		req := *m.pendingTrade
		m.pendingTrade = nil
		m.vimMode = ModeInsert
		m.textInput.Focus()

		client := m.client
		width := m.width
		return m, tea.Batch(
			textinput.Blink,
			tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
			func() tea.Msg {
				order, err := client.PlaceOrder(req)
				if err != nil {
					return apiResponseMsg{content: RenderOrderError(req.Side, req.Symbol, err)}
				}
				// Journal entry for manual trades.
				filledPrice := order.FilledPrice
				if filledPrice == 0 {
					filledPrice = order.Price
				}
				_ = journal.Add(journal.JournalEntry{
					ID:        randomID(8),
					OrderID:   order.ID,
					Symbol:    order.Symbol,
					Side:      order.Side,
					Quantity:  order.Quantity,
					Price:     filledPrice,
					Source:    "manual",
					Timestamp: time.Now(),
				})
				return apiResponseMsg{content: RenderOrderConfirmation(order, width), isTrade: true}
			},
		)

	case "n", "N", "q":
		// AI-initiated trade cancellation.
		if m.pendingAITrade != nil {
			confirmCh := m.toolRegistry.ConfirmCh
			m.toolRegistry.ResponseCh <- tools.ConfirmResponse{Approved: false}
			m.pendingAITrade = nil
			m.pendingRationale = ""
			m.vimMode = ModeInsert
			m.textInput.Focus()
			m.addBotMessage(DimStyle.Render("  AI trade declined."))
			m.updateViewport()
			return m, tea.Batch(textinput.Blink, waitForConfirmation(confirmCh))
		}
		// Trigger trade cancellation.
		if m.pendingTrigger != nil {
			t := *m.pendingTrigger
			m.pendingTrigger = nil
			_ = trigger.MarkFired(t.ID)
			for i := range m.triggers {
				if m.triggers[i].ID == t.ID {
					m.triggers = append(m.triggers[:i], m.triggers[i+1:]...)
					break
				}
			}
			m.vimMode = ModeInsert
			m.textInput.Focus()
			m.addBotMessage(DimStyle.Render("  Trigger skipped. It won't fire again."))
			m.updateViewport()
			return m, textinput.Blink
		}
		// Automation rule cancellation.
		if m.pendingAutoRule != nil {
			m.pendingAutoRule = nil
			m.pendingAutoPrice = 0
			m.vimMode = ModeInsert
			m.textInput.Focus()
			m.addBotMessage(DimStyle.Render("  Automation trade skipped."))
			m.updateViewport()
			return m, textinput.Blink
		}
		// Strategy slice cancellation.
		if m.pendingStrategySlice != nil {
			m.pendingStrategySlice = nil
			m.pendingSlicePrice = 0
			m.vimMode = ModeInsert
			m.textInput.Focus()
			m.addBotMessage(DimStyle.Render("  Strategy slice skipped."))
			m.updateViewport()
			return m, textinput.Blink
		}
		// Manual trade cancellation.
		m.pendingTrade = nil
		m.vimMode = ModeInsert
		m.textInput.Focus()
		m.addBotMessage(DimStyle.Render("  Trade cancelled."))
		m.updateViewport()
		return m, textinput.Blink
	}

	return m, nil
}

// --- Overlay dialog input ---

func (m Model) updateDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.dialog.Active {
	case DialogHelp:
		// Any key closes help.
		m.dialog.Active = DialogNone
		return m, nil

	case DialogTheme:
		names := sortedThemeNames()
		switch msg.Type {
		case tea.KeyEsc:
			m.dialog.Active = DialogNone
			return m, nil
		case tea.KeyUp:
			if m.dialog.Cursor > 0 {
				m.dialog.Cursor--
				if m.dialog.Cursor < m.dialog.ScrollOffset {
					m.dialog.ScrollOffset = m.dialog.Cursor
				}
			}
			return m, nil
		case tea.KeyDown:
			if m.dialog.Cursor < len(names)-1 {
				m.dialog.Cursor++
				if m.dialog.Cursor >= m.dialog.ScrollOffset+maxVisibleDialogItems {
					m.dialog.ScrollOffset = m.dialog.Cursor - maxVisibleDialogItems + 1
				}
			}
			return m, nil
		case tea.KeyEnter:
			selected := names[m.dialog.Cursor]
			if t, ok := Themes[selected]; ok {
				ApplyTheme(t)
				m.refreshInputStyles()
				m.cfg.Theme = selected
				_ = m.cfg.Save()
				m.addBotMessage(BotMsgStyle.Render("nick: ") +
					"Theme switched to " + CommandStyle.Render(selected))
				m.updateViewport()
			}
			m.dialog.Active = DialogNone
			return m, nil
		}
		switch msg.String() {
		case "j":
			if m.dialog.Cursor < len(names)-1 {
				m.dialog.Cursor++
				if m.dialog.Cursor >= m.dialog.ScrollOffset+maxVisibleDialogItems {
					m.dialog.ScrollOffset = m.dialog.Cursor - maxVisibleDialogItems + 1
				}
			}
			return m, nil
		case "k":
			if m.dialog.Cursor > 0 {
				m.dialog.Cursor--
				if m.dialog.Cursor < m.dialog.ScrollOffset {
					m.dialog.ScrollOffset = m.dialog.Cursor
				}
			}
			return m, nil
		}
		return m, nil

	case DialogModel:
		models := ai.AvailableModels
		switch msg.Type {
		case tea.KeyEsc:
			m.dialog.Active = DialogNone
			return m, nil
		case tea.KeyUp:
			if m.dialog.Cursor > 0 {
				m.dialog.Cursor--
				if m.dialog.Cursor < m.dialog.ScrollOffset {
					m.dialog.ScrollOffset = m.dialog.Cursor
				}
			}
			return m, nil
		case tea.KeyDown:
			if m.dialog.Cursor < len(models)-1 {
				m.dialog.Cursor++
				if m.dialog.Cursor >= m.dialog.ScrollOffset+maxVisibleDialogItems {
					m.dialog.ScrollOffset = m.dialog.Cursor - maxVisibleDialogItems + 1
				}
			}
			return m, nil
		case tea.KeyEnter:
			if m.agent == nil {
				m.addBotMessage(ErrorStyle.Render("No AI agent configured. Set an API key first."))
				m.updateViewport()
				m.dialog.Active = DialogNone
				return m, nil
			}
			selected := models[m.dialog.Cursor]
			if err := m.agent.SetModel(selected.ID); err != nil {
				m.addBotMessage(ErrorStyle.Render("Error: " + err.Error()))
			} else {
				m.cfg.Model = selected.ID
				_ = m.cfg.Save()
				m.addBotMessage(BotMsgStyle.Render("nick: ") +
					"Model switched to " + CommandStyle.Render(selected.ID))
			}
			m.updateViewport()
			m.dialog.Active = DialogNone
			return m, nil
		}
		switch msg.String() {
		case "j":
			if m.dialog.Cursor < len(models)-1 {
				m.dialog.Cursor++
				if m.dialog.Cursor >= m.dialog.ScrollOffset+maxVisibleDialogItems {
					m.dialog.ScrollOffset = m.dialog.Cursor - maxVisibleDialogItems + 1
				}
			}
			return m, nil
		case "k":
			if m.dialog.Cursor > 0 {
				m.dialog.Cursor--
				if m.dialog.Cursor < m.dialog.ScrollOffset {
					m.dialog.ScrollOffset = m.dialog.Cursor
				}
			}
			return m, nil
		}
		return m, nil

	case DialogPalette:
		switch msg.Type {
		case tea.KeyEsc:
			m.dialog.Active = DialogNone
			return m, nil
		case tea.KeyUp:
			if m.dialog.Cursor > 0 {
				m.dialog.Cursor--
				if m.dialog.Cursor < m.dialog.ScrollOffset {
					m.dialog.ScrollOffset = m.dialog.Cursor
				}
			}
			return m, nil
		case tea.KeyDown:
			maxIdx := len(m.dialog.FilteredList) - 1
			if m.dialog.Cursor < maxIdx {
				m.dialog.Cursor++
				if m.dialog.Cursor >= m.dialog.ScrollOffset+12 {
					m.dialog.ScrollOffset = m.dialog.Cursor - 11
				}
			}
			return m, nil
		case tea.KeyEnter:
			if m.dialog.Cursor < len(m.dialog.FilteredList) {
				entry := m.dialog.FilteredList[m.dialog.Cursor]
				cmd := strings.SplitN(entry, "|", 2)[0]
				m.dialog.Active = DialogNone
				// Inject the command as if typed.
				m.textInput.SetValue(cmd)
				// Simulate enter.
				return m.updateInsertMode(tea.KeyMsg{Type: tea.KeyEnter})
			}
			m.dialog.Active = DialogNone
			return m, nil
		case tea.KeyBackspace:
			if len(m.dialog.Filter) > 0 {
				m.dialog.Filter = m.dialog.Filter[:len(m.dialog.Filter)-1]
				m.dialog.FilteredList = filterPaletteCommands(m.dialog.Filter)
				m.dialog.Cursor = 0
				m.dialog.ScrollOffset = 0
			}
			return m, nil
		case tea.KeyRunes:
			m.dialog.Filter += msg.String()
			m.dialog.FilteredList = filterPaletteCommands(m.dialog.Filter)
			m.dialog.Cursor = 0
			m.dialog.ScrollOffset = 0
			return m, nil
		case tea.KeySpace:
			m.dialog.Filter += " "
			m.dialog.FilteredList = filterPaletteCommands(m.dialog.Filter)
			m.dialog.Cursor = 0
			m.dialog.ScrollOffset = 0
			return m, nil
		}
		return m, nil
	}

	// Fallback: close dialog.
	m.dialog.Active = DialogNone
	return m, nil
}

// --- Tab completion ---

func (m Model) handleTabCompletion() (tea.Model, tea.Cmd) {
	input := m.textInput.Value()

	if len(m.completionCandidates) > 0 {
		// Cycle through existing candidates.
		m.completionIndex = (m.completionIndex + 1) % len(m.completionCandidates)
		m.textInput.SetValue(m.completionCandidates[m.completionIndex])
		m.textInput.CursorEnd()
		return m, nil
	}

	candidates := buildCompletionCandidates(input)
	if len(candidates) == 0 {
		return m, nil
	}

	m.completionCandidates = candidates
	m.completionIndex = 0
	m.textInput.SetValue(candidates[0])
	m.textInput.CursorEnd()
	return m, nil
}

func buildCompletionCandidates(input string) []string {
	if input == "" {
		return nil
	}

	parts := strings.Fields(input)

	// Complete command name when typing "/..." with no space yet.
	if strings.HasPrefix(input, "/") && (len(parts) == 1 && !strings.HasSuffix(input, " ")) {
		prefix := strings.ToLower(parts[0])
		var matches []string
		for _, cmd := range knownCommands {
			if strings.HasPrefix(cmd, prefix) {
				matches = append(matches, cmd)
			}
		}
		return matches
	}

	// Complete symbol after a symbol-taking command.
	if len(parts) >= 1 && strings.HasPrefix(input, "/") {
		cmd := strings.ToLower(parts[0])
		if symbolCommands[cmd] {
			var partial string
			if strings.HasSuffix(input, " ") {
				partial = ""
			} else if len(parts) > 1 {
				partial = strings.ToUpper(parts[len(parts)-1])
			} else {
				return nil
			}

			baseInput := input
			if partial != "" {
				baseInput = input[:len(input)-len(partial)]
			}

			var matches []string
			for _, sym := range knownSymbols {
				if strings.HasPrefix(sym, partial) {
					matches = append(matches, baseInput+sym)
				}
			}
			return matches
		}
	}

	return nil
}

// --- Helper: slash suggestion selection ---

// selectSuggestion fills the input with the currently highlighted suggestion.
func (m Model) selectSuggestion() (tea.Model, tea.Cmd) {
	if m.completionIndex >= len(m.completionCandidates) {
		return m, nil
	}
	entry := m.completionCandidates[m.completionIndex]
	cmd := strings.SplitN(entry, "|", 2)[0]
	m.textInput.SetValue(cmd + " ")
	m.textInput.CursorEnd()
	m.completionCandidates = nil
	return m, nil
}

// composeSuggestions overlays the suggestion box above the input bar.
func (m Model) composeSuggestions(base string) string {
	boxWidth := min(m.width-4, 48)
	box := renderSuggestionsBox(m.completionCandidates, m.completionIndex, m.completionScroll, boxWidth)
	boxLines := strings.Split(box, "\n")
	baseLines := strings.Split(base, "\n")

	// Place the box ending just above the input bar (last 2 lines).
	insertAt := len(baseLines) - 2 - len(boxLines)
	if insertAt < 1 {
		insertAt = 1
	}

	for i, bLine := range boxLines {
		lineIdx := insertAt + i
		if lineIdx >= 0 && lineIdx < len(baseLines) {
			baseLines[lineIdx] = " " + bLine
		}
	}

	return strings.Join(baseLines, "\n")
}
