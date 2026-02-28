# NickAI CLI

Conversational trading terminal for [NickAI](https://getnick.ai). Paper trade, deploy agents, and manage workflows — all from your terminal.

![demo](demo.gif)

## Features

### AI Trading Agent
Chat naturally with **Nick**, your AI trading analyst. Ask "should I buy ETH?" and get data-driven analysis with live prices, technical indicators, and portfolio context. The AI can execute trades, create strategies, and set up automations — all with your confirmation.

### Market Analysis (`/analyze`)
Full technical analysis with **RSI**, **MACD**, **Bollinger Bands**, **SMA 20/50**, trend detection, and the **Fear & Greed Index**. The AI uses these indicators automatically when you ask for trading advice.

### Portfolio Analytics (`/analytics`)
Advanced metrics: **Sharpe ratio**, **max drawdown**, **win rate**, **profit factor**, allocation charts, and trade statistics. Ask the AI "how am I performing?" for a natural language summary.

### Automation (`/auto`)
Tell the AI "buy $100 of BTC every day" and it creates a recurring rule. Supports schedule-based (hourly, daily, weekly), price-condition, and portfolio-metric triggers. Every fire requires your confirmation.

### TWAP Strategies (`/strategy`)
Split large orders into time-weighted slices. `/strategy twap ETH buy $2000 4h` creates 8 slices over 4 hours — each with confirmation and risk checks.

### Risk Guardrails (`/risk`)
Set max order size, position limits, and daily loss caps. Risk checks apply to **every** trade — manual, AI, trigger, strategy, and automation.

### Notifications (`/notify`)
Get **desktop notifications** (macOS/Linux) and **webhook alerts** when prices hit targets, trades execute, or automations fire. Never miss an event.

### Conditional Triggers (`/trigger`)
"If BTC drops below $60K, sell 0.5 BTC" — triggers persist across restarts and check every 30 seconds.

### Price Alerts (`/alert`)
Background alerts that fire when a symbol crosses your target price, with terminal bell and optional desktop notification.

### Trade Journal (`/history`)
Every trade is logged with source (manual, AI, trigger, strategy, automation), rationale, and timestamps.

### MCP Integrations (`/mcp`)
Connect external tools via Model Context Protocol — DeFi data, exchange APIs, on-chain analytics. Install with `/mcp add <name>`.

### Streaming AI + Markdown
Responses stream token-by-token with full markdown rendering — bold, code blocks, tables, lists. Powered by [Glamour](https://github.com/charmbracelet/glamour).

### One-Command Setup
`/config init` provisions a PaperNick account instantly — no signup required.

## Install

### Homebrew (macOS & Linux)

```bash
brew tap frankybeatz/nickai
brew install nickai
```

### Go

```bash
go install github.com/nickai/cli@latest
```

### From source

```bash
git clone https://github.com/frankybeatz/nickai-cli.git
cd nickai-cli
make build
./nickai
```

### Pre-built binaries

Download from [Releases](https://github.com/frankybeatz/nickai-cli/releases), or build all platforms locally:

```bash
make release    # builds darwin/linux/windows × arm64/amd64 to build/
```

Requires Go 1.23+.

## Quick Start

```
# Auto-provision a PaperNick account (no signup required)
/config init

# Set your Anthropic key for AI chat
/config set anthropic_key sk-ant-...

# Start trading
what's the price of BTC?
/buy BTC 0.01
/status
```

## Configure

```
/config init                               # auto-provision API key (new)
/config set api_key <your-papernick-key>   # or set manually
/config set anthropic_key <your-key>       # Claude AI
/config set minimax_key <your-key>         # free model
/config test                               # verify connection
```

Or set keys via environment:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
export MINIMAX_API_KEY=...
```

Config is stored at `~/.nickai/config.json`.

## AI Models

Switch between AI providers with `/model`:

| Model | Provider | Streaming | Free |
|---|---|---|---|
| `claude-sonnet` | Anthropic | Yes | No |
| `claude-haiku` | Anthropic | Yes | No |
| `minimax` | MiniMax | No | Yes |

Anthropic models stream responses token-by-token with tool use. MiniMax returns the full response at once.

## Themes

10 built-in themes — switch with `/theme <name>`:

`default` `cyberpunk` `bloomberg` `minimal` `matrix` `tokyonight` `dracula` `catppuccin` `nord` `gruvbox`

## Commands

### Trading
| Command | Description |
|---|---|
| `/buy BTC 0.1` | Market buy |
| `/sell ETH 1.0 limit 4200` | Limit sell |
| `/price BTC ETH` | Live price quotes |
| `/watch BTC ETH SOL` | Live price dashboard |
| `/chart BTC` | ASCII sparkline chart |
| `/analyze BTC` | Technical analysis (RSI, MACD, Bollinger) |
| `/alert BTC > 100000` | Background price alert |
| `/trigger add BTC < 60000 sell 0.5` | Conditional trade |

### Portfolio
| Command | Description |
|---|---|
| `/status` | Positions & cash balance |
| `/orders` | Recent orders & trades |
| `/pnl` | Profit & loss summary |
| `/analytics` | Sharpe ratio, drawdown, allocation |
| `/history` | Trade journal |
| `/snapshot` | Combined dashboard |
| `/market` | Full market overview (10 assets) |

### Automation & Strategy
| Command | Description |
|---|---|
| `/auto list` | View automation rules |
| `/auto pause <id>` | Pause a rule |
| `/strategy twap ETH buy $2000 4h` | TWAP execution strategy |
| `/strategy list` | View active strategies |
| `/risk set max-order 5000` | Set risk guardrails |
| `/risk show` | View risk limits |
| `/notify set desktop on` | Enable desktop notifications |
| `/notify set webhook <url>` | Enable webhook alerts |

### Setup & Tools
| Command | Description |
|---|---|
| `/config init` | Auto-provision API key |
| `/config` | Manage settings & keys |
| `/model` | Switch AI model |
| `/theme` | Switch color theme |
| `/mcp list` | Connected MCP servers |
| `/mcp add <name>` | Install MCP server |
| `/credential list` | Exchange API keys |
| `/workflow list` | Automation workflows |
| `/man <command>` | Detailed manual page |

Or just type naturally — the AI can check prices, run technical analysis, execute trades, create automations, and manage your portfolio.

## Vim Mode

The terminal supports modal editing. Press `Esc` to enter NORMAL mode.

| Mode | How to enter | What it does |
|---|---|---|
| **INSERT** | `i`, `a`, `o`, `I`, `A` | Normal chat input (default) |
| **NORMAL** | `Esc` | `j`/`k` scroll, `gg`/`G` jump, `d`/`u` half-page, `q` quit |
| **COMMAND** | `:` | `:q` `:help` `:man buy` `:e file` `:set key=value` `:wf list` `:cred list` |
| **SEARCH** | `/` | `/pattern` then Enter to jump to first match |
| **CONFIRM** | automatic | `y` confirm, `n` cancel (trade execution) |

Mode indicator badge and border color change per mode. Tab completes commands and symbols. Up/Down arrow cycles command history.

### Overlay Dialogs

| Shortcut | Dialog |
|---|---|
| `Ctrl+K` | Command palette — fuzzy search all commands |
| `Ctrl+T` | Theme picker with color swatches |
| `Ctrl+O` | Model selector (switch LLM provider) |
| `?` (normal mode) | Help overlay — 3-column keybinding reference |

## Workflows

Create automation pipelines from JSON definitions:

```bash
/workflow create examples/btc-momentum.json
/workflow run btc-momentum
/logs btc-momentum
```

See `examples/` for sample workflows.

## Security

- Zero known vulnerabilities (`govulncheck` clean)
- All API communication over HTTPS
- Credentials stored with `0600` permissions, masked in output
- No telemetry, no auto-updates, no inbound network surface

See [SECURITY.md](SECURITY.md) for the full security policy and audit details.

## Stack

- [Bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — Styling
- [Glamour](https://github.com/charmbracelet/glamour) — Markdown rendering
- [PaperNick API](https://paper.getnick.ai) — Paper trading
- [Anthropic Claude](https://anthropic.com) — AI agent with streaming tool use
- [MiniMax](https://www.minimaxi.com) — Free LLM option
