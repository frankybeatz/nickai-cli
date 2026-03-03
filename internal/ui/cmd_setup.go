package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/ai"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/credential"
	"github.com/nickai/cli/internal/mcp"
	"github.com/nickai/cli/internal/personality"
)

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
		if m.cfg.HasAPIKey() && !(len(args) > 1 && args[1] == "force") {
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
		m.cfg.SetSecureKey("api_key", result.User.APIKey)
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
				CommandStyle.Render("/config set <key> <value>") + "\n" +
				DimStyle.Render("  Keys: api_key, url, anthropic_key, minimax_key, openrouter_key")
		}
		key := strings.ToLower(args[1])
		value := args[2]

		switch key {
		case "api_key", "anthropic_key", "minimax_key", "openrouter_key":
			m.cfg.SetSecureKey(key, value)
		case "url":
			m.cfg.BaseURL = value
		default:
			return ErrorStyle.Render("  Unknown config key: ") + key +
				"\n" + DimStyle.Render("  Valid keys: api_key, url, anthropic_key, minimax_key, openrouter_key")
		}

		if err := m.cfg.Save(); err != nil {
			return ErrorStyle.Render("  Failed to save config: ") + err.Error()
		}
		m.client.UpdateConfig(m.cfg)

		if akKey := m.cfg.AnthropicKeyOrEnv(); akKey != "" {
			if m.agent == nil {
				m.agent = ai.NewAgent(m.client, akKey, m.toolRegistry, m.cfg.Vibe)
			}
		}
		if mmKey := m.cfg.MinimaxKeyOrEnv(); mmKey != "" {
			if m.agent == nil {
				m.agent = ai.NewAgent(m.client, "", m.toolRegistry, m.cfg.Vibe)
			}
			m.agent.SetMinimaxKey(mmKey)
		}
		if orKey := m.cfg.DataKeyOrEnv("openrouter"); orKey != "" {
			if m.agent == nil {
				m.agent = ai.NewAgent(m.client, "", m.toolRegistry, m.cfg.Vibe)
			}
			m.agent.SetOpenRouterKey(orKey)
		}
		m.updatePlaceholder()

		return RenderConfigSet(key, value)

	case "reset":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") +
				CommandStyle.Render("/config reset api_key") + "\n" +
				DimStyle.Render("  Valid keys: api_key, anthropic_key, minimax_key")
		}
		key := strings.ToLower(args[1])
		switch key {
		case "api_key", "anthropic_key", "minimax_key", "openrouter_key":
			m.cfg.DeleteSecureKey(key)
		default:
			return ErrorStyle.Render("  Unknown config key: ") + key +
				"\n" + DimStyle.Render("  Valid keys: api_key, anthropic_key, minimax_key, openrouter_key")
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
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/mcp add <name> [KEY=value ...]")
		}
		entry := mcp.GetEntry(args[1])
		if entry == nil {
			return ErrorStyle.Render("  Unknown server: ") + args[1] +
				DimStyle.Render("\n  Use ") + CommandStyle.Render("/mcp search") +
				DimStyle.Render(" to browse available servers.")
		}
		// Parse inline KEY=VALUE pairs from remaining args.
		inlineEnv := map[string]string{}
		for _, a := range args[2:] {
			if idx := strings.Index(a, "="); idx > 0 {
				inlineEnv[a[:idx]] = a[idx+1:]
			}
		}
		// Check if required env vars are provided (inline, env, or already in config).
		var missing []string
		for _, key := range entry.EnvKeys {
			if _, ok := inlineEnv[key]; ok {
				continue
			}
			if os.Getenv(key) != "" {
				continue
			}
			missing = append(missing, key)
		}
		if len(missing) > 0 {
			lines := []string{
				BotMsgStyle.Render("nick: ") + "To add " + BrandStyle.Render(entry.DisplayName) + ", provide the required keys:",
				"",
			}
			example := "/mcp add " + entry.Name
			for _, k := range missing {
				hint := "<your-value>"
				if entry.EnvHints != nil {
					if h, ok := entry.EnvHints[k]; ok {
						hint = h
					}
				}
				example += " " + k + "=" + hint
			}
			lines = append(lines, "  "+CommandStyle.Render(example))
			return strings.Join(lines, "\n")
		}
		// Write to mcp.json config.
		err := mcp.AddServerToConfig(entry, inlineEnv)
		if err != nil {
			return ErrorStyle.Render("  Failed to save MCP config: ") + err.Error()
		}
		return BotMsgStyle.Render("nick: ") + "Added " + BrandStyle.Render(entry.DisplayName) + " to " +
			DimStyle.Render("~/.nickai/mcp.json") + "." +
			DimStyle.Render("\n  Restart nickai to activate, or it will load on next launch.")

	case "remove", "rm":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/mcp remove <name>") +
				DimStyle.Render("  or  ") + CommandStyle.Render("/mcp remove all")
		}
		if strings.ToLower(args[1]) == "all" {
			mcpCfg, err := mcp.LoadMCPConfig()
			if err != nil || len(mcpCfg.MCPServers) == 0 {
				return BotMsgStyle.Render("nick: ") + "No MCP servers configured."
			}
			count := len(mcpCfg.MCPServers)
			for name := range mcpCfg.MCPServers {
				_ = mcp.RemoveServerFromConfig(name)
			}
			return BotMsgStyle.Render("nick: ") + fmt.Sprintf("Removed all %d MCP servers.", count) +
				DimStyle.Render("\n  Restart nickai to apply changes.")
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
				if err := mcp.AddServerToConfig(&e, nil); err == nil {
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
				// Truncate long descriptions to keep the list readable.
				desc := t.Description
				if idx := strings.IndexAny(desc, ".\n"); idx > 0 {
					desc = desc[:idx]
				}
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				lines = append(lines, "    "+CommandStyle.Render(t.Name)+
					DimStyle.Render("  "+desc))
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
	m.refreshInputStyles()
	m.cfg.Theme = name
	_ = m.cfg.Save()

	m.statusFlash = "Theme: " + name
	m.statusFlashExpiry = time.Now().Add(2 * time.Second)

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
		lines = append(lines, DimStyle.Render("  Custom: ")+CommandStyle.Render("/model <openrouter-slug>")+DimStyle.Render("  (e.g. openai/gpt-4o-mini)"))
		return strings.Join(lines, "\n")
	}

	modelID := strings.ToLower(args[0])

	if m.agent == nil {
		// Create agent if we have any key.
		anthKey := m.cfg.AnthropicKeyOrEnv()
		mmKey := m.cfg.MinimaxKeyOrEnv()
		orKey := m.cfg.DataKeyOrEnv("openrouter")
		if anthKey == "" && mmKey == "" && orKey == "" {
			return ErrorStyle.Render("  No API keys configured.") + "\n" +
				DimStyle.Render("  Set one with ") +
				CommandStyle.Render("/config set anthropic_key <key>") +
				DimStyle.Render(" or ") +
				CommandStyle.Render("/config set openrouter_key <key>")
		}
		if anthKey != "" {
			m.agent = ai.NewAgent(m.client, anthKey, m.toolRegistry, m.cfg.Vibe)
		} else {
			m.agent = ai.NewAgent(m.client, "", m.toolRegistry, m.cfg.Vibe)
		}
		if mmKey != "" {
			m.agent.SetMinimaxKey(mmKey)
		}
		if orKey != "" {
			m.agent.SetOpenRouterKey(orKey)
		}
		m.updatePlaceholder()
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

	m.statusFlash = "Model: " + name
	m.statusFlashExpiry = time.Now().Add(2 * time.Second)

	result := BotMsgStyle.Render("nick: ") + "Switched to " + BrandStyle.Render(name) + "."

	// Warn if non-Anthropic model (no tool use).
	if m.agent.Provider() != ai.ProviderAnthropic {
		result += "\n" + WarningStyle.Render("  ⚠ Tools are unavailable with this model.") +
			DimStyle.Render(" Trading, portfolio, and MCP tools require an Anthropic model.")
	}

	return result
}

// handleVibe processes /vibe commands (list, set).
func (m *Model) handleVibe(args []string) string {
	allVibes := personality.AllVibes()

	// Determine current vibe.
	currentID := personality.DefaultVibeID
	if m.cfg.Vibe != "" {
		currentID = m.cfg.Vibe
	}

	// No args or "list" → show all vibes.
	if len(args) == 0 || strings.ToLower(args[0]) == "list" {
		var sb strings.Builder
		sb.WriteString(BotMsgStyle.Render("nick: ") + "Pick your vibe:\n\n")
		for _, v := range allVibes {
			marker := "  "
			if v.ID == currentID {
				marker = "▸ "
			}
			line := fmt.Sprintf("%s%s %s — \"%s\"", marker, v.Emoji, v.Name, v.Tagline)
			if v.ID == currentID {
				line = lipgloss.NewStyle().Bold(true).Render(line)
			}
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n" + DimStyle.Render("Usage: ") + CommandStyle.Render("/vibe set <id>") +
			DimStyle.Render("  IDs: degen, quant, zen, hype, sensei, degen-bets"))
		return sb.String()
	}

	// "set <id>"
	if strings.ToLower(args[0]) == "set" && len(args) >= 2 {
		id := strings.ToLower(args[1])
		vibe := personality.GetVibe(id)
		if vibe.ID != id {
			return BotMsgStyle.Render("nick: ") + "Unknown vibe " + CommandStyle.Render(id) +
				". Try: degen, quant, zen, hype, sensei, degen-bets"
		}
		m.cfg.Vibe = id
		_ = m.cfg.Save()
		if m.agent != nil {
			m.agent.SetVibe(id)
		}
		m.welcomeDirty = true
		return BotMsgStyle.Render("nick: ") + vibe.Emoji + " " + lipgloss.NewStyle().Bold(true).Render(vibe.Name) +
			" activated. " + DimStyle.Render("\""+vibe.Tagline+"\"")
	}

	return BotMsgStyle.Render("nick: ") + "Usage: " + CommandStyle.Render("/vibe") +
		" or " + CommandStyle.Render("/vibe set <id>")
}
