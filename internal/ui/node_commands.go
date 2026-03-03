package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nickai/cli/internal/node"
	"github.com/nickai/cli/internal/node/pb"
)

// handleNode dispatches /node subcommands.
func (m *Model) handleNode(sub string, args []string) (string, tea.Cmd) {
	prefix := BotMsgStyle.Render("nick: ")

	switch sub {
	case "connect":
		return m.handleNodeConnect(args)
	case "status":
		return m.handleNodeStatus()
	case "deploy":
		return m.handleNodeDeploy(args)
	case "strategies":
		return m.handleNodeStrategies()
	case "stop":
		return m.handleNodeStop(args)
	case "disconnect":
		return m.handleNodeDisconnect()
	case "":
		return m.handleNodeHelp()
	default:
		return prefix + fmt.Sprintf("Unknown node subcommand: %s\n\nUse /node for available commands.", sub), nil
	}
}

// handleNodeHelp shows the node command help.
func (m *Model) handleNodeHelp() (string, tea.Cmd) {
	prefix := BotMsgStyle.Render("nick: ")
	var sb strings.Builder
	sb.WriteString("Node Commands\n\n")
	sb.WriteString(CommandStyle.Render("/node connect [addr]") + "  Connect to a Nick Node (default: " + node.DefaultAddr + ")\n")
	sb.WriteString(CommandStyle.Render("/node status") + "         Show node status\n")
	sb.WriteString(CommandStyle.Render("/node deploy <sym>") + "   Deploy a monitoring strategy\n")
	sb.WriteString(CommandStyle.Render("/node strategies") + "     List running strategies\n")
	sb.WriteString(CommandStyle.Render("/node stop <id>") + "      Stop a running strategy\n")
	sb.WriteString(CommandStyle.Render("/node disconnect") + "     Disconnect from node\n")

	if m.nodeClient != nil {
		sb.WriteString("\nConnected to " + BrandStyle.Render(m.nodeClient.Addr()))
	} else {
		sb.WriteString("\nNot connected. Start a node with " + CommandStyle.Render("nickai-node") + " then /node connect")
	}

	return prefix + sb.String(), nil
}

// handleNodeConnect connects to a Nick Node server.
func (m *Model) handleNodeConnect(args []string) (string, tea.Cmd) {
	prefix := BotMsgStyle.Render("nick: ")

	if m.nodeClient != nil {
		return prefix + "Already connected to " + BrandStyle.Render(m.nodeClient.Addr()) + ". Use /node disconnect first.", nil
	}

	addr := node.DefaultAddr
	if len(args) > 0 {
		addr = args[0]
	}

	// Connect asynchronously to avoid blocking the TUI.
	return prefix + "Connecting to " + CommandStyle.Render(addr) + "...",
		func() tea.Msg {
			client, err := node.NewClient(addr)
			if err != nil {
				return nodeConnectResultMsg{err: err}
			}
			// Ping to verify the connection is live.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			resp, err := client.Ping(ctx)
			if err != nil {
				client.Close()
				return nodeConnectResultMsg{err: fmt.Errorf("ping failed: %w", err)}
			}
			return nodeConnectResultMsg{client: client, version: resp.Version, uptime: resp.UptimeSeconds}
		}
}

// nodeConnectResultMsg is sent when a node connection attempt completes.
type nodeConnectResultMsg struct {
	client  *node.Client
	version string
	uptime  int64
	err     error
}

// handleNodeStatus shows the connected node's status.
func (m *Model) handleNodeStatus() (string, tea.Cmd) {
	prefix := BotMsgStyle.Render("nick: ")

	if m.nodeClient == nil {
		return prefix + "Not connected to a node. Use " + CommandStyle.Render("/node connect") + " first.", nil
	}

	return prefix + "Fetching node status...",
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			resp, err := m.nodeClient.GetStatus(ctx)
			if err != nil {
				return apiResponseMsg{content: BotMsgStyle.Render("nick: ") + ErrorStyle.Render("Node status error: "+err.Error())}
			}

			var sb strings.Builder
			sb.WriteString("Node Status\n\n")
			sb.WriteString(fmt.Sprintf("  Version:     %s\n", resp.Version))
			sb.WriteString(fmt.Sprintf("  Uptime:      %s\n", formatDuration(time.Duration(resp.UptimeSeconds)*time.Second)))
			sb.WriteString(fmt.Sprintf("  Strategies:  %d running\n", resp.RunningStrategies))
			sb.WriteString(fmt.Sprintf("  Alerts:      %d active\n", resp.ActiveAlerts))
			sb.WriteString(fmt.Sprintf("  Goroutines:  %d\n", resp.Goroutines))
			sb.WriteString(fmt.Sprintf("  Memory:      %.1f MB\n", float64(resp.MemoryBytes)/1024/1024))
			if len(resp.ConnectedSymbols) > 0 {
				sb.WriteString(fmt.Sprintf("  Symbols:     %s\n", strings.Join(resp.ConnectedSymbols, ", ")))
			}

			return apiResponseMsg{content: BotMsgStyle.Render("nick: ") + sb.String()}
		}
}

// handleNodeDeploy deploys a simple monitoring strategy to the node.
func (m *Model) handleNodeDeploy(args []string) (string, tea.Cmd) {
	prefix := BotMsgStyle.Render("nick: ")

	if m.nodeClient == nil {
		return prefix + "Not connected to a node. Use " + CommandStyle.Render("/node connect") + " first.", nil
	}

	if len(args) == 0 {
		return prefix + "Usage: " + CommandStyle.Render("/node deploy <symbol> [name]") + "\n  Example: /node deploy BTC \"RSI Monitor\"", nil
	}

	symbol := strings.ToUpper(args[0])
	name := symbol + " Monitor"
	if len(args) > 1 {
		name = strings.Join(args[1:], " ")
	}

	spec := &pb.StrategySpec{
		Name:     name,
		Symbol:   symbol,
		Interval: "5m",
		EntryRules: []*pb.StrategyCondition{
			{Indicator: "rsi", Operator: "<", Value: 30},
		},
		ExitRules: []*pb.StrategyCondition{
			{Indicator: "rsi", Operator: ">", Value: 70},
		},
	}

	client := m.nodeClient
	return prefix + "Deploying " + CommandStyle.Render(name) + " to node...",
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			id, err := client.DeployStrategy(ctx, spec)
			if err != nil {
				return apiResponseMsg{content: BotMsgStyle.Render("nick: ") + ErrorStyle.Render("Deploy failed: "+err.Error())}
			}
			return apiResponseMsg{content: BotMsgStyle.Render("nick: ") + BrandStyle.Render("Deployed!") + fmt.Sprintf(" Strategy %s is now running on node.\n  ID: %s", name, id)}
		}
}

// handleNodeStrategies lists running strategies on the node.
func (m *Model) handleNodeStrategies() (string, tea.Cmd) {
	prefix := BotMsgStyle.Render("nick: ")

	if m.nodeClient == nil {
		return prefix + "Not connected to a node. Use " + CommandStyle.Render("/node connect") + " first.", nil
	}

	client := m.nodeClient
	return prefix + "Fetching strategies...",
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			strategies, err := client.ListStrategies(ctx)
			if err != nil {
				return apiResponseMsg{content: BotMsgStyle.Render("nick: ") + ErrorStyle.Render("List strategies error: "+err.Error())}
			}
			if len(strategies) == 0 {
				return apiResponseMsg{content: BotMsgStyle.Render("nick: ") + "No strategies deployed. Use " + CommandStyle.Render("/node deploy <symbol>")}
			}

			var sb strings.Builder
			sb.WriteString("Node Strategies\n\n")
			for _, s := range strategies {
				statusBadge := s.Status.String()
				if s.Status == pb.StrategyStatusRunning {
					statusBadge = BrandStyle.Render(statusBadge)
				} else if s.Status == pb.StrategyStatusErrored {
					statusBadge = ErrorStyle.Render(statusBadge)
				}
				sb.WriteString(fmt.Sprintf("  %s  %s  %s  [%s]\n",
					CommandStyle.Render(s.ID),
					s.Spec.Symbol,
					s.Spec.Name,
					statusBadge,
				))
			}
			return apiResponseMsg{content: BotMsgStyle.Render("nick: ") + sb.String()}
		}
}

// handleNodeStop stops a running strategy on the node.
func (m *Model) handleNodeStop(args []string) (string, tea.Cmd) {
	prefix := BotMsgStyle.Render("nick: ")

	if m.nodeClient == nil {
		return prefix + "Not connected to a node. Use " + CommandStyle.Render("/node connect") + " first.", nil
	}

	if len(args) == 0 {
		return prefix + "Usage: " + CommandStyle.Render("/node stop <id>"), nil
	}

	id := args[0]
	client := m.nodeClient
	return prefix + "Stopping strategy " + CommandStyle.Render(id) + "...",
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			stopped, err := client.StopStrategy(ctx, id)
			if err != nil {
				return apiResponseMsg{content: BotMsgStyle.Render("nick: ") + ErrorStyle.Render("Stop failed: "+err.Error())}
			}
			if stopped {
				return apiResponseMsg{content: BotMsgStyle.Render("nick: ") + BrandStyle.Render("Strategy stopped.")}
			}
			return apiResponseMsg{content: BotMsgStyle.Render("nick: ") + "Strategy was already stopped."}
		}
}

// handleNodeDisconnect disconnects from the node.
func (m *Model) handleNodeDisconnect() (string, tea.Cmd) {
	prefix := BotMsgStyle.Render("nick: ")

	if m.nodeClient == nil {
		return prefix + "Not connected to any node.", nil
	}

	addr := m.nodeClient.Addr()
	m.nodeClient.Close()
	m.nodeClient = nil

	return prefix + "Disconnected from " + CommandStyle.Render(addr), nil
}

// formatDuration formats a time.Duration into a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if hours < 24 {
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
	days := hours / 24
	hours = hours % 24
	return fmt.Sprintf("%dd%dh", days, hours)
}
