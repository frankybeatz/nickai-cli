package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nickai/cli/internal/mcp"
)

// handlePlugin processes /plugin subcommands.
func (m *Model) handlePlugin(args []string) string {
	if len(args) == 0 {
		return renderPluginHelp()
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "list", "ls":
		return renderPluginList()

	case "install", "add":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/plugin install <name> [KEY=value ...]")
		}
		name := args[1]
		plugin := mcp.GetPlugin(name)
		if plugin == nil {
			return ErrorStyle.Render("  Unknown plugin: ") + name +
				DimStyle.Render("\n  Use ") + CommandStyle.Render("/plugin search") +
				DimStyle.Render(" to browse available plugins.")
		}

		// Parse inline KEY=VALUE pairs from remaining args.
		inlineEnv := map[string]string{}
		for _, a := range args[2:] {
			if idx := strings.Index(a, "="); idx > 0 {
				inlineEnv[a[:idx]] = a[idx+1:]
			}
		}

		// Check for missing required env vars.
		missing := mcp.MissingEnvKeys(name, inlineEnv)
		if len(missing) > 0 {
			lines := []string{
				BotMsgStyle.Render("nick: ") + "To install " +
					BrandStyle.Render(name) + ", provide the required keys:",
				"",
			}
			example := "/plugin install " + name
			for _, k := range missing {
				hint := plugin.Env[k]
				if hint == "" || hint == k {
					hint = "<your-value>"
				}
				example += " " + k + "=" + hint
			}
			lines = append(lines, "  "+CommandStyle.Render(example))
			return strings.Join(lines, "\n")
		}

		// Install the plugin.
		if err := mcp.InstallPlugin(name, inlineEnv); err != nil {
			return ErrorStyle.Render("  " + err.Error())
		}
		return BotMsgStyle.Render("nick: ") + "Installed " +
			BrandStyle.Render(name) + " plugin." +
			DimStyle.Render("\n  Restart nickai to activate, or it will load on next launch.")

	case "remove", "rm", "uninstall":
		if len(args) < 2 {
			return ErrorStyle.Render("  Usage: ") + CommandStyle.Render("/plugin remove <name>")
		}
		name := args[1]
		if err := mcp.RemovePlugin(name); err != nil {
			return ErrorStyle.Render("  " + err.Error())
		}
		return BotMsgStyle.Render("nick: ") + "Removed " +
			CommandStyle.Render(name) + " plugin." +
			DimStyle.Render("\n  Restart nickai to apply changes.")

	case "search":
		query := ""
		if len(args) > 1 {
			query = strings.Join(args[1:], " ")
		}
		results := mcp.SearchPlugins(query)
		if len(results) == 0 {
			return BotMsgStyle.Render("nick: ") + "No plugins found for " +
				CommandStyle.Render(query) + "." +
				DimStyle.Render("\n  Try: /plugin search github, /plugin search database, /plugin search crypto")
		}
		return renderPluginSearchResults(results)

	default:
		return renderPluginHelp()
	}
}

// renderPluginHelp shows available /plugin subcommands.
func renderPluginHelp() string {
	header := SectionHeader("/plugin — install MCP server plugins")
	lines := []string{
		header,
		"  " + CommandStyle.Render("/plugin list") + DimStyle.Render("                — installed & available plugins"),
		"  " + CommandStyle.Render("/plugin install <name>") + DimStyle.Render("      — install a plugin"),
		"  " + CommandStyle.Render("/plugin remove <name>") + DimStyle.Render("       — remove a plugin"),
		"  " + CommandStyle.Render("/plugin search <query>") + DimStyle.Render("      — search available plugins"),
		"",
		DimStyle.Render("  Try: ") + CommandStyle.Render("/plugin list") +
			DimStyle.Render("  or  ") + CommandStyle.Render("/plugin search github"),
	}
	return strings.Join(lines, "\n")
}

// renderPluginList shows installed and available plugins.
func renderPluginList() string {
	installed := mcp.ListInstalled()
	installedSet := map[string]bool{}
	for _, name := range installed {
		installedSet[name] = true
	}

	lines := []string{SectionHeader("Plugins")}

	// Installed section.
	if len(installed) > 0 {
		sort.Strings(installed)
		lines = append(lines, "  "+BrandStyle.Render("Installed")+":")
		for _, name := range installed {
			plugin := mcp.GetPlugin(name)
			desc := ""
			if plugin != nil {
				desc = DimStyle.Render("  " + plugin.Description)
			}
			lines = append(lines, "  "+StatusIndicator("running")+
				CommandStyle.Render(padRight(name, 18))+desc)
		}
		lines = append(lines, "")
	}

	// Available section.
	var available []mcp.PluginEntry
	for _, p := range mcp.PluginRegistry {
		if !installedSet[p.Name] {
			available = append(available, p)
		}
	}

	if len(available) > 0 {
		lines = append(lines, "  "+BrandStyle.Render("Available")+":")
		for _, p := range available {
			envInfo := ""
			if len(p.Env) > 0 {
				keys := make([]string, 0, len(p.Env))
				for k := range p.Env {
					keys = append(keys, k)
				}
				envInfo = DimStyle.Render(fmt.Sprintf("  (needs %s)", strings.Join(keys, ", ")))
			}
			lines = append(lines, "  "+StatusIndicator("stopped")+
				CommandStyle.Render(padRight(p.Name, 18))+
				DimStyle.Render(p.Description)+envInfo)
		}
	} else if len(installed) == 0 {
		lines = append(lines, DimStyle.Render("  No plugins installed."))
		lines = append(lines, "")
		lines = append(lines, DimStyle.Render("  Get started:")+
			"\n  "+CommandStyle.Render("/plugin search")+DimStyle.Render("          — browse available plugins")+
			"\n  "+CommandStyle.Render("/plugin install <name>")+DimStyle.Render("  — install a plugin"))
	}

	lines = append(lines, "")
	lines = append(lines, DimStyle.Render(fmt.Sprintf(
		"  %d installed, %d available", len(installed), len(available))))
	return strings.Join(lines, "\n")
}

// renderPluginSearchResults shows matching plugins.
func renderPluginSearchResults(results []mcp.PluginEntry) string {
	installed := mcp.ListInstalled()
	installedSet := map[string]bool{}
	for _, name := range installed {
		installedSet[name] = true
	}

	lines := []string{SectionHeader("Plugin Search Results")}
	for _, p := range results {
		status := "stopped"
		if installedSet[p.Name] {
			status = "running"
		}
		envInfo := ""
		if len(p.Env) > 0 {
			keys := make([]string, 0, len(p.Env))
			for k := range p.Env {
				keys = append(keys, k)
			}
			envInfo = DimStyle.Render(fmt.Sprintf("  (needs %s)", strings.Join(keys, ", ")))
		}
		lines = append(lines, "  "+StatusIndicator(status)+
			BrandStyle.Render(padRight(p.Name, 18))+
			DimStyle.Render(p.Description)+envInfo)
	}
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("  Use ")+
		CommandStyle.Render("/plugin install <name>")+
		DimStyle.Render(" to install a plugin."))
	return strings.Join(lines, "\n")
}
