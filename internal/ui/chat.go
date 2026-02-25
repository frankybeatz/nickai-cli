package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/ai"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/commands"
	"github.com/nickai/cli/internal/config"
)

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

	cfg    *config.Config
	client *api.PapernickClient
	agent  *ai.Agent
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

	return Model{
		textInput: ti,
		cfg:       cfg,
		client:    client,
		agent:     agent,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Discard all mouse events to prevent raw escape codes leaking into input.
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			input := strings.TrimSpace(m.textInput.Value())
			if input == "" {
				return m, nil
			}
			m.textInput.SetValue("")

			// Add user message.
			m.messages = append(m.messages, message{
				content: UserMsgStyle.Render("you: ") + input,
				isUser:  true,
			})

			// Route and render.
			result := commands.Route(input)

			switch result.Type {
			case commands.TypeQuit:
				return m, tea.Quit
			case commands.TypeClear:
				m.messages = nil
				m.viewport.SetContent(RenderWelcome(m.width))
				m.viewport.GotoBottom()
				return m, nil
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

	// Update sub-components.
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// renderResult turns a command Result into styled output.
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

	case commands.TypeUnknown:
		return ErrorStyle.Render("Unknown command: ") + r.Input + "\n" +
			DimStyle.Render("Type /help for available commands.")

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

		// Rebuild agent if anthropic key changed.
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

// handleTrade processes /buy and /sell commands.
// Formats: /buy BTC 0.1 | /sell ETH 0.5 | /buy BTC 0.1 limit 65000
func (m *Model) handleTrade(side string, args []string) string {
	if !m.client.IsConfigured() {
		return connectPrompt()
	}

	// Need at least symbol + quantity.
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

	// Parse optional "limit <price>".
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
		m.viewport.SetContent(RenderWelcome(m.width))
		return
	}

	welcome := RenderWelcome(m.width)
	var parts []string
	parts = append(parts, welcome)
	for _, msg := range m.messages {
		parts = append(parts, msg.content)
		parts = append(parts, "") // spacing between messages
	}
	m.viewport.SetContent(strings.Join(parts, "\n"))
	m.viewport.GotoBottom()
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing NickAI..."
	}

	topBar := lipgloss.NewStyle().
		Background(lipgloss.Color("#0D0D1A")).
		Foreground(ColorPrimary).
		Bold(true).
		Width(m.width).
		Padding(0, 1).
		Render("NickAI Terminal" + DimStyle.Render("  v0.1.0"))

	inputBar := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(ColorDim).
		Width(m.width).
		Padding(0, 1).
		Render(m.textInput.View())

	return topBar + "\n" + m.viewport.View() + "\n" + inputBar
}
