# NickAI CLI

Conversational trading terminal for [NickAI](https://getnick.ai). Paper trade, deploy agents, and manage workflows — all from your terminal.

![screenshot](docs/screenshot.png)
<!-- TODO: add actual screenshot -->

## Install

```bash
go run .
```

Or build the binary:

```bash
go build -o nickai .
./nickai
```

Requires Go 1.23+.

## Configure

```
/config set api_key <your-papernick-api-key>
/config set anthropic_key <your-anthropic-api-key>
/config test
```

Or set the Anthropic key via environment:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

Config is stored at `~/.nickai/config.json`.

## Commands

| Command | Description |
|---|---|
| `/help` | Show all commands |
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
| `/config` | Manage API keys & connection |
| `/clear` | Clear chat |
| `/quit` | Exit |

Or just type naturally — the AI agent can check prices, analyze markets, and place trades for you.

## Vim Mode

The terminal supports modal editing. Press `Esc` to enter NORMAL mode.

| Mode | How to enter | What it does |
|---|---|---|
| **INSERT** | `i`, `a`, `o`, `I`, `A` | Normal chat input (default) |
| **NORMAL** | `Esc` | `j`/`k` scroll, `gg`/`G` jump, `d`/`u` half-page, `q` quit |
| **COMMAND** | `:` | `:q` `:help` `:man buy` `:e file` `:set key=value` `:wf list` `:cred list` |
| **SEARCH** | `/` | `/pattern` then Enter to jump to first match |

Mode indicator badge and border color change per mode.

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
- Only 3 direct dependencies, all from [Charmbracelet](https://github.com/charmbracelet)

See [SECURITY.md](SECURITY.md) for the full security policy and audit details.

## Stack

- [Bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — Styling
- [PaperNick API](https://paper.getnick.ai) — Paper trading
- [Anthropic Claude](https://anthropic.com) — AI agent with tool use
