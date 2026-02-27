# NickAI CLI

Conversational trading terminal for [NickAI](https://getnick.ai). Paper trade, deploy agents, and manage workflows — all from your terminal.

![demo](demo.gif)

## What's New

### Streaming AI Responses

AI responses now appear **token-by-token** as they're generated instead of blocking behind a spinner. The terminal feels alive — you see the AI thinking in real time. Tool-use rounds (price lookups, portfolio checks, trade execution) still work seamlessly: the stream pauses during tool execution and resumes when results are ready.

### Markdown Rendering

AI responses are rendered with full markdown formatting — **bold**, `inline code`, code blocks, lists, tables, and headings all display correctly in the terminal. Powered by [Glamour](https://github.com/charmbracelet/glamour).

### One-Command Setup (`/config init`)

New users no longer need to visit a website, create an account, and copy-paste an API key. Just run `/config init` and NickAI provisions an anonymous PaperNick account automatically, stores the key, and you're trading in seconds.

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

| Command | Description |
|---|---|
| `/help` | Show all commands |
| `/config init` | Auto-provision API key |
| `/status` | Portfolio, positions & cash |
| `/orders` | Recent orders & trades |
| `/price BTC ETH` | Live price quotes |
| `/watch BTC ETH SOL` | Live price dashboard |
| `/snapshot` | Combined portfolio dashboard |
| `/market` | Full market overview (10 assets) |
| `/pnl` | Profit & loss summary |
| `/history` | Trade journal with all orders |
| `/buy BTC 0.1` | Market buy |
| `/sell ETH 1.0 limit 4200` | Limit sell |
| `/agents` | List trading agents |
| `/templates` | Browse marketplace |
| `/workflow list` | Manage automation workflows |
| `/credential list` | Manage exchange API keys |
| `/logs <workflow>` | Workflow execution logs |
| `/man <command>` | Detailed manual page |
| `/chart BTC` | ASCII sparkline chart |
| `/alert BTC > 100000` | Set a price alert |
| `/model` | Switch AI model |
| `/theme` | Switch color theme |
| `/config` | Manage API keys & connection |
| `/clear` | Clear chat |
| `/quit` | Exit |

Or just type naturally — the AI agent can check prices, analyze markets, and place trades for you. Responses stream in real time with full markdown formatting.

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
