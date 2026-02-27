package ui

import (
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
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/commands"
	"github.com/nickai/cli/internal/config"
	"github.com/nickai/cli/internal/credential"
	"github.com/nickai/cli/internal/workflow"
)

// VimMode represents the current editing mode.
type VimMode int

const (
	ModeInsert  VimMode = iota // default — normal chat input
	ModeNormal                 // vim normal mode — navigation
	ModeCommand                // : command line
	ModeSearch                 // / search
)

// editorFinishedMsg is sent when an external editor process completes.
type editorFinishedMsg struct{ err error }

// bootTickMsg advances the boot animation.
type bootTickMsg struct{}

// spinnerTickMsg advances the loading spinner.
type spinnerTickMsg struct{}

// aiResponseMsg carries the result of an async AI call.
type aiResponseMsg struct {
	response string
	err      error
}

// apiResponseMsg carries the result of an async API command.
type apiResponseMsg struct {
	content string
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

	// Data stores.
	cfg       *config.Config
	client    *api.PapernickClient
	agent     *ai.Agent
	credStore *credential.Store
	wfStore   *workflow.Store
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

	var agent *ai.Agent
	if key := cfg.AnthropicKeyOrEnv(); key != "" {
		agent = ai.NewAgent(client, key)
	}

	credStore, _ := credential.Load()
	wfStore, _ := workflow.Load()

	return Model{
		textInput:     ti,
		vimMode:       ModeInsert,
		booting:       true,
		bootFrame:     0,
		bootTagline:   startupTaglines[rand.Intn(len(startupTaglines))],
		bootStartTime: time.Now(),
		cfg:           cfg,
		client:        client,
		agent:         agent,
		credStore:     credStore,
		wfStore:       wfStore,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return bootTickMsg{}
	}))
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
			return m, nil
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
				m.messages[len(m.messages)-1].content = BotMsgStyle.Render("nick: ") + msg.response
			}
		}
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

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		// Ignore key input during boot animation.
		if m.booting {
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
			m.viewport.SetContent(RenderWelcome(msg.Width))
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
		m.textInput.Blur()
		return m, nil

	case tea.KeyEnter:
		input := strings.TrimSpace(m.textInput.Value())
		if input == "" {
			return m, nil
		}
		m.textInput.SetValue("")

		m.messages = append(m.messages, message{
			content: UserMsgStyle.Render("you: ") + input,
			isUser:  true,
		})

		result := commands.Route(input)

		switch result.Type {
		case commands.TypeQuit:
			return m, tea.Quit
		case commands.TypeClear:
			m.messages = nil
			m.viewport.SetContent(RenderWelcome(m.width))
			m.viewport.GotoBottom()
			return m, nil

		case commands.TypeChat:
			// Async AI call with loading spinner.
			if m.agent == nil {
				m.addBotMessage(BotMsgStyle.Render("nick: ") +
					"I need an Anthropic API key to chat. Set one with " +
					CommandStyle.Render("/config set anthropic_key <key>") +
					" or " + DimStyle.Render("export ANTHROPIC_API_KEY=..."))
				m.updateViewport()
				return m, nil
			}
			m.loading = true
			m.loadingFrame = 0
			m.loadingText = "Thinking..."
			m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)
			m.updateViewport()
			agent := m.agent
			userInput := result.Input
			return m, tea.Batch(
				tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
				func() tea.Msg {
					resp, err := agent.Chat(userInput)
					return aiResponseMsg{response: resp, err: err}
				},
			)

		case commands.TypePrice, commands.TypeStatus, commands.TypeOrders,
			commands.TypeBuy, commands.TypeSell, commands.TypeSnapshot,
			commands.TypeMarket, commands.TypePnl, commands.TypeHistory:
			// Async API call with loading spinner.
			m.loading = true
			m.loadingFrame = 0
			m.loadingText = apiLoadingText(result.Type)
			m.addBotMessage(BotMsgStyle.Render("nick: ") + spinnerFrames[0] + " " + m.loadingText)
			m.updateViewport()
			rCopy := result
			width := m.width
			return m, tea.Batch(
				tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return spinnerTickMsg{} }),
				func() tea.Msg {
					output := m.renderResult(rCopy)
					_ = width
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

	// Quit.
	case "q":
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
		return m, tea.Quit
	case "wq":
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
		m.viewport.SetContent(RenderWelcome(m.width))
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
			return RenderStatusLive(m.client, m.width)
		}
		return RenderStatusMock()

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
			return BotMsgStyle.Render("nick: ") + resp
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
		default:
			return ErrorStyle.Render("  Unknown config key: ") + key +
				"\n" + DimStyle.Render("  Valid keys: api_key, url, anthropic_key")
		}

		if err := m.cfg.Save(); err != nil {
			return ErrorStyle.Render("  Failed to save config: ") + err.Error()
		}
		m.client.UpdateConfig(m.cfg)

		if akKey := m.cfg.AnthropicKeyOrEnv(); akKey != "" {
			if m.agent == nil {
				m.agent = ai.NewAgent(m.client, akKey)
			}
		}

		return RenderConfigSet(key, value)

	default:
		return RenderConfigHelp()
	}
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

	order, err := m.client.PlaceOrder(req)
	if err != nil {
		return RenderOrderError(side, symbol, err)
	}

	return RenderOrderConfirmation(order, m.width)
}

func (m *Model) updateViewport() {
	if len(m.messages) == 0 {
		content := RenderWelcome(m.width)
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

	welcome := RenderWelcome(m.width)
	var parts []string
	parts = append(parts, welcome)
	for _, msg := range m.messages {
		parts = append(parts, msg.content)
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

	topBar := lipgloss.NewStyle().
		Background(lipgloss.Color("#0D0D1A")).
		Foreground(ColorPrimary).
		Bold(true).
		Width(m.width).
		Padding(0, 1).
		Render("NickAI Terminal" + DimStyle.Render("  v0.3.0"))

	inputBar := m.renderInputBar()

	return topBar + "\n" + m.viewport.View() + "\n" + inputBar
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
	}
	checks := []bootCheck{
		{"Connected"},
		{"Paper trading active"},
		{"AI agent ready"},
		{"Vim mode enabled"},
	}

	for i, check := range checks {
		checkFrame := checksStart + i*2
		if m.bootFrame >= checkFrame+1 {
			// Completed — green checkmark.
			lines = append(lines, pad+"  "+checkStyle.Render("✓ "+check.label))
		} else if m.bootFrame >= checkFrame {
			// Spinning.
			spinIdx := m.bootFrame % len(spinnerFrames)
			spinner := lipgloss.NewStyle().Foreground(ColorSecondary).Render(spinnerFrames[spinIdx])
			lines = append(lines, pad+"  "+spinner+" "+DimStyle.Render(check.label+"..."))
		}
	}

	// Phase 4: Ready message.
	readyFrame := checksStart + len(checks)*2
	if m.bootFrame >= readyFrame {
		lines = append(lines, "")
		lines = append(lines, pad+DimStyle.Render("  Ready. Type /help or just ask me anything."))
	}

	content := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(content)
}

// renderInputBar renders the bottom input area based on vim mode.
func (m Model) renderInputBar() string {
	var borderColor lipgloss.Color
	var content string

	switch m.vimMode {
	case ModeInsert:
		borderColor = ColorDim
		badge := lipgloss.NewStyle().
			Background(ColorSecondary).
			Foreground(ColorWhite).
			Bold(true).
			Render(" INSERT ")
		content = badge + " " + m.textInput.View()

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
			DimStyle.Render("  j/k") + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(":scroll") +
			DimStyle.Render("  q") + lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(":quit")

		// Place badge left, hints right.
		hintsWidth := m.width - 14 // badge + padding
		if hintsWidth < 0 {
			hintsWidth = 0
		}
		rightAligned := lipgloss.NewStyle().Width(hintsWidth).Align(lipgloss.Right).Render(hints)
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
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(borderColor).
		Width(m.width).
		Padding(0, 1).
		Render(content)
}
