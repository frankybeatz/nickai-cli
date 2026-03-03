package ui

import (
	"fmt"
	"os"
	"regexp"
	"sort"
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
	"github.com/nickai/cli/internal/config"
	"github.com/nickai/cli/internal/credential"
	"github.com/nickai/cli/internal/guidance"
	"github.com/nickai/cli/internal/journal"
	"github.com/nickai/cli/internal/market"
	"github.com/nickai/cli/internal/mcp"
	"github.com/nickai/cli/internal/memory"
	"github.com/nickai/cli/internal/node"
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
// oscResponseRe matches full OSC 10/11 terminal color query responses
// (e.g. ESC]11;rgb:XXXX/XXXX/XXXX BEL) delivered as a single key event.
var oscResponseRe = regexp.MustCompile(`\x1b?\]1[01];rgb:[0-9a-fA-F/]+\x07?`)

// oscLeakRe matches OSC color fragments that leak into the text input buffer.
// Requires ESC or ] prefix to avoid false positives on valid hex user input.
var oscLeakRe = regexp.MustCompile(`[\x1b\]]+[01]?1?;?r?g?b?:?[0-9a-fA-F]{0,4}/?[0-9a-fA-F]{4}/[0-9a-fA-F]{4}\x07?`)

// ansiEscRe matches any ANSI escape sequence that could leak as a key event:
// CSI (ESC[...), OSC (ESC]...), DCS (ESCP...), and other standard sequences.
var ansiEscRe = regexp.MustCompile(`\x1b[\[\]PX^_][^\x1b]*`)

// sanitizeStreamText strips terminal control/escape sequences from streamed text
// so partial chunks cannot inject background colors or cursor control into the UI.
func sanitizeStreamText(s string) string {
	s = oscLeakRe.ReplaceAllString(oscResponseRe.ReplaceAllString(s, ""), "")
	s = ansiEscRe.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		// Keep newline/tab, drop other control characters.
		if (r >= 0 && r < 32 && r != '\n' && r != '\t') || r == 127 {
			return -1
		}
		return r
	}, s)
}

var knownCommands = []string{
	"/help", "/status", "/orders", "/agents", "/templates",
	"/buy", "/sell", "/price", "/watch", "/snapshot",
	"/market", "/pnl", "/history", "/credential", "/workflow",
	"/logs", "/man", "/config", "/clear", "/quit",
	"/alert", "/chart", "/theme", "/model", "/mcp", "/trigger",
	"/risk", "/strategy", "/notify", "/analytics", "/analyze", "/auto",
	"/backtest", "/bt", "/polymarket", "/pm", "/guide",
	"/memory", "/mem", "/consensus", "/con",
	// Multi-vertical commands.
	"/connect", "/balances", "/bal", "/positions", "/pos",
	"/markets", "/bet", "/wallet", "/swap", "/gas",
	"/stock", "/screen", "/odds", "/lines", "/funding",
	"/dashboard", "/dash",
	"/vibe",
	"/export",
	"/plugin", "/plugins",
}

var knownSymbols = []string{
	"BTC", "ETH", "SOL", "DOGE", "ADA", "AVAX", "LINK", "DOT", "XRP",
}

var symbolCommands = map[string]bool{
	"/price": true, "/buy": true, "/sell": true,
	"/watch": true, "/chart": true, "/alert": true,
	"/trigger": true, "/analyze": true, "/backtest": true,
	"/consensus": true, "/stock": true, "/funding": true,
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
	isTrade bool // when true, refresh portfolio context after displaying
}

// backtestDoneMsg carries the result of an async backtest run.
type backtestDoneMsg struct {
	result *backtest.Result
	err    error
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

// statusFlashExpiredMsg signals the status flash should clear.
type statusFlashExpiredMsg struct{}

// mcpConnectedMsg is sent when background MCP connection completes.
type mcpConnectedMsg struct {
	manager *mcp.ClientManager
}

// mcpHealthMsg triggers a periodic MCP health check + reconnect.
type mcpHealthMsg struct{}

// mcpHealthResultMsg carries the result of a health check cycle.
type mcpHealthResultMsg struct {
	removed   int
	recovered int
}

// tickerFetchMsg signals time to fetch new ticker data.
type tickerFetchMsg struct{}

// tickerUpdateMsg carries refreshed ticker prices.
type tickerUpdateMsg struct {
	prices []api.Price
}

// wsPriceMsg carries a live price update from the Binance websocket.
type wsPriceMsg struct {
	symbol string
	price  float64
}

// wsDisconnectedMsg signals the websocket has disconnected (for logging/fallback).
type wsDisconnectedMsg struct{}

// watchTickMsg triggers periodic refresh in watch mode.
type watchTickMsg struct{}

// watchDataMsg carries refreshed watch data.
type watchDataMsg struct {
	prices  []api.Price
	history map[string][]float64
}

// consensusDoneMsg carries the result of an async consensus run.
type consensusDoneMsg struct {
	result *ai.ConsensusResult
	err    error
}

// Braille spinner frames.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Thinking text variants.
var thinkingTexts = []string{"Thinking...", "Analyzing...", "Reasoning...", "Almost there..."}

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
	searchMatches []int  // line indices of all matches
	searchCurrent int    // current match index (for n/N)
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
	historyIndex int    // -1 means not browsing history
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
	pendingTrade         *api.PlaceOrderRequest // manual /buy /sell
	pendingAITrade       *tools.ConfirmRequest  // AI-initiated place_order
	pendingStrategySlice *strategy.TWAPStrategy // TWAP slice awaiting confirmation
	pendingSlicePrice    float64                // price for pending strategy slice
	pendingRationale     string                 // last AI message for journal capture

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

	// Stream origin for AI-routed commands (for next-step hints).
	streamOrigin string

	// Dashboard mode.
	dashboardMode bool

	// Guidance context for smart next-step hints.
	guidanceCtx guidance.StageContext
	// Last detected journey stage (used for stage-up milestone events).
	lastJourneyStage guidance.Stage

	// Cached portfolio for top bar display.
	cachedPortfolio *api.Portfolio

	// Cached trade count (updated on ticker, avoids blocking API calls in render).
	cachedTradeCount    int
	cachedHasAnalyzed   bool
	cachedHasBacktested bool

	// Last backtest result (for /export backtest and /backtest analyze).
	lastBacktestResult *backtest.Result

	// Node client (for /node commands).
	nodeClient *node.Client

	// Cached welcome screen content (avoids re-rendering + random flicker on every frame).
	cachedWelcome string
	welcomeDirty  bool

	// Force-quit: second Ctrl+C exits immediately.
	quitAttempts int

	// Ticker direction tracking.
	prevTickerPrices map[string]float64

	// Websocket live price streaming.
	binanceWS *api.BinanceWS
	wsPriceCh chan wsPriceMsg
	wsActive  bool

	// Watch mode.
	watchMode    bool
	watchSymbols []string
	watchPrices  []api.Price
	watchHistory map[string][]float64

	// Status flash (brief confirmations).
	statusFlash       string
	statusFlashExpiry time.Time

	// Recent commands ring buffer (last 3).
	recentCommands []string

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
	memoryStore  *memory.Store
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

	// MCP servers connect in the background (see mcpConnectedMsg handler).
	var mcpMgr *mcp.ClientManager

	var agent *ai.Agent
	anthKey := cfg.AnthropicKeyOrEnv()
	mmKey := cfg.MinimaxKeyOrEnv()
	orKey := cfg.DataKeyOrEnv("openrouter")
	if anthKey != "" || mmKey != "" || orKey != "" {
		agent = ai.NewAgent(client, anthKey, registry, cfg.Vibe)
		if mmKey != "" {
			agent.SetMinimaxKey(mmKey)
		}
		if orKey != "" {
			agent.SetOpenRouterKey(orKey)
		}
		if cfg.Model != "" {
			_ = agent.SetModel(cfg.Model)
		}
	} else {
		ti.Placeholder = "No API key configured — type /config set to get started"
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

	// Load AI memory store.
	memStore, _ := memory.Load()

	// Update risk info on agent.
	if agent != nil && riskLimits != nil && !riskLimits.IsEmpty() {
		agent.SetRiskInfo(riskPromptFromLimits(riskLimits))
	}

	// Inject memory context into agent.
	if agent != nil && memStore != nil {
		if info := memStore.ForPrompt(500); info != "" {
			agent.SetMemoryInfo("Here are memories from previous sessions:\n" + info)
		}
	}

	// Inject portfolio context into agent on boot.
	if agent != nil && client.IsConfigured() {
		if portfolio, err := client.GetPortfolio(); err == nil {
			summary := buildPortfolioSummary(portfolio)
			if summary != "" {
				agent.SetPortfolioContext(summary)
			}
		}
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

	model := Model{
		textInput:     ti,
		vimMode:       ModeInsert,
		booting:       true,
		bootFrame:     0,
		bootTagline:   startupTaglines[time.Now().UnixNano()%int64(len(startupTaglines))],
		bootStartTime: time.Now(),
		historyIndex:  -1,
		inputHistory:  inputHistory,
		alerts:        savedAlerts,
		triggers:      savedTriggers,
		riskLimits:    riskLimits,
		strategies:    savedStrategies,
		notifyConfig:  notifyCfg,
		automations:   savedAutomations,
		cfg:           cfg,
		client:        client,
		agent:         agent,
		toolRegistry:  registry,
		mcpManager:    mcpMgr,
		credStore:     credStore,
		wfStore:       wfStore,
		memoryStore:   memStore,
	}
	model.updateJourneyContext(false)
	return model
}

// cleanup shuts down MCP connections, saves history, and summarizes session.
func (m Model) cleanup() {
	if m.binanceWS != nil {
		m.binanceWS.StopWebSocket()
	}
	if m.mcpManager != nil {
		m.mcpManager.CloseAll()
	}
	saveInputHistory(m.inputHistory)
	m.summarizeSession()
}

// summarizeSession scans messages for traded symbols and saves context to memory.
func (m Model) summarizeSession() {
	if m.memoryStore == nil {
		return
	}
	// Collect symbols mentioned in user messages.
	symbolSet := map[string]bool{}
	for _, msg := range m.messages {
		if !msg.isUser {
			continue
		}
		upper := strings.ToUpper(msg.content)
		for _, sym := range knownSymbols {
			if strings.Contains(upper, sym) {
				symbolSet[sym] = true
			}
		}
	}
	if len(symbolSet) == 0 {
		return
	}
	var syms []string
	for s := range symbolSet {
		syms = append(syms, s)
	}
	sort.Strings(syms)
	entry := memory.Entry{
		ID:         fmt.Sprintf("ses-%d", time.Now().UnixMilli()),
		Type:       memory.TypeContext,
		Content:    fmt.Sprintf("Session discussed: %s", strings.Join(syms, ", ")),
		Tags:       syms,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		Score:      3,
	}
	m.memoryStore.Add(entry)
	m.memoryStore.Prune(50)
	_ = m.memoryStore.Save()
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
	// Connect MCP servers in background so the app starts instantly.
	registry := m.toolRegistry
	riskLimitsPtr := m.riskLimits
	riskFn := func() *risk.RiskLimits { return riskLimitsPtr }
	cmds = append(cmds, func() tea.Msg {
		mcpCfg, err := mcp.LoadMCPConfig()
		if err != nil || len(mcpCfg.MCPServers) == 0 {
			return mcpConnectedMsg{manager: nil}
		}
		mgr := mcp.NewClientManager(Version)
		mgr.ConnectAll(mcpCfg)
		mgr.RegisterTools(registry, mcp.RiskLimitsFunc(riskFn))
		return mcpConnectedMsg{manager: mgr}
	})
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
		totalFrames := checksEnd + 2      // + ready message + buffer

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
				// Start Binance websocket for real-time price updates.
				priceCh := make(chan wsPriceMsg, 50)
				m.wsPriceCh = priceCh
				ws, err := api.StartWebSocket([]string{"BTC", "ETH", "SOL"}, func(symbol string, price float64) {
					select {
					case priceCh <- wsPriceMsg{symbol: symbol, price: price}:
					default:
					}
				})
				if err == nil {
					m.binanceWS = ws
					m.wsActive = true
					bootCmds = append(bootCmds, waitForWSPrice(priceCh))
				}
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
			token := sanitizeStreamText(msg.token)
			lastMsg := &m.messages[len(m.messages)-1]
			if m.loading {
				// First token: replace spinner with nick prefix.
				m.loading = false
				lastMsg.content = BotMsgStyle.Render("nick: ") + token
			} else {
				lastMsg.content += token
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
		origin := m.streamOrigin
		// Clean up any escape sequences that leaked into the text input
		// buffer during streaming (e.g. OSC color responses, ANSI fragments).
		if val := m.textInput.Value(); val != "" {
			cleaned := oscLeakRe.ReplaceAllString(oscResponseRe.ReplaceAllString(val, ""), "")
			cleaned = ansiEscRe.ReplaceAllString(cleaned, "")
			if cleaned != val {
				m.textInput.SetValue(cleaned)
				if cleaned == "" {
					m.textInput.CursorStart()
				} else {
					m.textInput.CursorEnd()
				}
			}
		}
			if len(m.messages) > 0 && !m.messages[len(m.messages)-1].isUser {
				if msg.err != nil {
					m.messages[len(m.messages)-1].content = ErrorStyle.Render("  AI error: ") + msg.err.Error()
				} else {
					finalContent := sanitizeStreamText(msg.finalContent)
					rendered := renderMarkdown(finalContent, m.width-8)
					m.messages[len(m.messages)-1].content = BotMsgStyle.Render("nick:") + "\n" + rendered
					// Append origin-based next-step hints.
					if hints := m.streamOriginHints(m.streamOrigin); hints != "" {
					m.messages[len(m.messages)-1].content += hints
				}
				if origin == "analyze" {
					m.cachedHasAnalyzed = true
					m.updateJourneyContext(true)
				}
				if origin == "backtest" {
					m.cachedHasBacktested = true
					m.updateJourneyContext(true)
				}
			}
		}
		m.streamOrigin = ""
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

	case backtestDoneMsg:
		m.loading = false
		var content string
		if msg.err != nil {
			content = ErrorStyle.Render("  Backtest failed: ") + msg.err.Error()
		} else {
			m.lastBacktestResult = msg.result
			m.cachedHasBacktested = true
			m.updateJourneyContext(true)
			content = RenderBacktestCard(msg.result)
		}
		if len(m.messages) > 0 && !m.messages[len(m.messages)-1].isUser {
			m.messages[len(m.messages)-1].content = content
		}
		m.updateViewport()
		return m, nil

	case nodeConnectResultMsg:
		m.loading = false
		var content string
		if msg.err != nil {
			content = BotMsgStyle.Render("nick: ") + ErrorStyle.Render("Node connection failed: "+msg.err.Error())
		} else {
			m.nodeClient = msg.client
			content = BotMsgStyle.Render("nick: ") + BrandStyle.Render("Connected to Nick Node!") +
				fmt.Sprintf(" v%s, uptime %s", msg.version, formatDuration(time.Duration(msg.uptime)*time.Second))
		}
		if len(m.messages) > 0 && !m.messages[len(m.messages)-1].isUser {
			m.messages[len(m.messages)-1].content = content
		}
		m.updateViewport()
		return m, nil

	case apiResponseMsg:
		m.loading = false
		// Replace the last loading message with the actual result.
		if len(m.messages) > 0 && !m.messages[len(m.messages)-1].isUser {
			m.messages[len(m.messages)-1].content = msg.content
		}
		// Refresh portfolio context after trades so AI knows the new state.
		if msg.isTrade && m.agent != nil && m.client.IsConfigured() {
			m.cachedTradeCount++
			if portfolio, err := m.client.GetPortfolio(); err == nil {
				m.cachedPortfolio = portfolio
				if summary := buildPortfolioSummary(portfolio); summary != "" {
					m.agent.SetPortfolioContext(summary)
				}
			}
			m.updateJourneyContext(true)
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

					case automation.RulePortfolio:
						portfolio, err := client.GetPortfolio()
						if err != nil || portfolio == nil {
							continue
						}
						var metricVal float64
						switch r.MetricName {
						case "total_value":
							metricVal = portfolio.TotalValue
						case "drawdown_pct":
							metricVal = (100000 - portfolio.TotalValue) / 100000 * 100
						case "cash":
							metricVal = portfolio.Cash
						case "cash_pct":
							if portfolio.TotalValue > 0 {
								metricVal = portfolio.Cash / portfolio.TotalValue * 100
							}
						default:
							continue
						}
						fired := false
						if r.Operator == ">" && metricVal > r.Threshold {
							fired = true
						}
						if r.Operator == "<" && metricVal < r.Threshold {
							fired = true
						}
						if fired {
							return autoRuleFiredMsg{rule: r, price: metricVal}
						}

					case automation.RuleIndicator:
						if r.Symbol == "" || len(r.IndicatorConditions) == 0 {
							continue
						}
						baseSymbol := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(r.Symbol, "USDT"), "USDC"), "USD")
						candles, err := market.FetchKlines(baseSymbol, "1h", 60)
						if err != nil || len(candles) < 30 {
							continue
						}
						closePrices := market.ClosePrices(candles)
						snap := computeIndicatorSnapshot(closePrices)
						var prevSnap *automation.IndicatorSnapshot
						if len(closePrices) > 1 {
							ps := computeIndicatorSnapshot(closePrices[:len(closePrices)-1])
							prevSnap = &ps
						}
						if automation.EvalIndicatorConditions(r.IndicatorConditions, snap, prevSnap) {
							prices, err := client.GetPrices([]string{baseSymbol})
							if err != nil || len(prices) == 0 {
								continue
							}
							return autoRuleFiredMsg{rule: r, price: prices[0].Price}
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

	case consensusDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.messages[len(m.messages)-1] = message{
				content: ErrorStyle.Render("  Consensus failed: ") + msg.err.Error(),
				isUser:  false,
			}
		} else {
			m.messages[len(m.messages)-1] = message{
				content: RenderConsensusCard(msg.result),
				isUser:  false,
			}
		}
		m.updateViewport()
		return m, nil

	case statusFlashExpiredMsg:
		m.statusFlash = ""
		return m, nil

	case mcpConnectedMsg:
		if msg.manager != nil {
			m.mcpManager = msg.manager
			m.welcomeDirty = true
			m.updatePlaceholder()
			m.updateJourneyContext(true)
		}
		// Start periodic health check (every 5 minutes).
		return m, tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
			return mcpHealthMsg{}
		})

	case mcpHealthMsg:
		mgr := m.mcpManager
		if mgr == nil {
			return m, nil
		}
		return m, func() tea.Msg {
			removed := mgr.HealthCheck()
			recovered := mgr.ReconnectFailed()
			return mcpHealthResultMsg{removed: removed, recovered: recovered}
		}

	case mcpHealthResultMsg:
		if msg.recovered > 0 {
			// Re-register tools from reconnected servers.
			if m.mcpManager != nil && m.toolRegistry != nil {
				m.mcpManager.RegisterTools(m.toolRegistry, mcp.RiskLimitsFunc(func() *risk.RiskLimits {
					return m.riskLimits
				}))
				if m.agent != nil {
					m.agent.SetVibe(m.agent.VibeID())
				}
			}
			m.welcomeDirty = true
			m.updateJourneyContext(true)
		}
		// Re-schedule next health check.
		return m, tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
			return mcpHealthMsg{}
		})

	case wsDisconnectedMsg:
		m.wsActive = false
		m.wsPriceCh = nil
		return m, nil

	case wsPriceMsg:
		if !m.wsActive {
			return m, nil
		}
		if m.prevTickerPrices == nil {
			m.prevTickerPrices = make(map[string]float64)
		}
		found := false
		for i, p := range m.tickerPrices {
			if p.Symbol == msg.symbol {
				m.prevTickerPrices[p.Symbol] = p.Price
				m.tickerPrices[i].Price = msg.price
				found = true
				break
			}
		}
		if !found {
			m.tickerPrices = append(m.tickerPrices, api.Price{Symbol: msg.symbol, Price: msg.price})
		}
		if m.wsPriceCh != nil {
			return m, waitForWSPrice(m.wsPriceCh)
		}
		return m, nil

	case watchTickMsg:
		if !m.watchMode {
			return m, nil
		}
		symbols := m.watchSymbols
		client := m.client
		return m, func() tea.Msg {
			prices, _ := client.GetPrices(symbols)
			history := make(map[string][]float64)
			for _, sym := range symbols {
				if candles, err := market.FetchKlines(sym, "1h", 24); err == nil && len(candles) > 0 {
					prices := make([]float64, len(candles))
					for j, c := range candles {
						prices[j] = c.Close
					}
					history[sym] = prices
				}
			}
			return watchDataMsg{prices: prices, history: history}
		}

	case watchDataMsg:
		m.watchPrices = msg.prices
		m.watchHistory = msg.history
		if m.watchMode {
			return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg { return watchTickMsg{} })
		}
		return m, nil

	case tickerFetchMsg:
		if !m.tickerTicking {
			return m, nil
		}
		if m.wsActive {
			return m, tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
				return tickerFetchMsg{}
			})
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
			// Store previous prices for direction arrows before updating.
			if m.prevTickerPrices == nil {
				m.prevTickerPrices = make(map[string]float64)
			}
			for _, p := range m.tickerPrices {
				m.prevTickerPrices[p.Symbol] = p.Price
			}
			m.tickerPrices = msg.prices
			// Cache portfolio and guidance data for top bar + welcome screen.
			if m.client.IsConfigured() {
				if portfolio, err := m.client.GetPortfolio(); err == nil {
					m.cachedPortfolio = portfolio
				}
			}
			m.refreshGuidanceCaches()
			m.welcomeDirty = true
		}
		if !m.tickerTicking {
			return m, nil
		}
		return m, tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
			return tickerFetchMsg{}
		})

	case tea.KeyMsg:
		// Drop leaked OSC terminal color responses (e.g. ]11;rgb:XXXX/XXXX/XXXX)
		// and any partial hex-color fragments that arrive as separate events.
		// Also drop any ANSI escape sequences that leak as keyboard input.
		s := msg.String()
		if oscResponseRe.MatchString(s) || oscLeakRe.MatchString(s) || ansiEscRe.MatchString(s) {
			return m, nil
		}

		if msg.Type == tea.KeyCtrlC {
			// Restore terminal immediately — no blocking calls before this.
			fmt.Print("\033[?1049l\033[?25h\033[0m")
			// Save non-blocking state.
			saveInputHistory(m.inputHistory)
			if m.binanceWS != nil {
				m.binanceWS.StopWebSocket()
			}
			// Force-exit. MCP cleanup is skipped — OS will reap child processes.
			os.Exit(0)
		}
		// Ignore key input during boot animation.
		if m.booting {
			return m, nil
		}
		// During streaming, block all key input except Ctrl+C (handled above)
		// and Esc (to cancel). This prevents escape sequences from the terminal
		// or streamed content from contaminating the text input buffer.
		if m.streaming {
			if msg.Type == tea.KeyEsc {
				// Allow Esc to cancel streaming gracefully (user may want to stop).
				return m, nil
			}
			return m, nil
		}

		// ── Overlay dialog input (takes priority) ──
		if m.dialog.Active != DialogNone {
			return m.updateDialog(msg)
		}

		// ── Global hotkeys (Ctrl+K, Ctrl+T, Ctrl+O, F1) ──
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
		case "f1":
			m.dialog = DialogState{Active: DialogHelp}
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
		m.welcomeDirty = true
		m.textInput.Width = msg.Width - 10

		headerHeight := 1 // top bar
		inputHeight := 3  // input area
		vpHeight := msg.Height - headerHeight - inputHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, vpHeight)
			m.viewport.MouseWheelEnabled = true
			m.welcomeDirty = true
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = vpHeight
		}
		m.updateViewport()
	}

	// Forward non-key messages to sub-components.
	// During streaming, skip textInput updates to prevent escape sequence
	// contamination. The textInput is effectively frozen while the AI streams.
	var cmds []tea.Cmd
	var cmd tea.Cmd
	if !m.streaming {
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
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

	// ── Top bar: logo + live ticker + portfolio ──
	topLeft := lipgloss.NewStyle().
		Foreground(ColorPrimary).Bold(true).
		Render("NickAI")
	topLeft += DimStyle.Render(" v" + Version)

	var tickerStr string
	if len(m.tickerPrices) > 0 && m.width > 60 {
		var tickerParts []string
		for _, p := range m.tickerPrices {
			sym := strings.TrimSuffix(p.Symbol, "USDT")
			arrow := DimStyle.Render("─")
			if prev, ok := m.prevTickerPrices[p.Symbol]; ok {
				if p.Price > prev {
					arrow = lipgloss.NewStyle().Foreground(ColorPrimary).Render("▲")
				} else if p.Price < prev {
					arrow = lipgloss.NewStyle().Foreground(ColorError).Render("▼")
				}
			}
			tickerParts = append(tickerParts,
				arrow+DimStyle.Render(sym+" ")+BrandStyle.Render(formatPrice(p.Price)))
		}
		tickerStr = strings.Join(tickerParts, "  ")
	}

	// Portfolio value in top bar.
	portfolioStr := formatPortfolioTopBar(m.cachedPortfolio)
	if portfolioStr != "" {
		tickerStr += "    " + portfolioStr
	}

	// Build right-side status indicators.
	statusRight := m.renderStatusRight()
	leftContent := topLeft + "    " + tickerStr
	rightWidth := lipgloss.Width(statusRight)
	leftWidth := m.width - rightWidth - 3
	if leftWidth < 20 {
		leftWidth = 20
	}
	leftPart := lipgloss.NewStyle().Width(leftWidth).Render(leftContent)
	topBar := lipgloss.NewStyle().
		Foreground(ColorWhite).
		Width(m.width).
		Padding(0, 1).
		Render(leftPart + statusRight)

	inputBar := m.renderInputBar()

	var base string
	if m.watchMode {
		watchContent := m.renderWatchView()
		base = topBar + "\n" + watchContent + "\n" + inputBar
	} else if m.dashboardMode {
		dashContent := m.renderDashboard()
		base = topBar + "\n" + dashContent + "\n" + inputBar
	} else {
		base = topBar + "\n" + m.viewport.View() + "\n" + inputBar
	}

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
			dialog = renderThemeDialog(m.dialog.Cursor, m.dialog.ScrollOffset, m.width, m.height)
		case DialogModel:
			dialog = renderModelDialog(m.dialog.Cursor, m.dialog.ScrollOffset, m.agent, m.width, m.height)
		case DialogPalette:
			dialog = renderPaletteDialog(m.dialog.Cursor, m.dialog.ScrollOffset, m.dialog.Filter, m.dialog.FilteredList, m.width, m.height)
		}
		if dialog != "" {
			return compositeOverlay(base, dialog, m.width, m.height)
		}
	}

	return base
}
