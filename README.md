# NickAI CLI

The universal trading terminal. Paper trade crypto, backtest strategies with walk-forward validation, run multi-model consensus, deploy persistent strategies via Nick Node, connect real exchanges — all from your terminal with an AI copilot that knows your portfolio.

![demo](demo.gif)

## Why NickAI

Most trading tools are dashboards. NickAI is a **conversation**. Ask "should I buy ETH?" and get an answer backed by live prices, 6 technical indicators, your portfolio state, and risk limits — not a chart you have to interpret yourself.

- **One prompt does the work of 5 tabs.** "Analyze BTC, backtest a momentum strategy, and show my portfolio" — the AI chains tools automatically.
- **Idea to live execution in 5 messages.** Backtest → AI critique → optimize → validate → deploy. No code, no switching tools.
- **Every trade has a safety net.** Risk guardrails, confirmation prompts, and paper money by default. You can't accidentally lose real funds.
- **Trade anything digital.** Crypto, stocks, prediction markets, DeFi, sports betting — extensible via MCP. If someone writes a server for it, Nick can trade it.
- **Runs anywhere a terminal does.** macOS, Linux, Windows, Docker. No browser, no Electron.

---

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

### Docker

```bash
docker pull ghcr.io/frankybeatz/nickai
docker run -it -v nickai-data:/home/nickai/.nickai ghcr.io/frankybeatz/nickai
```

### From Source

```bash
git clone https://github.com/frankybeatz/nickai-cli.git
cd nickai-cli
make build
./nickai
```

### Pre-Built Binaries

Download from [Releases](https://github.com/frankybeatz/nickai-cli/releases) — darwin/linux (amd64/arm64) + windows.

---

## Quick Start

```
/config init                              # auto-provision PaperNick account ($100K paper money)
/config set anthropic_key sk-ant-...      # enable AI (Claude with tool use + streaming)
/mcp quick                                # install free market data servers

what's the price of BTC?                  # AI fetches live price
/analyze ETH                              # RSI, MACD, Bollinger, SMA, Fear & Greed
/backtest run rsi-reversal BTC            # backtest with Monte Carlo + walk-forward
/backtest analyze                         # AI critiques your strategy
/consensus BTC                            # 10 AI models vote BUY/SELL/HOLD
/buy BTC 0.01                             # paper trade (requires confirmation)
```

---

## Features

### AI Trading Agent

Chat naturally with **Nick**, your AI analyst. He knows your positions, recent commands, risk limits, and trading history — and can execute trades, run backtests, create automations, and analyze markets with your confirmation. The AI chains up to 15 tool calls per response.

### Gamified Onboarding

A 7-stage journey from first launch to advanced trader. Earn **XP** for milestones, level up from **Rookie** to **Legend**, and complete stage challenges ("Boot Sequence", "First Blood", "Prove the Edge", "Boss Fight"). Nick knows your level and guides you naturally — no tutorials, no docs, just play.

### Backtesting Engine (`/backtest`)

Institutional-grade quant tools, zero code. 12 preset strategies with real OHLCV data from Binance, **walk-forward analysis** for out-of-sample validation, **grid search optimizer** for parameter tuning, **Monte Carlo simulation** for confidence intervals, and **AI strategy critique** via `/backtest analyze`.

| Preset | Entry Signal | Exit Signal |
|---|---|---|
| `rsi-reversal` | RSI < 30 | RSI > 70 |
| `macd-crossover` | MACD histogram > 0 | Histogram < 0 |
| `bollinger-bounce` | Price < lower band | +10% take-profit |
| `golden-cross` | SMA20 > SMA50 crossover | SMA20 < SMA50 |
| `momentum` | RSI > 50 + MACD > 0 | RSI < 40 |
| `fear-greed` | RSI < 30 + FGI < 25 | +20% take-profit |
| `dip-buyer` | Price < lower BB + FGI < 30 | +12% take-profit |
| `rsi-short` | RSI > 75 (short sell) | RSI < 45 |
| `macd-short` | MACD histogram < 0 (short) | Histogram > 0 |

Plus `triple-screen`, `mean-revert`, `breakout`. Returns win rate, Sharpe ratio, max drawdown, profit factor, and full equity curve.

### Nick Node (`nickai-node`)

Always-on strategy execution server. Deploy TWAP strategies, stream live prices, run background backtests, and manage alerts — all persistent across TUI restarts.

```bash
nickai-node --addr 127.0.0.1:9400        # start the node
/node connect localhost:9400              # connect from TUI
/node deploy twap BTC 1000 24h           # deploy a strategy
/node status                              # health + active jobs
```

10 gRPC RPCs with optional token authentication for non-loopback deployments.

### Multi-LLM Consensus (`/consensus`)

Query **10 AI models** across 3 tiers (Claude, GPT, DeepSeek, Gemini, Grok, Qwen, Llama) in parallel. Each votes **BUY/SELL/HOLD** with confidence scoring. Consensus requires 67%+ agreement. Contrarian signals surface when models disagree.

### AI Memory (`/memory`)

Nick remembers your trading style, preferences, and insights across sessions. Memories are tagged, scored by usage, and injected into every conversation — so the AI gets better the more you use it.

### Portfolio Analytics (`/analytics`)

**Sharpe ratio**, **max drawdown**, **win rate**, **profit factor**, allocation breakdown, and trade statistics. Ask the AI "how am I performing?" for a narrative summary grounded in your actual numbers.

### Automation (`/auto`)

Tell the AI "buy $100 of BTC every day" and it creates a recurring rule. 4 trigger types:

- **Schedule** — hourly, daily, weekly DCA
- **Price condition** — "if ETH drops below $3000"
- **Portfolio metric** — "if drawdown exceeds 5%"
- **Indicator** — RSI, MACD, Bollinger, SMA crossovers

Every fire requires your confirmation. Rules persist across restarts.

### TWAP Strategies (`/strategy`)

Split large orders into time-weighted slices. `/strategy twap ETH buy $2000 4h` creates 8 slices over 4 hours — each with confirmation and risk checks.

### Risk Guardrails (`/risk`)

Set max order size, position limits, and daily loss caps. Risk checks apply to **every trade path** — manual, AI, trigger, strategy, automation, and MCP tools. No exceptions.

### Multi-Exchange (`/connect`)

Connect real exchanges: **Binance**, **Coinbase**, **Hyperliquid**, **Alpaca** (stocks), **Polymarket**. Unified balance and position views across all connected accounts.

### Prediction Markets (`/polymarket`, `/bet`, `/odds`)

Scan Polymarket for mispriced contracts. Compare implied probability vs. AI-assessed probability. Place bets, check odds, track line movements — all with AI context.

### Stocks & Equities (`/stock`, `/screen`)

AI stock analysis and natural language screening. `/stock AAPL` for fundamentals + news. `/screen high dividend tech under $50` for filtered results. Powered by Alpaca MCP.

### On-Chain (`/wallet`, `/swap`, `/gas`)

Check wallet balances, execute DEX swaps (Jupiter), and monitor gas prices across chains via MCP.

### Notifications (`/notify`)

Desktop notifications (macOS/Linux) and webhook alerts (Slack/Discord/custom) when prices hit targets, trades execute, or automations fire.

### Data Export (`/export`)

Export trades, portfolio snapshots, and backtest results to CSV for external analysis.

---

## AI Models

Switch providers with `/model` or `Ctrl+O`:

| Model | Provider | Tools | Streaming | Free |
|---|---|---|---|---|
| `claude-opus` | Anthropic | Yes | Yes | No |
| `claude-sonnet` | Anthropic | Yes | Yes | No |
| `claude-haiku` | Anthropic | Yes | Yes | No |
| `gpt-4o` | OpenRouter | No | No | No |
| `gemini-flash` | OpenRouter | No | No | No |
| `deepseek-v3` | OpenRouter | No | No | No |
| `deepseek-r1` | OpenRouter | No | No | Yes |
| `llama-3.3` | OpenRouter | No | No | Yes |
| `minimax` | MiniMax | No | No | Yes |

**Custom models:** `/model openai/gpt-4o-mini` — any slug with `/` routes to OpenRouter.

**Tool use** (trading, portfolio, analysis, MCP) requires an Anthropic model. Non-Anthropic models are chat-only but useful for quick questions and consensus voting.

---

## Configuration

```
/config init                                  # auto-provision PaperNick API key
/config set anthropic_key <key>               # Claude AI (tools + streaming)
/config set openrouter_key <key>              # GPT-4o, Gemini, DeepSeek, Llama
/config set minimax_key <key>                 # free model
/config test                                  # verify connection
```

Or set keys via environment:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
export OPENROUTER_API_KEY=sk-or-...
export MINIMAX_API_KEY=...
```

Config stored at `~/.nickai/config.json`. API keys stored in OS keyring when available, config file fallback with `0600` permissions.

---

## Commands

### Trading

| Command | Description |
|---|---|
| `/buy BTC 0.1` | Market buy |
| `/sell ETH 1.0 limit 4200` | Limit sell |
| `/price BTC ETH SOL` | Live price quotes |
| `/watch BTC ETH SOL` | Live price dashboard |
| `/chart BTC` | ASCII sparkline chart |
| `/alert BTC > 100000` | Background price alert |
| `/trigger add BTC < 60000 sell 0.5` | Conditional trade trigger |
| `/funding BTC` | Perpetual funding rates |

### Analysis & AI

| Command | Description |
|---|---|
| `/analyze BTC` | Technical analysis (RSI, MACD, Bollinger, SMA, F&G) |
| `/analyze run sentiment ETH` | Run an analysis preset |
| `/consensus BTC` | Multi-LLM consensus (10 models) |
| `/backtest run rsi-reversal BTC` | Backtest a preset strategy |
| `/backtest analyze` | AI strategy critique |
| `/backtest presets` | Browse 12 preset strategies |
| `/memory` | View saved AI memories |
| `/analytics` | Sharpe ratio, drawdown, allocation |
| `/export trades` | Export trades to CSV |

### Portfolio

| Command | Description |
|---|---|
| `/status` | Positions & cash balance |
| `/orders` | Recent orders & trades |
| `/pnl` | Profit & loss summary |
| `/history` | Trade journal with source attribution |
| `/snapshot` | Combined portfolio dashboard |
| `/market` | Full market overview |

### Automation & Strategy

| Command | Description |
|---|---|
| `/auto list` | View automation rules |
| `/strategy twap ETH buy $2000 4h` | TWAP execution strategy |
| `/risk set max-order 5000` | Set risk guardrails |
| `/notify set desktop on` | Desktop notifications |
| `/notify set webhook <url>` | Webhook alerts (Slack/Discord) |

### Nick Node

| Command | Description |
|---|---|
| `/node connect localhost:9400` | Connect to a running node |
| `/node status` | Node health & active jobs |
| `/node deploy twap BTC 1000 24h` | Deploy strategy to node |
| `/node strategies` | List deployed strategies |
| `/node stop <id>` | Stop a running strategy |

### Multi-Vertical

| Command | Description |
|---|---|
| `/connect <exchange>` | Connect Binance, Coinbase, Alpaca, etc. |
| `/balances` | Unified balance view across exchanges |
| `/positions` | Open positions across exchanges |
| `/polymarket scan` | Scan prediction markets |
| `/bet <market> <side> <amt>` | Prediction market bet |
| `/stock AAPL` | Stock analysis |
| `/screen <filters>` | AI stock screener |
| `/wallet balance <addr>` | Wallet balances |
| `/swap SOL USDC 10` | DEX token swap |
| `/odds Lakers vs Celtics` | Betting odds |

### Setup

| Command | Description |
|---|---|
| `/config init` | Auto-provision API key |
| `/model` | Switch AI model (`Ctrl+O`) |
| `/theme` | Switch color theme (`Ctrl+T`) |
| `/vibe` | Switch AI personality |
| `/mcp add <name>` | Install MCP server |
| `/mcp quick` | Install all free servers |
| `/plugin search <query>` | Browse MCP plugins |
| `/man <command>` | Detailed manual page |
| `/guide` | Interactive walkthrough |
| `/help` | All commands (`F1`) |

Or just type naturally — the AI fetches prices, runs analysis, executes trades, and manages your portfolio.

---

## MCP Integrations

NickAI supports the [Model Context Protocol](https://modelcontextprotocol.io) — plug in external tools and the AI uses them automatically. Connections are health-checked every 5 minutes with automatic reconnection. MCP commands are sandboxed with an allowlist (`npx`, `node`, `python`, `uvx`) and sanitized environment variables.

```
/mcp search          # browse the curated server directory
/mcp add ccxt        # install a server
/mcp list            # see connected servers & tools
/mcp quick           # install all free servers at once
```

### Available Servers

| Server | What It Does | Auth | Capabilities |
|---|---|---|---|
| `ccxt` | Trade on 100+ crypto exchanges | API keys | trade |
| `alpaca` | Stocks, ETFs, options, crypto | API keys | trade |
| `binance` | Dedicated Binance integration | API keys | trade |
| `polymarket` | Prediction market data & trading | API keys | trade |
| `hyperliquid` | Perpetual futures data | Free | read-data |
| `defillama` | DeFi TVL, yields, DEX volumes | Free | read-data |
| `tradingview` | Technical analysis, screeners | Free | read-data |
| `brave-search` | News search and sentiment | Free | read-data |
| `onchain` | ERC20 tokens, transactions, contracts | Free | read-data |
| `evm` | 30+ EVM blockchains | Free | read-data |
| `solana` | 40+ Solana actions — tokens, DeFi, NFTs | RPC URL | on-chain |
| `jupiter` | Solana DEX trades via Jupiter | Private key | on-chain |
| `coinmarketcap` | 50+ crypto data tools | API key | read-data |

Servers with `trade` or `on-chain` capability require **user confirmation** on every call and run through the same risk limit checks as built-in trades.

---

## Vim Mode

Press `Esc` to enter NORMAL mode. Full modal editing with search navigation.

| Mode | Enter With | What It Does |
|---|---|---|
| **INSERT** | `i`, `a`, `o`, `I`, `A` | Normal chat input (default) |
| **NORMAL** | `Esc` | `j`/`k` scroll, `gg`/`G` jump, `d`/`u` half-page, `n`/`N` search next/prev |
| **COMMAND** | `:` | `:q` `:help` `:man buy` `:set key=value` |
| **SEARCH** | `/` | `/pattern` then `n`/`N` to cycle matches `[3/17]` |
| **CONFIRM** | automatic | `y` confirm, `n` cancel (trade execution) |

### Shortcuts

| Key | Action |
|---|---|
| `Ctrl+K` | Command palette — fuzzy search all commands |
| `Ctrl+T` | Theme picker with color swatches |
| `Ctrl+O` | Model selector |
| `F1` | Help overlay |
| `Tab` | Autocomplete commands and symbols |
| `Up/Down` | Command history |

---

## Themes

10 built-in themes — switch with `/theme` or `Ctrl+T`:

`default` `cyberpunk` `bloomberg` `minimal` `matrix` `tokyonight` `dracula` `catppuccin` `nord` `gruvbox`

Theme changes apply instantly to all UI elements including the input bar.

---

## Nick Node

Nick Node is a standalone gRPC server for persistent, always-on operations:

```bash
# Start the node
nickai-node --addr 127.0.0.1:9400

# With auth token (required for non-loopback)
NICKAI_NODE_TOKEN=your-secret nickai-node --addr 0.0.0.0:9400

# Docker
docker compose up -d
```

See [docs/deployment.md](docs/deployment.md) for Docker, binary install, and CI/CD details.

---

## Debugging

```bash
nickai --debug              # logs to ~/.nickai/debug.log
NICKAI_DEBUG=1 nickai       # same via environment variable
tail -f ~/.nickai/debug.log # watch in another terminal
```

Structured logging with `log/slog` — covers MCP connections, API retries, model switches, and tool execution. Debug logs never contain API keys.

---

## Telemetry

Opt-in local error journal. Disabled by default — no external services, no Sentry, no data leaves your machine.

```
/config set telemetry on     # enable
/config set telemetry off    # disable
```

Events stored at `~/.nickai/telemetry_events.json` with `0600` permissions. Auto-pruned after 30 days.

---

## Security

- **MCP sandbox** — command allowlist, sanitized subprocess environment
- **Prompt sanitization** — strips LLM role markers and tool injection from user content
- **Input validation** — symbol validation, MCP result truncation (50KB)
- **Atomic file writes** — write-to-temp-then-rename prevents corruption on crash
- **File permissions** — all data `0600`, directories `0700` (owner-only)
- **OS keyring** — API keys stored in system keychain when available
- **`crypto/rand`** — all ID generation uses cryptographic randomness
- **TLS 1.2+ enforced** — all API calls over HTTPS
- **Node auth** — token-based gRPC authentication for non-loopback deployments
- **Trade confirmation** — required for all trade paths (manual, AI, MCP, automation)
- **MCP health checks** — dead connections detected every 5 minutes, auto-reconnected
- **CI** — `go vet`, `go test`, `go build` on every push

See [SECURITY.md](SECURITY.md) and [docs/THREAT_MODEL.md](docs/THREAT_MODEL.md) for the full security policy and threat model.

---

## Architecture

27 internal packages, ~28K lines of Go. Key design decisions:

- **Bubbletea** — Elm-architecture TUI with message-driven async (no goroutine soup)
- **Nick Node** — gRPC server for persistent strategy execution, live streaming, background backtests
- **Backtest engine** — streaming indicators (O(1) memory), walk-forward, Monte Carlo, optimizer
- **Tool registry** — unified registry for built-in tools + MCP tools, with collision-safe naming
- **Command registry** — single `CommandDef` source generates routing, palette, and help
- **Provider abstraction** — Anthropic, OpenRouter, MiniMax behind a common `Chat`/`ChatStream` interface
- **MCP client manager** — background connection, health monitoring, sandbox, auto-reconnection
- **Telemetry** — opt-in structured event journal with prune and summary
- **Sanitize** — prompt injection prevention for memory, automations, and MCP results
- **Safefile** — atomic writes + per-path mutex for all persistent state

## Stack

- [Bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — Styling
- [Glamour](https://github.com/charmbracelet/glamour) — Markdown rendering
- [gRPC](https://grpc.io) — Nick Node RPC framework
- [PaperNick API](https://paper.getnick.ai) — Paper trading with real-time Pyth prices
- [Anthropic Claude](https://anthropic.com) — AI agent with streaming tool use
- [OpenRouter](https://openrouter.ai) — Multi-LLM chat + consensus
- [Binance](https://binance.com) — OHLCV data + live WebSocket prices
- [Alternative.me](https://alternative.me) — Fear & Greed Index

## License

MIT
