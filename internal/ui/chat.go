package ui

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/ai"
	"github.com/nickai/cli/internal/alert"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/automation"
	"github.com/nickai/cli/internal/backtest"
	"github.com/nickai/cli/internal/commands"
	"github.com/nickai/cli/internal/config"
	"github.com/nickai/cli/internal/credential"
	"github.com/nickai/cli/internal/journal"
	"github.com/nickai/cli/internal/mcp"
	"github.com/nickai/cli/internal/notify"
	"github.com/nickai/cli/internal/risk"
	"github.com/nickai/cli/internal/strategy"
	"github.com/nickai/cli/internal/tools"
	"github.com/nickai/cli/internal/trigger"
	"github.com/nickai/cli/internal/workflow"
)

// VimMode represents the current editing mode.
type VimMode int

const (
	ModeInsert  VimMode = iota // default — normal chat input
	ModeNormal                 // vim normal mode — navigation
	ModeCommand                // : command line
	ModeSearch                 // / search
	ModeConfirm                // trade confirmation prompt
)

// Tab completion data.
var knownCommands = []string{
	"/help", "/status", "/orders", "/agents", "/templates",
	"/buy", "/sell", "/price", "/watch", "/snapshot",
	"/market", "/pnl", "/history", "/credential", "/workflow",
	"/logs", "/man", "/config", "/clear", "/quit",
	"/alert", "/chart", "/theme", "/model", "/mcp", "/trigger",
	"/risk", "/strategy", "/notify", "/analytics", "/analyze", "/auto",
	"/backtest", "/bt", "/polymarket", "/pm", "/guide",
}

var knownSymbols = []string{
	"BTC", "ETH", "SOL", "DOGE", "ADA", "AVAX", "LINK", "DOT", "XRP",
}

var symbolCommands = map[string]bool{
	"/price": true, "/buy": true, "/sell": true,
	"/watch": true, "/chart": true, "/alert": true,
	"/trigger": true, "/analyze": true,
}

// editorFinishedMsg is sent when an external editor process completes.
type editorFinishedMsg struct{ err error }

// bootTickMsg advances the boot animation.
type bootTickMsg struct{}

// spinnerTickMsg advances the loading spinner.
type spinnerTickMsg struct{}

// aiResponseMsg carries the result of an async AI call (non-streaming fallback).
type aiResponseMsg struct {
	response string
	err      error
}

// aiStreamMsg carries a partial token from streaming.
type aiStreamMsg struct {
	token string
}

// aiStreamDoneMsg signals end of streaming with the final complete text.
type aiStreamDoneMsg struct {
	finalContent string
	err          error
}

// apiResponseMsg carries the result of an async API command.
type apiResponseMsg struct {
	content string
}

// alertCheckMsg triggers a periodic alert check.
type alertCheckMsg struct{}

// alertTriggeredMsg is sent when an alert fires.
type alertTriggeredMsg struct {
	symbol       string
	currentPrice float64
	operator     string
	target       float64
}

// aiTradeConfirmMsg is sent when the AI agent's place_order tool needs user confirmation.
type aiTradeConfirmMsg struct {
	req tools.ConfirmRequest
}

// triggerFiredMsg is sent when a conditional trigger's price condition is met.
type triggerFiredMsg struct {
	trigger trigger.Trigger
	price   float64
}

// journalEntryMsg carries a journal entry from the tool executor.
type journalEntryMsg struct {
	entry journal.JournalEntry
}

// strategyTickMsg triggers a periodic strategy check.
type strategyTickMsg struct{}

// strategySliceMsg is sent when a TWAP strategy slice is due.
type strategySliceMsg struct {
	strategy strategy.TWAPStrategy
	price    float64
}

// autoTickMsg triggers periodic automation check.
type autoTickMsg struct{}

// autoRuleFiredMsg is sent when an automation rule fires.
type autoRuleFiredMsg struct {
	rule  automation.AutoRule
	price float64
}

// tickerFetchMsg signals time to fetch new ticker data.
type tickerFetchMsg struct{}

// tickerUpdateMsg carries refreshed ticker prices.
type tickerUpdateMsg struct {
	prices []api.Price
}

// Braille spinner frames.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Thinking text variants.
var thinkingTexts = []string{"Thinking...", "Analyzing...", "Reasoning..."}

// Boot sequence ASCII logo — block letters for readability.
var bootLogo = []string{
	` ███╗   ██╗ ██╗  ██████╗ ██╗  ██╗  █████╗  ██╗`,
	` ████╗  ██║ ██║ ██╔════╝ ██║ ██╔╝ ██╔══██╗ ██║`,
	` ██╔██╗ ██║ ██║ ██║      █████╔╝  ███████║ ██║`,
	` ██║╚██╗██║ ██║ ██║      ██╔═██╗  ██╔══██║ ██║`,
	` ██║ ╚████║ ██║ ╚██████╗ ██║  ██╗ ██║  ██║ ██║`,
	` ╚═╝  ╚═══╝ ╚═╝  ╚═════╝ ╚═╝  ╚═╝ ╚═╝  ╚═╝ ╚═╝`,
}

// message represents a single chat entry.
type message struct {
	content string
	isUser  bool
}

// Model is the top-level Bubbletea model.
type Model struct {
	viewport  viewport.Model
	textInput textinput.Model
	messages  []message
	width     int
	height    int
	ready     bool

	// Vim mode state.
	vimMode       VimMode
	commandBuffer string // for COMMAND mode (:buffer)
	searchBuffer  string // for SEARCH mode (/pattern)
	searchPattern string // last search pattern
	normalKeyBuf  string // for multi-key sequences (gg)
	viewContent   string // cached viewport content for search

	// Boot sequence state.
	booting       bool
	bootFrame     int
	bootTagline   string
	bootStartTime time.Time

	// Loading spinner state.
	loading      bool
	loadingFrame int
	loadingText  string

	// Streaming state.
	streaming bool
	streamCh  chan string

	// Tab completion state.
	completionCandidates []string
	completionIndex      int
	completionScroll     int // first visible index in suggestion box

	// Command history (up/down arrow).
	inputHistory []string
	historyIndex int // -1 means not browsing history
	historySaved string // saves current input when browsing

	// Alert state (persistent — saved to ~/.nickai/alerts.json).
	alerts       []alert.Alert
	alertTicking bool

	// Trigger state (persistent — saved to ~/.nickai/triggers.json).
	triggers       []trigger.Trigger
	pendingTrigger *trigger.Trigger // trigger awaiting confirmation

	// Ticker bar state.
	tickerPrices  []api.Price
	tickerTicking bool

	// Trade confirmation state.
	pendingTrade         *api.PlaceOrderRequest   // manual /buy /sell
	pendingAITrade       *tools.ConfirmRequest     // AI-initiated place_order
	pendingStrategySlice *strategy.TWAPStrategy    // TWAP slice awaiting confirmation
	pendingSlicePrice    float64                   // price for pending strategy slice
	pendingRationale     string                    // last AI message for journal capture

	// Risk guardrails (persistent — saved to ~/.nickai/risk.json).
	riskLimits *risk.RiskLimits

	// Strategy state (persistent — saved to ~/.nickai/strategies.json).
	strategies      []strategy.TWAPStrategy
	strategyTicking bool

	// Notification config (persistent — saved to ~/.nickai/notify.json).
	notifyConfig *notify.Config

	// Automation state (persistent — saved to ~/.nickai/automations.json).
	automations      []automation.AutoRule
	autoTicking      bool
	pendingAutoRule  *automation.AutoRule
	pendingAutoPrice float64

	// Overlay dialog state.
	dialog DialogState

	// Data stores.
	cfg          *config.Config
	client       *api.PapernickClient
	agent        *ai.Agent
	toolRegistry *tools.Registry
	mcpManager   *mcp.ClientManager
	credStore    *credential.Store
	wfStore      *workflow.Store
}

// New creates the initial model, loading config from disk.
func New() Model {
	ti := textinput.New()
	ti.Placeholder = "Ask NickAI anything or type / for commands..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 80
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	ti.Prompt = "nick → "
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorWhite)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(ColorDim)

	cfg, _ := config.Load()
	client := api.NewClient(cfg)

	// Load risk limits from disk.
	riskLimits, _ := risk.Load()

	// Create tool registry and register built-in trading tools.
	// riskFn closure captures the pointer — updates to riskLimits are visible.
	riskLimitsPtr := &riskLimits
	riskFn := func() *risk.RiskLimits { return *riskLimitsPtr }
	registry := tools.NewRegistry()
	tools.RegisterBuiltins(registry, client, riskFn)

	// Connect to external MCP servers (if configured in ~/.nickai/mcp.json).
	var mcpMgr *mcp.ClientManager
	if mcpCfg, err := mcp.LoadMCPConfig(); err == nil && len(mcpCfg.MCPServers) > 0 {
		mcpMgr = mcp.NewClientManager()
		mcpMgr.ConnectAll(mcpCfg)
		mcpRiskFn := mcp.RiskLimitsFunc(riskFn)
		mcpMgr.RegisterTools(registry, mcpRiskFn)
	}

	var agent *ai.Agent
	anthKey := cfg.AnthropicKeyOrEnv()
	mmKey := cfg.MinimaxKeyOrEnv()
	if anthKey != "" || mmKey != "" {
		agent = ai.NewAgent(client, anthKey, registry)
		if mmKey != "" {
			agent.SetMinimaxKey(mmKey)
		}
		if cfg.Model != "" {
			_ = agent.SetModel(cfg.Model)
		}
	}

	credStore, _ := credential.Load()
	wfStore, _ := workflow.Load()

	// Apply saved theme.
	if cfg.Theme != "" {
		if t, ok := Themes[cfg.Theme]; ok {
			ApplyTheme(t)
		}
	}

	// Load input history from disk.
	inputHistory := loadInputHistory()

	// Load persistent alerts and triggers from disk.
	savedAlerts, _ := alert.Load()
	savedTriggers, _ := trigger.Active()

	// Load active TWAP strategies from disk.
	savedStrategies, _ := strategy.Load()

	// Load notification config.
	notifyCfg, _ := notify.Load()

	// Load automation rules.
	savedAutomations, _ := automation.Load()

	// Update risk info on agent.
	if agent != nil && riskLimits != nil && !riskLimits.IsEmpty() {
		agent.SetRiskInfo(riskPromptFromLimits(riskLimits))
	}

	// Update automation hint on agent.
	if agent != nil && len(savedAutomations) > 0 {
		activeCount := 0
		for _, r := range savedAutomations {
			if r.Status == "active" {
				activeCount++
			}
		}
		if activeCount > 0 {
			agent.SetAutoInfo(fmt.Sprintf("The user has %d active automation rule(s). You can create new ones with the create_automation tool.", activeCount))
		}
	}

	return Model{
		textInput:     ti,
		vimMode:       ModeInsert,
		booting:       true,
		bootFrame:     0,
		bootTagline:   startupTaglines[rand.Intn(len(startupTaglines))],
		bootStartTime: time.Now(),
		historyIndex:  -1,
		inputHistory:  inputHistory,
		alerts:       savedAlerts,
		triggers:     savedTriggers,
		riskLimits:   riskLimits,
		strategies:   savedStrategies,
		notifyConfig: notifyCfg,
		automations:  savedAutomations,
		cfg:          cfg,
		client:       client,
		agent:        agent,
		toolRegistry: registry,
		mcpManager:   mcpMgr,
		credStore:    credStore,
		wfStore:      wfStore,
	}
}

// cleanup shuts down MCP connections and saves history.
func (m Model) cleanup() {
	if m.mcpManager != nil {
		m.mcpManager.CloseAll()
	}
	saveInputHistory(m.inputHistory)
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		textinput.Blink,
		tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg { return bootTickMsg{} }),
	}
	// Resume alert/trigger polling if we have persistent entries from a previous session.
	if len(m.alerts) > 0 || len(m.triggers) > 0 {
		m.alertTicking = true
		cmds = append(cmds, tea.Tick(30*time.Second, func(t time.Time) tea.Msg { return alertCheckMsg{} }))
	}
	// Resume strategy ticking if we have active strategies.
	activeStrats := 0
	for _, s := range m.strategies {
		if s.Status == "active" {
			activeStrats++
		}
	}
	if activeStrats > 0 {
		m.strategyTicking = true
		cmds = append(cmds, tea.Tick(60*time.Second, func(t time.Time) tea.Msg { return strategyTickMsg{} }))
	}
	// Resume automation ticking if we have active rules.
	activeAutos := 0
	for _, r := range m.automations {
		if r.Status == "active" {
			activeAutos++
		}
	}
	if activeAutos > 0 {
		m.autoTicking = true
		cmds = append(cmds, tea.Tick(60*time.Second, func(t time.Time) tea.Msg { return autoTickMsg{} }))
	}
	// Listen for journal entries from tool executors.
	if m.toolRegistry != nil {
		cmds = append(cmds, waitForJournalEntry(m.toolRegistry.JournalCh))
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		return m, nil

	case editorFinishedMsg:
		m.vimMode = ModeNormal
		m.textInput.Blur()
		return m, nil

	case bootTickMsg:
		if !m.booting {
			return m, nil
		}
		m.bootFrame++

		logoEnd := len(bootLogo)
		taglineEnd := logoEnd + len(m.bootTagline)
		checksEnd := taglineEnd + 1 + 4*2 // gap + 4 checks * 2 frames each
		totalFrames := checksEnd + 2       // + ready message + buffer

		if m.bootFrame >= totalFrames {
			m.booting = false
			var bootCmds []tea.Cmd
			// Start live ticker if API is configured.
			if m.client.IsConfigured() {
				m.tickerTicking = true
				client := m.client
				bootCmds = append(bootCmds, func() tea.Msg {
					prices, err := client.GetPrices([]string{"BTC", "ETH", "SOL"})
					if err != nil {
						return tickerUpdateMsg{}
					}
					return tickerUpdateMsg{prices: prices}
				})
			}
			return m, tea.Batch(bootCmds...)
		}

		// Variable speed: slow for logo, fast for tagline, medium for checks.
		var delay time.Duration
		switch {
		case m.bootFrame <= logoEnd:
			delay = 150 * time.Millisecond // logo lines: slow reveal
		case m.bootFrame <= taglineEnd:
			delay = 30 * time.Millisecond // tagline typing: fast
		default:
			delay = 200 * time.Millisecond // checks: deliberate pace
		}

		return m, tea.Tick(delay, func(t time.Time) tea.Msg {
			return bootTickMsg{}
		})

	case spinnerTickMsg:
		if !m.loading {
			return m, nil
		}
		m.loadingFrame++
		m.updateViewport()
		return m, tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
			return spinnerTickMsg{}
		})

	case aiResponseMsg:
		m.loading = false
		// Replace the last "Thinking..." message with the actual response.
		if len(m.messages) > 0 && !m.messages[len(m.messages)-1].isUser {
			if msg.err != nil {
				m.messages[len(m.messages)-1].content = ErrorStyle.Render("  AI error: ") + msg.err.Error()
			} else {
				rendered := renderMarkdown(msg.response, m.width-8)
				m.messages[len(m.messages)-1].content = BotMsgStyle.Render("nick:") + "\n" + rendered
			}
		}
		m.updateViewport()
		return m, nil

	case aiStreamMsg:
		if m.streaming && len(m.messages) > 0 && !m.messages[len(m.messages)-1].isUser {
			lastMsg := &m.messages[len(m.messages)-1]
			if m.loading {
				// First token: replace spinner with nick prefix.
				m.loading = false
				lastMsg.content = BotMsgStyle.Render("nick: ") + msg.token
			} else {
				lastMsg.content += msg.token
			}
			m.updateViewport()
		}
		if m.streamCh != nil {
			return m, waitForStreamToken(m.streamCh)
		}
		return m, nil

	case aiStreamDoneMsg:
		m.loading = false
		m.streaming = false
		m.streamCh = nil
		if len(m.messages) > 0 && !m.messages[len(m.messages)-1].isUser {
			if msg.err != nil {
				m.messages[len(m.messages)-1].content = ErrorStyle.Render("  AI error: ") + msg.err.Error()
			} else {
				rendered := renderMarkdown(msg.finalContent, m.width-8)
				m.messages[len(m.messages)-1].content = BotMsgStyle.Render("nick:") + "\n" + rendered
			}
		}
		m.updateViewport()
		return m, nil

	case aiTradeConfirmMsg:
		// The AI agent wants to place an order — show confirmation prompt.
		m.pendingAITrade = &msg.req
		m.vimMode = ModeConfirm
		m.textInput.Blur()
		// Capture last AI message as rationale for journal.
		for i := len(m.messages) - 1; i >= 0; i-- {
			if !m.messages[i].isUser {
				m.pendingRationale = m.messages[i].content
				break
			}
		}
		// Show the trade confirmation card in chat.
		confirmCard := WarningStyle.Render("  AI TRADE REQUEST ") + "\n" +
			"  " + lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(msg.req.Display) + "\n" +
			DimStyle.Render("  Press y to confirm, n to cancel")
		m.addBotMessage(confirmCard)
		m.updateViewport()
		return m, nil

	case apiResponseMsg:
		m.loading = false
		// Replace the last loading message with the actual result.
		if len(m.messages) > 0 && !m.messages[len(m.messages)-1].isUser {
			m.messages[len(m.messages)-1].content = msg.content
		}
		m.updateViewport()
		return m, nil

	case alertCheckMsg:
		if len(m.alerts) == 0 && len(m.triggers) == 0 {
			m.alertTicking = false
			return m, nil
		}
		symbolSet := make(map[string]bool)
		for _, a := range m.alerts {
			symbolSet[a.Symbol] = true
		}
		for _, t := range m.triggers {
			symbolSet[t.Symbol] = true
		}
		symbols := make([]string, 0, len(symbolSet))
		for s := range symbolSet {
			symbols = append(symbols, s)
		}
		client := m.client
		alerts := make([]alert.Alert, len(m.alerts))
		copy(alerts, m.alerts)
		triggers := make([]trigger.Trigger, len(m.triggers))
		copy(triggers, m.triggers)
		return m, tea.Batch(
			func() tea.Msg {
				prices, err := client.GetPrices(symbols)
				if err != nil {
					return nil
				}
				priceMap := make(map[string]float64)
				for _, p := range prices {
					priceMap[p.Symbol] = p.Price
				}
				// Check alerts.
				for _, a := range alerts {
					normalized := api.NormalizeSymbol(a.Symbol)
					price, ok := priceMap[normalized]
					if !ok {
						continue
					}
					if (a.Operator == ">" && price > a.Target) ||
						(a.Operator == "<" && price < a.Target) {
						return alertTriggeredMsg{
							symbol: a.Symbol, currentPrice: price,
							operator: a.Operator, target: a.Target,
						}
					}
				}
				// Check triggers.
				for _, t := range triggers {
					normalized := api.NormalizeSymbol(t.Symbol)
					price, ok := priceMap[normalized]
					if !ok {
						continue
					}
					if (t.Operator == ">" && price > t.Target) ||
						(t.Operator == "<" && price < t.Target) {
						return triggerFiredMsg{trigger: t, price: price}
					}
				}
				return nil
			},
			tea.Tick(30*time.Second, func(t time.Time) tea.Msg { return alertCheckMsg{} }),
		)

	case alertTriggeredMsg:
		// Remove the triggered alert from memory and disk.
		for i, a := range m.alerts {
			if a.Symbol == msg.symbol && a.Operator == msg.operator && a.Target == msg.target {
				m.alerts = append(m.alerts[:i], m.alerts[i+1:]...)
				break
			}
		}
		_ = alert.Remove(msg.symbol, msg.operator, msg.target)
		alertContent := "\a" + WarningStyle.Render("  ALERT ") +
			BrandStyle.Render(msg.symbol) + " is now " +
			lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(formatPrice(msg.currentPrice)) +
			DimStyle.Render(fmt.Sprintf("  (target: %s %s)", msg.operator, formatPrice(msg.target)))
		m.addBotMessage(alertContent)
		m.updateViewport()
		notify.Send(m.notifyConfig, "Price Alert",
			fmt.Sprintf("%s is now %s (target: %s %s)", msg.symbol, formatPrice(msg.currentPrice), msg.operator, formatPrice(msg.target)))
		return m, nil

	case triggerFiredMsg:
		// Show trigger confirmation — user must approve before trade executes.
		m.pendingTrigger = &msg.trigger
		m.vimMode = ModeConfirm
		m.textInput.Blur()
		confirmCard := "\a" + RenderTriggerConfirm(msg.trigger, msg.price)
		m.addBotMessage(confirmCard)
		m.updateViewport()
		notify.Send(m.notifyConfig, "Trigger Fired",
			fmt.Sprintf("%s hit %s — %s %g %s", msg.trigger.Symbol, formatPrice(msg.price),
				strings.ToUpper(msg.trigger.Action.Side), msg.trigger.Action.Quantity, msg.trigger.Symbol))
		return m, nil

	case journalEntryMsg:
		// Attach pending rationale and save journal entry.
		entry := msg.entry
		if m.pendingRationale != "" {
			entry.Rationale = m.pendingRationale
			m.pendingRationale = ""
		}
		_ = journal.Add(entry)
		// Re-listen for more journal entries.
		if m.toolRegistry != nil {
			return m, waitForJournalEntry(m.toolRegistry.JournalCh)
		}
		return m, nil

	case strategyTickMsg:
		activeStrats := 0
		for _, s := range m.strategies {
			if s.Status == "active" {
				activeStrats++
			}
		}
		if activeStrats == 0 {
			m.strategyTicking = false
			return m, nil
		}
		// Check if any slice is due.
		now := time.Now()
		client := m.client
		strategiesCopy := make([]strategy.TWAPStrategy, len(m.strategies))
		copy(strategiesCopy, m.strategies)
		return m, tea.Batch(
			func() tea.Msg {
				for _, s := range strategiesCopy {
					if s.Status != "active" {
						continue
					}
					if now.After(s.NextSliceAt) {
						// Fetch current price.
						baseSymbol := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(s.Symbol, "USDT"), "USDC"), "USD")
						prices, err := client.GetPrices([]string{baseSymbol})
						if err != nil || len(prices) == 0 {
							continue
						}
						return strategySliceMsg{strategy: s, price: prices[0].Price}
					}
				}
				return nil
			},
			tea.Tick(60*time.Second, func(t time.Time) tea.Msg { return strategyTickMsg{} }),
		)

	case strategySliceMsg:
		// Show strategy slice confirmation — user must approve.
		s := msg.strategy
		m.pendingStrategySlice = &s
		m.pendingSlicePrice = msg.price
		m.vimMode = ModeConfirm
		m.textInput.Blur()
		confirmCard := "\a" + RenderStrategySliceConfirm(s, msg.price)
		m.addBotMessage(confirmCard)
		m.updateViewport()
		notify.Send(m.notifyConfig, "TWAP Slice Due",
			fmt.Sprintf("%s %s — slice %d/%d", strings.ToUpper(s.Side), s.Symbol, s.Executed+1, s.SliceCount))
		return m, nil

	case autoTickMsg:
		activeRules := 0
		for _, r := range m.automations {
			if r.Status == "active" {
				activeRules++
			}
		}
		if activeRules == 0 {
			m.autoTicking = false
			return m, nil
		}
		now := time.Now()
		client := m.client
		rulesCopy := make([]automation.AutoRule, len(m.automations))
		copy(rulesCopy, m.automations)
		return m, tea.Batch(
			func() tea.Msg {
				for _, r := range rulesCopy {
					if r.Status != "active" {
						continue
					}
					switch r.Type {
					case automation.RuleSchedule:
						if !r.NextCheck.IsZero() && now.Before(r.NextCheck) {
							continue
						}
						// Time to fire — fetch price.
						baseSymbol := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(r.ActionSymbol, "USDT"), "USDC"), "USD")
						prices, err := client.GetPrices([]string{baseSymbol})
						if err != nil || len(prices) == 0 {
							continue
						}
						return autoRuleFiredMsg{rule: r, price: prices[0].Price}

					case automation.RuleCondition:
						if r.Symbol == "" {
							continue
						}
						baseSymbol := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(r.Symbol, "USDT"), "USDC"), "USD")
						prices, err := client.GetPrices([]string{baseSymbol})
						if err != nil || len(prices) == 0 {
							continue
						}
						price := prices[0].Price
						fired := false
						if r.Operator == ">" && price > r.Target {
							fired = true
						}
						if r.Operator == "<" && price < r.Target {
							fired = true
						}
						if fired {
							return autoRuleFiredMsg{rule: r, price: price}
						}
					}
				}
				return nil
			},
			tea.Tick(60*time.Second, func(t time.Time) tea.Msg { return autoTickMsg{} }),
		)

	case autoRuleFiredMsg:
		r := msg.rule
		m.pendingAutoRule = &r
		m.pendingAutoPrice = msg.price
		m.vimMode = ModeConfirm
		m.textInput.Blur()
		confirmCard := "\a" + RenderAutoConfirm(r, msg.price)
		m.addBotMessage(confirmCard)
		m.updateViewport()
		notify.Send(m.notifyConfig, "Automation Fired", r.Description)
		return m, nil

	case tickerFetchMsg:
		if !m.tickerTicking {
			return m, nil
		}
		client := m.client
		return m, func() tea.Msg {
			prices, err := client.GetPrices([]string{"BTC", "ETH", "SOL"})
			if err != nil {
				return tickerUpdateMsg{}
			}
			return tickerUpdateMsg{prices: prices}
		}

	case tickerUpdateMsg:
		if len(msg.prices) > 0 {
			m.tickerPrices = msg.prices
		}
		if !m.tickerTicking {
			return m, nil
		}
		return m, tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
			return tickerFetchMsg{}
		})

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.cleanup()
			return m, tea.Quit
		}
		// Ignore key input during boot animation.
		if m.booting {
			return m, nil
		}

		// ── Overlay dialog input (takes priority) ──
		if m.dialog.Active != DialogNone {
			return m.updateDialog(msg)
		}

		// ── Global hotkeys (Ctrl+K, Ctrl+T, Ctrl+O) ──
		switch msg.String() {
		case "ctrl+k":
			m.dialog = DialogState{Active: DialogPalette}
			m.dialog.FilteredList = filterPaletteCommands("")
			return m, nil
		case "ctrl+t":
			m.dialog = DialogState{Active: DialogTheme}
			return m, nil
		case "ctrl+o":
			m.dialog = DialogState{Active: DialogModel}
			return m, nil
		}

		switch m.vimMode {
		case ModeInsert:
			return m.updateInsertMode(msg)
		case ModeNormal:
			return m.updateNormalMode(msg)
		case ModeCommand:
			return m.updateCommandMode(msg)
		case ModeSearch:
			return m.updateSearchMode(msg)
		case ModeConfirm:
			return m.updateConfirmMode(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 10

		headerHeight := 1 // top bar
		inputHeight := 3  // input area
		vpHeight := msg.Height - headerHeight - inputHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, vpHeight)
			m.viewport.MouseWheelEnabled = false
			m.viewport.SetContent(RenderWelcome(msg.Width, m.client.IsConfigured()))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = vpHeight
		}
		m.updateViewport()
	}

	// Forward non-key messages to sub-components.
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// --- INSERT mode ---

func (m Model) updateInsertMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
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
		input := strings.TrimSpace(m.textInput.Value())
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

		result := commands.Route(input)

		switch result.Type {
		case commands.TypeQuit:
			m.cleanup()
			return m, tea.Quit
		case commands.TypeClear:
			m.messages = nil
			m.viewport.SetContent(RenderWelcome(m.width, m.client.IsConfigured()))
			m.viewport.GotoBottom()
			return m, nil

		case commands.TypeChat:
			// Streaming AI call with loading spinner.
			if m.agent == nil {
				m.addBotMessage(BotMsgStyle.Render("nick: ") +
					"I need an Anthropic API key to chat. Set one with " +
					CommandStyle.Render("/config set anthropic_key <key>") +
					" or " + DimStyle.Render("export ANTHROPIC_API_KEY=..."))
				m.updateViewport()
				return m, nil
			}
			m.loading = true
			m.streaming = true
			m.loadingFrame = 0
			m.loadingText = "Thinking..."
			m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)
			m.updateViewport()
			tokenCh := make(chan string, 100)
			m.streamCh = tokenCh
			agent := m.agent
			userInput := result.Input
			confirmCh := m.toolRegistry.ConfirmCh
			return m, tea.Batch(
				tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
				func() tea.Msg {
					defer close(tokenCh)
					resp, err := agent.ChatStream(userInput, tokenCh)
					return aiStreamDoneMsg{finalContent: resp, err: err}
				},
				waitForStreamToken(tokenCh),
				waitForConfirmation(confirmCh),
			)

		case commands.TypeBuy, commands.TypeSell:
			// Trade confirmation flow (synchronous — no API call yet).
			side := "buy"
			if result.Type == commands.TypeSell {
				side = "sell"
			}
			output := m.handleTrade(side, result.Args)
			if output != "" {
				m.addBotMessage(output)
			}
			m.updateViewport()
			return m, nil

		case commands.TypeAlert:
			// Alert management (synchronous).
			output := m.handleAlert(result.Args)
			if output != "" {
				m.addBotMessage(output)
			}
			m.updateViewport()
			// Start alert/trigger ticker if we just added the first alert.
			if !m.alertTicking && (len(m.alerts) > 0 || len(m.triggers) > 0) {
				m.alertTicking = true
				return m, tea.Tick(30*time.Second, func(t time.Time) tea.Msg { return alertCheckMsg{} })
			}
			return m, nil

		case commands.TypeTrigger:
			// Trigger management (synchronous).
			output := m.handleTrigger(result.Args)
			if output != "" {
				m.addBotMessage(output)
			}
			m.updateViewport()
			// Start polling if we just added the first trigger.
			if !m.alertTicking && (len(m.alerts) > 0 || len(m.triggers) > 0) {
				m.alertTicking = true
				return m, tea.Tick(30*time.Second, func(t time.Time) tea.Msg { return alertCheckMsg{} })
			}
			return m, nil

		case commands.TypeRisk:
			output := m.handleRisk(result.Args)
			if output != "" {
				m.addBotMessage(output)
			}
			m.updateViewport()
			return m, nil

		case commands.TypeStrategy:
			output, startTick := m.handleStrategy(result.Args)
			if output != "" {
				m.addBotMessage(output)
			}
			m.updateViewport()
			if startTick && !m.strategyTicking {
				m.strategyTicking = true
				return m, tea.Tick(60*time.Second, func(t time.Time) tea.Msg { return strategyTickMsg{} })
			}
			return m, nil

		case commands.TypeNotify:
			output := m.handleNotify(result.Args)
			if output != "" {
				m.addBotMessage(output)
			}
			m.updateViewport()
			return m, nil

		case commands.TypeAuto:
			output, startTick := m.handleAuto(result.Args)
			if output != "" {
				m.addBotMessage(output)
			}
			m.updateViewport()
			if startTick && !m.autoTicking {
				m.autoTicking = true
				return m, tea.Tick(60*time.Second, func(t time.Time) tea.Msg { return autoTickMsg{} })
			}
			return m, nil

		case commands.TypeBacktest:
			output, cmd := m.handleBacktest(result.Args)
			if output != "" {
				m.addBotMessage(output)
			}
			m.updateViewport()
			if cmd != nil {
				return m, cmd
			}
			return m, nil

		case commands.TypePolymarket:
			output, cmd := m.handlePolymarket(result.Args)
			if output != "" {
				m.addBotMessage(output)
			}
			m.updateViewport()
			if cmd != nil {
				return m, cmd
			}
			return m, nil

		case commands.TypeGuide:
			output := m.handleGuide(result.Args)
			if output != "" {
				m.addBotMessage(output)
			}
			m.updateViewport()
			return m, nil

		case commands.TypeAnalytics:
			// Async API call with loading spinner.
			m.loading = true
			m.loadingFrame = 0
			m.loadingText = "Computing analytics..."
			m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)
			m.updateViewport()
			rCopy := result
			return m, tea.Batch(
				tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
				func() tea.Msg {
					output := m.renderResult(rCopy)
					return apiResponseMsg{content: output}
				},
			)

		case commands.TypeAnalyze:
			// Async API call with loading spinner.
			m.loading = true
			m.loadingFrame = 0
			m.loadingText = "Analyzing market..."
			m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)
			m.updateViewport()
			rCopy := result
			return m, tea.Batch(
				tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
				func() tea.Msg {
					output := m.renderResult(rCopy)
					return apiResponseMsg{content: output}
				},
			)

		case commands.TypePrice, commands.TypeStatus, commands.TypeOrders,
			commands.TypeSnapshot, commands.TypeMarket, commands.TypePnl,
			commands.TypeHistory, commands.TypeChart:
			// Async API call with loading spinner.
			m.loading = true
			m.loadingFrame = 0
			m.loadingText = apiLoadingText(result.Type)
			m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)
			m.updateViewport()
			rCopy := result
			return m, tea.Batch(
				tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
				func() tea.Msg {
					output := m.renderResult(rCopy)
					return apiResponseMsg{content: output}
				},
			)

		default:
			output := m.renderResult(result)
			if output != "" {
				m.messages = append(m.messages, message{
					content: output,
					isUser:  false,
				})
			}
		}

		m.updateViewport()
		return m, nil
	}

	// Forward to textInput.
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

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
		// Append after current cursor position (don't move to end).
		return m, textinput.Blink
	case "A":
		m.vimMode = ModeInsert
		m.textInput.Focus()
		m.textInput.CursorEnd()
		return m, textinput.Blink
	case "I":
		m.vimMode = ModeInsert
		m.textInput.SetValue("")
		m.textInput.Focus()
		return m, textinput.Blink
	case "o":
		m.vimMode = ModeInsert
		m.textInput.SetValue("")
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
		m.addBotMessage(RenderHelp())
		m.updateViewport()
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
		m.viewport.SetContent(RenderWelcome(m.width, m.client.IsConfigured()))
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
	if m.searchPattern == "" {
		return
	}
	pattern := strings.ToLower(m.searchPattern)
	lines := strings.Split(m.viewContent, "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), pattern) {
			m.viewport.SetYOffset(i)
			return
		}
	}
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
					return apiResponseMsg{content: RenderOrderConfirmation(order, width)}
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
					return apiResponseMsg{content: RenderOrderConfirmation(order, width)}
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
					return apiResponseMsg{content: RenderOrderConfirmation(order, width)}
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
				return apiResponseMsg{content: RenderOrderConfirmation(order, width)}
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
			}
			return m, nil
		case tea.KeyDown:
			if m.dialog.Cursor < len(names)-1 {
				m.dialog.Cursor++
			}
			return m, nil
		case tea.KeyEnter:
			selected := names[m.dialog.Cursor]
			if t, ok := Themes[selected]; ok {
				ApplyTheme(t)
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
			}
			return m, nil
		case "k":
			if m.dialog.Cursor > 0 {
				m.dialog.Cursor--
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
			}
			return m, nil
		case tea.KeyDown:
			if m.dialog.Cursor < len(models)-1 {
				m.dialog.Cursor++
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
			}
			return m, nil
		case "k":
			if m.dialog.Cursor > 0 {
				m.dialog.Cursor--
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

// --- Alert handling ---

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

// --- Trigger handling ---

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

// --- Notify handling ---

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

// --- Automation handling ---

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

// --- /backtest handler ---

func (m *Model) handleBacktest(args []string) (string, tea.Cmd) {
	if len(args) == 0 {
		return RenderBacktestHelp(), nil
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "presets", "list":
		return RenderBacktestPresets(), nil

	case "run":
		if len(args) < 3 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/backtest run <preset> <symbol> [period]"), nil
		}
		presetName := strings.ToLower(args[1])
		symbol := strings.ToUpper(args[2])
		period := "180d"
		if len(args) >= 4 {
			period = args[3]
		}

		preset := backtest.GetPreset(presetName)
		if preset == nil {
			return ErrorStyle.Render("  Unknown preset: ") + presetName + "\n" +
				DimStyle.Render("  Run /backtest presets to see available strategies"), nil
		}

		// Run backtest asynchronously with loading spinner.
		m.loading = true
		m.loadingFrame = 0
		m.loadingText = "Running backtest..."
		m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)

		strat := preset.Strategy
		strat.Symbol = symbol
		strat.Period = period

		return "", tea.Batch(
			tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
			func() tea.Msg {
				result, err := backtest.Run(strat)
				if err != nil {
					return apiResponseMsg{content: ErrorStyle.Render("  Backtest failed: ") + err.Error()}
				}
				return apiResponseMsg{content: RenderBacktestCard(result)}
			},
		)

	default:
		// Pass to AI as natural language backtest request.
		if m.agent == nil {
			return BotMsgStyle.Render("nick: ") +
				"I need an Anthropic API key to chat. Set one with " +
				CommandStyle.Render("/config set anthropic_key <key>"), nil
		}

		prompt := "Backtest the following strategy using the backtest_strategy tool: " + strings.Join(args, " ")
		m.loading = true
		m.streaming = true
		m.loadingFrame = 0
		m.loadingText = "Building backtest..."
		m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)

		tokenCh := make(chan string, 100)
		m.streamCh = tokenCh
		agent := m.agent
		confirmCh := m.toolRegistry.ConfirmCh
		return "", tea.Batch(
			tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
			func() tea.Msg {
				defer close(tokenCh)
				resp, err := agent.ChatStream(prompt, tokenCh)
				return aiStreamDoneMsg{finalContent: resp, err: err}
			},
			waitForStreamToken(tokenCh),
			waitForConfirmation(confirmCh),
		)
	}
}

// --- /polymarket handler ---

func (m *Model) handlePolymarket(args []string) (string, tea.Cmd) {
	if m.agent == nil {
		return BotMsgStyle.Render("nick: ") +
			"I need an Anthropic API key for Polymarket analysis. Set one with " +
			CommandStyle.Render("/config set anthropic_key <key>"), nil
	}

	sub := "scan"
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}

	var prompt string
	switch sub {
	case "scan":
		preset := backtest.GetAnalysisPreset("polymarket-scan")
		if preset != nil {
			prompt = preset.Prompt
		} else {
			prompt = "Scan top Polymarket events and find mispriced contracts using available MCP tools."
		}
	case "analyze":
		event := strings.Join(args[1:], " ")
		if event == "" {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/polymarket analyze <event>"), nil
		}
		preset := backtest.GetAnalysisPreset("polymarket-deep")
		if preset != nil {
			prompt = fmt.Sprintf(preset.Prompt, event)
		} else {
			prompt = "Do a deep analysis of this Polymarket event: " + event
		}
	case "hot":
		prompt = "Show trending Polymarket events with the biggest volume and price moves using available tools."
	default:
		prompt = "Analyze the following using Polymarket tools: " + strings.Join(args, " ")
	}

	m.loading = true
	m.streaming = true
	m.loadingFrame = 0
	m.loadingText = "Analyzing prediction markets..."
	m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)

	tokenCh := make(chan string, 100)
	m.streamCh = tokenCh
	agent := m.agent
	confirmCh := m.toolRegistry.ConfirmCh
	return "", tea.Batch(
		tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
		func() tea.Msg {
			defer close(tokenCh)
			resp, err := agent.ChatStream(prompt, tokenCh)
			return aiStreamDoneMsg{finalContent: resp, err: err}
		},
		waitForStreamToken(tokenCh),
		waitForConfirmation(confirmCh),
	)
}

// --- /guide handler ---

func (m *Model) handleGuide(args []string) string {
	section := "start"
	if len(args) > 0 {
		section = strings.ToLower(args[0])
	}
	return RenderGuideCard(section)
}

// --- Helper: streaming ---

// waitForStreamToken returns a tea.Cmd that reads the next token from a
// streaming channel. Returns nil when the channel is closed.
func waitForStreamToken(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		token, ok := <-ch
		if !ok {
			return nil
		}
		return aiStreamMsg{token: token}
	}
}

// waitForConfirmation listens for AI trade confirmation requests.
// Runs concurrently with the streaming agent — when the agent's place_order
// blocks on ConfirmCh, this returns an aiTradeConfirmMsg to the Bubbletea loop.
func waitForConfirmation(ch <-chan tools.ConfirmRequest) tea.Cmd {
	return func() tea.Msg {
		req, ok := <-ch
		if !ok {
			return nil
		}
		return aiTradeConfirmMsg{req: req}
	}
}

// randomID generates a random alphanumeric string of length n.
func randomID(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// --- Helper: add bot message ---

func (m *Model) addBotMessage(content string) {
	m.messages = append(m.messages, message{
		content: content,
		isUser:  false,
	})
}

// --- Command rendering ---

func (m *Model) renderResult(r commands.Result) string {
	switch r.Type {
	case commands.TypeHelp:
		return RenderHelp()

	case commands.TypeAgents:
		return RenderAgentListMock(m.width)

	case commands.TypeTemplates:
		return RenderTemplateList(m.width)

	case commands.TypeStatus:
		if m.client.IsConfigured() {
			return RenderStatusLive(m.client, m.mcpManager, m.width)
		}
		return RenderStatusMock(m.mcpManager)

	case commands.TypeOrders:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		return RenderOrderList(m.client, m.width)

	case commands.TypePrice:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		if len(r.Args) == 0 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/price BTC ETH SOL")
		}
		symbols := make([]string, len(r.Args))
		for i, s := range r.Args {
			symbols[i] = strings.ToUpper(s)
		}
		return RenderPrices(m.client, symbols, m.width)

	case commands.TypeBuy:
		return m.handleTrade("buy", r.Args)

	case commands.TypeSell:
		return m.handleTrade("sell", r.Args)

	case commands.TypeConfig:
		return m.handleConfig(r.Args)

	case commands.TypeCredential:
		return m.handleCredential(r.Args)

	case commands.TypeWorkflow:
		return m.handleWorkflow(r.Args)

	case commands.TypeLogs:
		return m.handleLogs(r.Args)

	case commands.TypeMan:
		if len(r.Args) > 0 {
			return RenderManPage(r.Args[0])
		}
		return RenderManIndex()

	case commands.TypeWatch:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		if len(r.Args) == 0 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/watch BTC ETH SOL")
		}
		symbols := make([]string, len(r.Args))
		for i, s := range r.Args {
			symbols[i] = strings.ToUpper(s)
		}
		return RenderWatch(m.client, symbols, m.width)

	case commands.TypeUnknown:
		return ErrorStyle.Render("Unknown command: ") + r.Input + "\n" +
			DimStyle.Render("Type /help for available commands.")

	case commands.TypeChart:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		if len(r.Args) == 0 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/chart BTC")
		}
		return RenderChart(m.client, strings.ToUpper(r.Args[0]), m.width)

	case commands.TypeAlert:
		return m.handleAlert(r.Args)

	case commands.TypeTrigger:
		return m.handleTrigger(r.Args)

	case commands.TypeRisk:
		return m.handleRisk(r.Args)

	case commands.TypeStrategy:
		output, _ := m.handleStrategy(r.Args)
		return output

	case commands.TypeNotify:
		return m.handleNotify(r.Args)

	case commands.TypeAnalytics:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		return RenderAnalytics(m.client, m.width)

	case commands.TypeAnalyze:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		if len(r.Args) == 0 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/analyze BTC")
		}
		return RenderAnalysis(m.client, strings.ToUpper(r.Args[0]), m.width)

	case commands.TypeAuto:
		output, _ := m.handleAuto(r.Args)
		return output

	case commands.TypeTheme:
		return m.handleTheme(r.Args)

	case commands.TypeModel:
		return m.handleModel(r.Args)

	case commands.TypeMCP:
		return m.handleMCP(r.Args)

	case commands.TypeSnapshot:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		return RenderSnapshot(m.client, m.width)

	case commands.TypeMarket:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		return RenderMarket(m.client, m.width)

	case commands.TypePnl:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		return RenderPnl(m.client, m.width)

	case commands.TypeHistory:
		if !m.client.IsConfigured() {
			return connectPrompt()
		}
		return RenderHistory(m.client, m.width)

	case commands.TypeChat:
		if m.agent != nil {
			resp, err := m.agent.Chat(r.Input)
			if err != nil {
				return ErrorStyle.Render("  AI error: ") + err.Error()
			}
			rendered := renderMarkdown(resp, m.width-8)
			return BotMsgStyle.Render("nick:") + "\n" + rendered
		}
		return BotMsgStyle.Render("nick: ") +
			"I need an Anthropic API key to chat. Set one with " +
			CommandStyle.Render("/config set anthropic_key <key>") +
			" or " + DimStyle.Render("export ANTHROPIC_API_KEY=...")

	default:
		return ""
	}
}

// handleConfig processes /config subcommands.
func (m *Model) handleConfig(args []string) string {
	if len(args) == 0 {
		return RenderConfigHelp()
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "init":
		// Auto-provision: create anonymous account and store API key.
		// Allow re-provisioning with "init force" if user set wrong key.
		if m.cfg.APIKey != "" && !(len(args) > 1 && args[1] == "force") {
			return BotMsgStyle.Render("nick: ") + "API key already configured. " +
				DimStyle.Render("Use ") + CommandStyle.Render("/config show") +
				DimStyle.Render(" to view, or ") + CommandStyle.Render("/config init force") +
				DimStyle.Render(" to re-provision.")
		}
		name := fmt.Sprintf("nickai-%s", randomID(8))
		baseURL := m.cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://paper.getnick.ai/api/v1"
		}
		result, err := api.CreateAccount(baseURL, name)
		if err != nil {
			return ErrorStyle.Render("  Account creation failed: ") + err.Error()
		}
		m.cfg.APIKey = result.User.APIKey
		if err := m.cfg.Save(); err != nil {
			return ErrorStyle.Render("  Failed to save config: ") + err.Error()
		}
		m.client.UpdateConfig(m.cfg)
		return RenderConfigInit(result.User.APIKey, result.User.Name)

	case "show":
		return RenderConfigShow(m.cfg)

	case "test":
		if !m.client.IsConfigured() {
			return ErrorStyle.Render("  No API key configured. ") +
				"Set one first with " + CommandStyle.Render("/config set api_key <key>")
		}
		return RenderConfigTest(m.client)

	case "set":
		if len(args) < 3 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/config set api_key <key>") + " or " +
				CommandStyle.Render("/config set url <url>")
		}
		key := strings.ToLower(args[1])
		value := args[2]

		switch key {
		case "api_key":
			m.cfg.APIKey = value
		case "url":
			m.cfg.BaseURL = value
		case "anthropic_key":
			m.cfg.AnthropicKey = value
		case "minimax_key":
			m.cfg.MinimaxKey = value
		default:
			return ErrorStyle.Render("  Unknown config key: ") + key +
				"\n" + DimStyle.Render("  Valid keys: api_key, url, anthropic_key, minimax_key")
		}

		if err := m.cfg.Save(); err != nil {
			return ErrorStyle.Render("  Failed to save config: ") + err.Error()
		}
		m.client.UpdateConfig(m.cfg)

		if akKey := m.cfg.AnthropicKeyOrEnv(); akKey != "" {
			if m.agent == nil {
				m.agent = ai.NewAgent(m.client, akKey, m.toolRegistry)
			}
		}
		if mmKey := m.cfg.MinimaxKeyOrEnv(); mmKey != "" {
			if m.agent == nil {
				m.agent = ai.NewAgent(m.client, "", m.toolRegistry)
			}
			m.agent.SetMinimaxKey(mmKey)
		}

		return RenderConfigSet(key, value)

	case "reset":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/config reset api_key") + "\n" +
				DimStyle.Render("  Valid keys: api_key, anthropic_key, minimax_key")
		}
		key := strings.ToLower(args[1])
		switch key {
		case "api_key":
			m.cfg.APIKey = ""
		case "anthropic_key":
			m.cfg.AnthropicKey = ""
		case "minimax_key":
			m.cfg.MinimaxKey = ""
		default:
			return ErrorStyle.Render("  Unknown config key: ") + key +
				"\n" + DimStyle.Render("  Valid keys: api_key, anthropic_key, minimax_key")
		}
		if err := m.cfg.Save(); err != nil {
			return ErrorStyle.Render("  Failed to save config: ") + err.Error()
		}
		m.client.UpdateConfig(m.cfg)
		return BotMsgStyle.Render("nick: ") + "Cleared " + CommandStyle.Render(key) + "."

	default:
		return RenderConfigHelp()
	}
}

// handleMCP processes /mcp subcommands.
func (m *Model) handleMCP(args []string) string {
	if len(args) == 0 {
		return RenderMCPHelp()
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "list", "ls":
		return m.renderMCPList()

	case "search":
		query := ""
		if len(args) > 1 {
			query = strings.Join(args[1:], " ")
		}
		results := mcp.SearchRegistry(query)
		if len(results) == 0 {
			return BotMsgStyle.Render("nick: ") + "No servers found for " +
				CommandStyle.Render(query) + "." +
				DimStyle.Render("\n  Try: /mcp search trading, /mcp search defi, /mcp search blockchain")
		}
		return RenderMCPSearchResults(results)

	case "info":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/mcp info <name>")
		}
		entry := mcp.GetEntry(args[1])
		if entry == nil {
			return ErrorStyle.Render("  Unknown server: ") + args[1] +
				DimStyle.Render("\n  Use ") + CommandStyle.Render("/mcp search") +
				DimStyle.Render(" to browse available servers.")
		}
		return RenderMCPInfo(entry)

	case "add":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/mcp add <name>")
		}
		entry := mcp.GetEntry(args[1])
		if entry == nil {
			return ErrorStyle.Render("  Unknown server: ") + args[1] +
				DimStyle.Render("\n  Use ") + CommandStyle.Render("/mcp search") +
				DimStyle.Render(" to browse available servers.")
		}
		// Check if required env vars are set.
		var missing []string
		for _, key := range entry.EnvKeys {
			if os.Getenv(key) == "" {
				missing = append(missing, key)
			}
		}
		if len(missing) > 0 {
			lines := []string{
				BotMsgStyle.Render("nick: ") + "To add " + BrandStyle.Render(entry.DisplayName) + ", set these env vars first:",
				"",
			}
			for _, k := range missing {
				hint := "<your-value>"
				if entry.EnvHints != nil {
					if h, ok := entry.EnvHints[k]; ok {
						hint = h
					}
				}
				lines = append(lines, "  "+CommandStyle.Render("export "+k+"=")+DimStyle.Render(hint))
			}
			lines = append(lines, "", DimStyle.Render("  Then run ")+CommandStyle.Render("/mcp add "+entry.Name)+DimStyle.Render(" again."))
			return strings.Join(lines, "\n")
		}
		// Write to mcp.json config.
		err := mcp.AddServerToConfig(entry)
		if err != nil {
			return ErrorStyle.Render("  Failed to save MCP config: ") + err.Error()
		}
		return BotMsgStyle.Render("nick: ") + "Added " + BrandStyle.Render(entry.DisplayName) + " to " +
			DimStyle.Render("~/.nickai/mcp.json") + "." +
			DimStyle.Render("\n  Restart nickai to activate, or it will load on next launch.")

	case "remove", "rm":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/mcp remove <name>")
		}
		err := mcp.RemoveServerFromConfig(args[1])
		if err != nil {
			return ErrorStyle.Render("  " + err.Error())
		}
		return BotMsgStyle.Render("nick: ") + "Removed " + CommandStyle.Render(args[1]) + " from config." +
			DimStyle.Render("\n  Restart nickai to apply changes.")

	case "quick":
		// Add all servers that need no API keys.
		var added []string
		for _, entry := range mcp.CuratedRegistry {
			if len(entry.EnvKeys) == 0 {
				e := entry
				if err := mcp.AddServerToConfig(&e); err == nil {
					added = append(added, entry.DisplayName)
				}
			}
		}
		if len(added) == 0 {
			return BotMsgStyle.Render("nick: ") + "All free servers already configured."
		}
		lines := []string{
			BotMsgStyle.Render("nick: ") + fmt.Sprintf("Added %d servers (no API keys needed):", len(added)),
			"",
		}
		for _, name := range added {
			lines = append(lines, "  "+StatusIndicator("running")+BrandStyle.Render(name))
		}
		lines = append(lines, "", DimStyle.Render("  Restart nickai to connect them all."))
		return strings.Join(lines, "\n")

	default:
		return RenderMCPHelp()
	}
}

// renderMCPList shows connected MCP servers and their tools.
func (m *Model) renderMCPList() string {
	lines := []string{SecondaryStyle.Render("  MCP Servers\n")}

	// Show connected servers.
	if m.mcpManager != nil && m.mcpManager.ConnectionCount() > 0 {
		for _, conn := range m.mcpManager.Connections() {
			lines = append(lines, "  "+StatusIndicator("running")+BrandStyle.Render(conn.Name)+
				DimStyle.Render(fmt.Sprintf("  (%d tools)", len(conn.Tools))))
			for _, t := range conn.Tools {
				lines = append(lines, "    "+CommandStyle.Render(t.Name)+
					DimStyle.Render("  "+t.Description))
			}
		}
	} else {
		lines = append(lines, DimStyle.Render("  No MCP servers connected."))
		lines = append(lines, "")
		lines = append(lines, DimStyle.Render("  Get started:")+
			"\n  "+CommandStyle.Render("/mcp search")+DimStyle.Render("        — browse available servers")+
			"\n  "+CommandStyle.Render("/mcp add <name>")+DimStyle.Render("   — install a server"))
	}

	// Show failed connections.
	if m.mcpManager != nil && len(m.mcpManager.Failed()) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+ErrorStyle.Render("Failed to connect:"))
		for _, f := range m.mcpManager.Failed() {
			lines = append(lines, "  "+StatusIndicator("stopped")+
				WarningStyle.Render(f.Name)+DimStyle.Render("  "+f.Error))
		}
	}

	// Show built-in tool count.
	if m.toolRegistry != nil {
		builtinCount := 0
		for _, entry := range m.toolRegistry.All() {
			if entry.Source == "builtin" {
				builtinCount++
			}
		}
		lines = append(lines, "")
		lines = append(lines, DimStyle.Render(fmt.Sprintf("  + %d built-in tools (get_prices, get_portfolio, get_orders, place_order)", builtinCount)))
	}

	return strings.Join(lines, "\n")
}

// handleCredential processes /credential subcommands.
func (m *Model) handleCredential(args []string) string {
	if len(args) == 0 {
		return RenderCredentialList(m.credStore)
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "list":
		return RenderCredentialList(m.credStore)

	case "add":
		if len(args) < 5 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/credential add <name> <exchange> <api_key> <api_secret>") +
				"\n" + DimStyle.Render("  Exchanges: "+strings.Join(credential.SupportedExchanges(), ", "))
		}
		name := args[1]
		exchange := strings.ToLower(args[2])
		apiKey := args[3]
		apiSecret := args[4]

		if !credential.IsSupportedExchange(exchange) {
			return ErrorStyle.Render("  Unsupported exchange: ") + exchange +
				"\n" + DimStyle.Render("  Supported: "+strings.Join(credential.SupportedExchanges(), ", "))
		}

		m.credStore.Add(credential.Credential{
			Name:      name,
			Exchange:  exchange,
			APIKey:    apiKey,
			APISecret: apiSecret,
		})
		if err := m.credStore.Save(); err != nil {
			return ErrorStyle.Render("  Failed to save credential: ") + err.Error()
		}
		return RenderCredentialAdded(name, exchange)

	case "remove":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/credential remove <name>")
		}
		name := args[1]
		if !m.credStore.Remove(name) {
			return ErrorStyle.Render("  Credential not found: ") + name
		}
		if err := m.credStore.Save(); err != nil {
			return ErrorStyle.Render("  Failed to save: ") + err.Error()
		}
		return RenderCredentialRemoved(name)

	default:
		return ErrorStyle.Render("  Unknown subcommand: ") + sub +
			"\n" + DimStyle.Render("  Usage: /credential <list|add|remove>")
	}
}

// handleWorkflow processes /workflow subcommands.
func (m *Model) handleWorkflow(args []string) string {
	if len(args) == 0 {
		return RenderWorkflowList(m.wfStore)
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "list":
		return RenderWorkflowList(m.wfStore)

	case "create":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/workflow create <path.json>")
		}
		w, err := m.wfStore.CreateFromFile(args[1])
		if err != nil {
			return ErrorStyle.Render("  Failed to create workflow: ") + err.Error()
		}
		if saveErr := m.wfStore.Save(); saveErr != nil {
			return ErrorStyle.Render("  Failed to save: ") + saveErr.Error()
		}
		return RenderWorkflowCreated(w)

	case "run":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/workflow run <name>")
		}
		logs, err := m.wfStore.Run(args[1])
		if err != nil {
			return ErrorStyle.Render("  " + err.Error())
		}
		if saveErr := m.wfStore.Save(); saveErr != nil {
			return ErrorStyle.Render("  Failed to save: ") + saveErr.Error()
		}
		return RenderWorkflowRunning(args[1], logs)

	case "stop":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/workflow stop <name>")
		}
		if err := m.wfStore.Stop(args[1]); err != nil {
			return ErrorStyle.Render("  " + err.Error())
		}
		if saveErr := m.wfStore.Save(); saveErr != nil {
			return ErrorStyle.Render("  Failed to save: ") + saveErr.Error()
		}
		return RenderWorkflowStopped(args[1])

	case "show":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/workflow show <name>")
		}
		w := m.wfStore.Get(args[1])
		if w == nil {
			return ErrorStyle.Render("  Workflow not found: ") + args[1]
		}
		return RenderWorkflowShow(w, m.width)

	case "remove":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/workflow remove <name>")
		}
		name := args[1]
		if !m.wfStore.Remove(name) {
			return ErrorStyle.Render("  Workflow not found: ") + name
		}
		if saveErr := m.wfStore.Save(); saveErr != nil {
			return ErrorStyle.Render("  Failed to save: ") + saveErr.Error()
		}
		return RenderWorkflowRemoved(name)

	case "edit":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/workflow edit <name>")
		}
		return DimStyle.Render("  Tip: use ") + CommandStyle.Render(":e ~/.nickai/workflows.json") +
			DimStyle.Render(" in COMMAND mode (press Esc then :)")

	default:
		return ErrorStyle.Render("  Unknown subcommand: ") + sub +
			"\n" + DimStyle.Render("  Usage: /workflow <list|create|run|stop|show|remove|edit>")
	}
}

// handleLogs processes /logs command.
func (m *Model) handleLogs(args []string) string {
	if len(args) == 0 {
		return ErrorStyle.Render("  Usage: ") +
			CommandStyle.Render("/logs <workflow-name>")
	}
	w := m.wfStore.Get(args[0])
	if w == nil {
		return ErrorStyle.Render("  Workflow not found: ") + args[0]
	}
	return RenderLogs(w)
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

// riskPromptFromLimits builds the risk info string for the AI system prompt.
func riskPromptFromLimits(limits *risk.RiskLimits) string {
	if limits == nil || limits.IsEmpty() {
		return ""
	}
	var parts []string
	parts = append(parts, "RISK GUARDRAILS ARE ACTIVE. Before placing trades, be aware of these limits:")
	if limits.MaxOrderValue > 0 {
		parts = append(parts, fmt.Sprintf("- Maximum single order value: $%.0f", limits.MaxOrderValue))
	}
	if limits.MaxPositionPct > 0 {
		parts = append(parts, fmt.Sprintf("- Maximum position size: %.0f%% of portfolio", limits.MaxPositionPct))
	}
	if limits.DailyLossPct > 0 {
		parts = append(parts, fmt.Sprintf("- Daily loss limit: %.0f%% (all trades blocked if exceeded)", limits.DailyLossPct))
	}
	parts = append(parts, "If a trade is rejected by risk limits, explain the reason to the user and suggest an alternative that fits within limits.")
	return strings.Join(parts, "\n")
}

// waitForJournalEntry listens for journal entries from tool executors.
func waitForJournalEntry(ch <-chan journal.JournalEntry) tea.Cmd {
	return func() tea.Msg {
		entry, ok := <-ch
		if !ok {
			return nil
		}
		return journalEntryMsg{entry: entry}
	}
}

// handleTheme processes /theme command.
func (m *Model) handleTheme(args []string) string {
	if len(args) == 0 {
		// Show available themes.
		var lines []string
		lines = append(lines, SecondaryStyle.Render("  Available Themes\n"))
		current := m.cfg.Theme
		if current == "" {
			current = "default"
		}
		for name := range Themes {
			indicator := "  "
			if name == current {
				indicator = BrandStyle.Render("● ")
			}
			lines = append(lines, "  "+indicator+CommandStyle.Render(name))
		}
		lines = append(lines, "")
		lines = append(lines, DimStyle.Render("  Usage: ")+CommandStyle.Render("/theme <name>"))
		return strings.Join(lines, "\n")
	}

	name := strings.ToLower(args[0])
	t, ok := Themes[name]
	if !ok {
		var names []string
		for n := range Themes {
			names = append(names, n)
		}
		return ErrorStyle.Render("  Unknown theme: ") + name + "\n" +
			DimStyle.Render("  Available: "+strings.Join(names, ", "))
	}

	ApplyTheme(t)
	m.cfg.Theme = name
	_ = m.cfg.Save()

	return BotMsgStyle.Render("nick: ") + "Theme set to " + BrandStyle.Render(name) + "."
}

// handleModel processes /model command.
func (m *Model) handleModel(args []string) string {
	if len(args) == 0 {
		// Show available models.
		var lines []string
		lines = append(lines, SecondaryStyle.Render("  Available Models\n"))
		currentModel := "claude-sonnet"
		if m.agent != nil {
			currentModel = m.agent.ModelID()
		}
		for _, opt := range ai.AvailableModels {
			indicator := "  "
			if opt.ID == currentModel {
				indicator = BrandStyle.Render("● ")
			}
			freeTag := ""
			if opt.Free {
				freeTag = lipgloss.NewStyle().Foreground(ColorPrimary).Render(" [FREE]")
			}
			lines = append(lines, "  "+indicator+CommandStyle.Render(padRight(opt.ID, 18))+
				DimStyle.Render(opt.Name)+freeTag)
		}
		lines = append(lines, "")
		lines = append(lines, DimStyle.Render("  Usage: ")+CommandStyle.Render("/model <id>"))
		return strings.Join(lines, "\n")
	}

	modelID := strings.ToLower(args[0])

	if m.agent == nil {
		// Create agent if we have any key.
		anthKey := m.cfg.AnthropicKeyOrEnv()
		mmKey := m.cfg.MinimaxKeyOrEnv()
		if anthKey == "" && mmKey == "" {
			return ErrorStyle.Render("  No API keys configured.") + "\n" +
				DimStyle.Render("  Set one with ") +
				CommandStyle.Render("/config set anthropic_key <key>") +
				DimStyle.Render(" or ") +
				CommandStyle.Render("/config set minimax_key <key>")
		}
		if anthKey != "" {
			m.agent = ai.NewAgent(m.client, anthKey, m.toolRegistry)
		} else {
			m.agent = ai.NewAgent(m.client, "", m.toolRegistry)
		}
		if mmKey != "" {
			m.agent.SetMinimaxKey(mmKey)
		}
	}

	if err := m.agent.SetModel(modelID); err != nil {
		return ErrorStyle.Render("  " + err.Error())
	}

	m.cfg.Model = modelID
	_ = m.cfg.Save()

	// Find model name for display.
	name := modelID
	for _, opt := range ai.AvailableModels {
		if opt.ID == modelID {
			name = opt.Name
			break
		}
	}

	return BotMsgStyle.Render("nick: ") + "Switched to " + BrandStyle.Render(name) + "."
}

// --- Session persistence ---

// historyFilePath returns ~/.nickai/input_history.json.
func historyFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/.nickai/input_history.json"
}

// loadInputHistory reads persisted input history from disk.
func loadInputHistory() []string {
	path := historyFilePath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var history []string
	if err := json.Unmarshal(data, &history); err != nil {
		return nil
	}
	// Keep last 100.
	if len(history) > 100 {
		history = history[len(history)-100:]
	}
	return history
}

// saveInputHistory persists input history to disk.
func saveInputHistory(history []string) {
	path := historyFilePath()
	if path == "" {
		return
	}
	// Keep last 100.
	if len(history) > 100 {
		history = history[len(history)-100:]
	}
	data, err := json.Marshal(history)
	if err != nil {
		return
	}
	_ = os.MkdirAll(home()+"/.nickai", 0700)
	_ = os.WriteFile(path, data, 0600)
}

func home() string {
	h, _ := os.UserHomeDir()
	return h
}

func (m *Model) updateViewport() {
	if len(m.messages) == 0 {
		content := RenderWelcome(m.width, m.client.IsConfigured())
		m.viewContent = content
		m.viewport.SetContent(content)
		return
	}

	// If loading, update the spinner on the last message.
	if m.loading && len(m.messages) > 0 && !m.messages[len(m.messages)-1].isUser {
		frame := spinnerFrames[m.loadingFrame%len(spinnerFrames)]
		text := thinkingTexts[(m.loadingFrame/15)%len(thinkingTexts)]
		m.messages[len(m.messages)-1].content = BotMsgStyle.Render("nick: ") + frame + " " + text
	}

	welcome := RenderWelcome(m.width, m.client.IsConfigured())
	var parts []string
	parts = append(parts, welcome)
	for _, msg := range m.messages {
		// Apply left-border accent bars.
		if msg.isUser {
			parts = append(parts, UserMsgBar(msg.content))
		} else {
			parts = append(parts, BotMsgBar(msg.content))
		}
		parts = append(parts, "")
	}
	content := strings.Join(parts, "\n")
	m.viewContent = content
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

// apiLoadingText returns the loading message for a given command type.
func apiLoadingText(t commands.CommandType) string {
	switch t {
	case commands.TypePrice:
		return "Fetching prices..."
	case commands.TypeStatus:
		return "Loading portfolio..."
	case commands.TypeOrders:
		return "Loading orders..."
	case commands.TypeBuy, commands.TypeSell:
		return "Executing trade..."
	case commands.TypeSnapshot:
		return "Loading snapshot..."
	case commands.TypeMarket:
		return "Fetching market data..."
	case commands.TypePnl:
		return "Calculating P&L..."
	case commands.TypeHistory:
		return "Loading trade history..."
	case commands.TypeChart:
		return "Generating chart..."
	default:
		return "Loading..."
	}
}

// --- View ---

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing NickAI..."
	}

	// Boot sequence animation.
	if m.booting {
		return m.renderBootSequence()
	}

	// ── Top bar: logo + live ticker ──
	topLeft := lipgloss.NewStyle().
		Foreground(ColorPrimary).Bold(true).
		Render("NickAI")
	topLeft += DimStyle.Render(" v" + Version)

	var tickerStr string
	if len(m.tickerPrices) > 0 && m.width > 60 {
		var tickerParts []string
		for _, p := range m.tickerPrices {
			sym := strings.TrimSuffix(p.Symbol, "USDT")
			tickerParts = append(tickerParts,
				DimStyle.Render(sym+" ")+BrandStyle.Render(formatPrice(p.Price)))
		}
		tickerStr = strings.Join(tickerParts, "  ")
	}

	topBar := lipgloss.NewStyle().
		Background(lipgloss.Color("#0D0D1A")).
		Foreground(ColorPrimary).
		Bold(true).
		Width(m.width).
		Padding(0, 1).
		Render(topLeft + "    " + tickerStr)

	inputBar := m.renderInputBar()

	base := topBar + "\n" + m.viewport.View() + "\n" + inputBar

	// Slash command suggestions overlay.
	if len(m.completionCandidates) > 0 && m.vimMode == ModeInsert {
		base = m.composeSuggestions(base)
	}

	// Composite overlay dialog if active.
	if m.dialog.Active != DialogNone {
		var dialog string
		switch m.dialog.Active {
		case DialogHelp:
			dialog = renderHelpDialog(m.width, m.height)
		case DialogTheme:
			dialog = renderThemeDialog(m.dialog.Cursor, m.width, m.height)
		case DialogModel:
			dialog = renderModelDialog(m.dialog.Cursor, m.agent, m.width, m.height)
		case DialogPalette:
			dialog = renderPaletteDialog(m.dialog.Cursor, m.dialog.ScrollOffset, m.dialog.Filter, m.dialog.FilteredList, m.width, m.height)
		}
		if dialog != "" {
			return compositeOverlay(base, dialog, m.width, m.height)
		}
	}

	return base
}

// renderBootSequence renders the animated boot screen.
func (m Model) renderBootSequence() string {
	var lines []string

	// Add top padding to roughly center vertically.
	topPad := (m.height - 18) / 3 // 18 ≈ content lines
	if topPad < 1 {
		topPad = 1
	}
	for i := 0; i < topPad; i++ {
		lines = append(lines, "")
	}

	logoStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	checkStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
	pad := "   " // left padding for all content

	// Phase 1: Logo lines appearing one by one (frames 1..6).
	visibleLogoLines := m.bootFrame
	if visibleLogoLines > len(bootLogo) {
		visibleLogoLines = len(bootLogo)
	}
	for i := 0; i < visibleLogoLines; i++ {
		lines = append(lines, pad+logoStyle.Render(bootLogo[i]))
	}

	// Pad remaining logo space so content below doesn't jump.
	for i := visibleLogoLines; i < len(bootLogo); i++ {
		lines = append(lines, "")
	}
	lines = append(lines, "")

	// Phase 2: Tagline types out character by character.
	taglineStart := len(bootLogo)
	if m.bootFrame > taglineStart {
		charsVisible := m.bootFrame - taglineStart
		if charsVisible > len(m.bootTagline) {
			charsVisible = len(m.bootTagline)
		}
		taglineText := m.bootTagline[:charsVisible]
		cursor := ""
		if charsVisible < len(m.bootTagline) {
			cursor = lipgloss.NewStyle().Foreground(ColorSecondary).Render("█")
		}
		lines = append(lines, pad+DimStyle.Render("  \"")+DimStyle.Render(taglineText)+cursor+DimStyle.Render("\""))
	} else {
		lines = append(lines, "")
	}
	lines = append(lines, "")

	// Phase 3: Boot checks (after tagline completes).
	checksStart := taglineStart + len(m.bootTagline) + 1
	type bootCheck struct {
		label string
		ok    bool
	}

	hasAPI := m.client.IsConfigured()
	hasAnthropicKey := m.cfg.AnthropicKey != ""
	hasMCP := m.mcpManager != nil && m.mcpManager.ConnectionCount() > 0

	checks := []bootCheck{
		{"API connected", hasAPI},
		{"Paper trading active", hasAPI},
		{"AI agent ready", hasAnthropicKey},
	}
	if hasMCP {
		checks = append(checks, bootCheck{
			fmt.Sprintf("MCP servers (%d)", m.mcpManager.ConnectionCount()), true,
		})
	}
	checks = append(checks, bootCheck{"Vim mode enabled", true})

	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9900"))

	for i, check := range checks {
		checkFrame := checksStart + i*2
		if m.bootFrame >= checkFrame+1 {
			if check.ok {
				lines = append(lines, pad+"  "+checkStyle.Render("✓ "+check.label))
			} else {
				lines = append(lines, pad+"  "+warnStyle.Render("○ "+check.label)+DimStyle.Render("  (not configured)"))
			}
		} else if m.bootFrame >= checkFrame {
			spinIdx := m.bootFrame % len(spinnerFrames)
			spinner := lipgloss.NewStyle().Foreground(ColorSecondary).Render(spinnerFrames[spinIdx])
			lines = append(lines, pad+"  "+spinner+" "+DimStyle.Render(check.label+"..."))
		}
	}

	// Phase 4: Ready message — context-aware.
	readyFrame := checksStart + len(checks)*2
	if m.bootFrame >= readyFrame {
		lines = append(lines, "")
		if !hasAPI {
			lines = append(lines, pad+"  "+DimStyle.Render("Run ")+CommandStyle.Render("/config init")+DimStyle.Render(" to get started."))
		} else {
			lines = append(lines, pad+DimStyle.Render("  Ready. Type /help or just ask me anything."))
		}
	}

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(content)
}


// statusIndicators returns the right-aligned status string (API dot, model, alerts).
func (m Model) statusIndicators() string {
	connDot := lipgloss.NewStyle().Foreground(ColorPrimary).Render("●")
	if !m.client.IsConfigured() {
		connDot = DimStyle.Render("○")
	}

	modelName := ""
	if m.agent != nil {
		modelName = DimStyle.Render(" ┃ ") +
			lipgloss.NewStyle().Foreground(ColorSecondary).Render(m.agent.ModelID())
	}

	alertPart := ""
	statusParts := []string{}
	if len(m.alerts) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("⚡%d", len(m.alerts)))
	}
	if len(m.triggers) > 0 {
		statusParts = append(statusParts, fmt.Sprintf("⏱%d", len(m.triggers)))
	}
	activeStrats := 0
	for _, s := range m.strategies {
		if s.Status == "active" {
			activeStrats++
		}
	}
	if activeStrats > 0 {
		statusParts = append(statusParts, fmt.Sprintf("📊%d", activeStrats))
	}
	if m.riskLimits != nil && !m.riskLimits.IsEmpty() {
		statusParts = append(statusParts, "🛡")
	}
	if len(statusParts) > 0 {
		alertPart = DimStyle.Render(" ┃ ") +
			lipgloss.NewStyle().Foreground(ColorWarning).Render(strings.Join(statusParts, " "))
	}

	return connDot + modelName + alertPart
}

// renderInputBar renders the bottom input area based on vim mode.
func (m Model) renderInputBar() string {
	var borderColor lipgloss.Color
	var content string

	switch m.vimMode {
	case ModeInsert:
		// No badge — clean input with status info right-aligned.
		borderColor = ColorDim
		inputView := m.textInput.View()
		status := m.statusIndicators()
		statusWidth := lipgloss.Width(status)
		inputWidth := m.width - statusWidth - 4
		if inputWidth < 20 {
			inputWidth = 20
		}
		leftSide := lipgloss.NewStyle().Width(inputWidth).Render(inputView)
		content = leftSide + status

	case ModeNormal:
		borderColor = ColorPrimary
		badge := lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Render(" NORMAL ")

		hints := DimStyle.Render("i") + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(":insert") +
			DimStyle.Render("  :") + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("cmd") +
			DimStyle.Render("  /") + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("search") +
			DimStyle.Render("  q") + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(":quit")

		status := m.statusIndicators()
		middle := hints + "  " + status
		middleWidth := m.width - 14
		if middleWidth < 0 {
			middleWidth = 0
		}
		rightAligned := lipgloss.NewStyle().Width(middleWidth).Align(lipgloss.Right).Render(middle)
		content = badge + " " + rightAligned

	case ModeCommand:
		borderColor = ColorSecondary
		badge := lipgloss.NewStyle().
			Background(ColorSecondary).
			Foreground(ColorWhite).
			Bold(true).
			Render(" COMMAND ")

		bufferStyle := lipgloss.NewStyle().Foreground(ColorWhite)
		content = badge + " " + bufferStyle.Render(":"+m.commandBuffer) +
			lipgloss.NewStyle().Foreground(ColorWhite).Render("█")

	case ModeSearch:
		borderColor = ColorWarning
		badge := lipgloss.NewStyle().
			Background(ColorWarning).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Render(" SEARCH ")

		bufferStyle := lipgloss.NewStyle().Foreground(ColorWarning)
		content = badge + " " + bufferStyle.Render("/"+m.searchBuffer) +
			lipgloss.NewStyle().Foreground(ColorWarning).Render("█")

	case ModeConfirm:
		borderColor = ColorWarning
		badge := lipgloss.NewStyle().
			Background(ColorWarning).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Render(" CONFIRM ")
		hint := lipgloss.NewStyle().Foreground(ColorWarning).
			Render("  y: confirm  n: cancel")
		content = badge + hint
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(borderColor).
		Width(m.width).
		Padding(0, 1).
		Render(content)
}
