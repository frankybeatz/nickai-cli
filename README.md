# NickAI CLI

Conversational trading terminal for [NickAI](https://getnick.ai). Paper trade, backtest strategies, run multi-model consensus, and manage automation — all from your terminal.

![demo](demo.gif)

## Features

### AI Trading Agent
Chat naturally with **Nick**, your AI trading analyst. Ask "should I buy ETH?" and get data-driven analysis with live prices, technical indicators, and portfolio context. The AI knows your positions, recent commands, and risk limits — and can execute trades, create strategies, and set up automations with your confirmation.

### Multi-LLM Consensus (`/consensus`)
Query **10 AI models** across 3 tiers (Claude, GPT, DeepSeek, Gemini, Grok, Qwen, Llama) in parallel. Each votes **BUY/SELL/HOLD** with confidence scoring. Consensus requires 67%+ agreement — contrarian signals surface when models disagree.

### Backtesting Engine (`/backtest`)
7 preset strategies with real OHLCV data from Binance and the **Fear & Greed Index**:

| Preset | Entry | Exit |
|---|---|---|
| `rsi-reversal` | RSI < 30 | RSI > 70 |
| `macd-crossover` | MACD histogram > 0 | Histogram < 0 |
| `bollinger-bounce` | Price < lower band | +10% TP |
| `golden-cross` | SMA20 > SMA50 | SMA20 < SMA50 |
| `momentum` | RSI > 50 + MACD > 0 | RSI < 40 |
| `fear-greed` | RSI < 30 + FGI < 25 | +20% TP |
| `dip-buyer` | Below Bollinger + FGI < 30 | +12% TP |

Returns win rate, Sharpe ratio, max drawdown, profit factor, and full equity curve. Or describe any strategy in plain English and the AI builds it.

### AI Memory (`/memory`)
Nick remembers your trading style, preferences, and insights across sessions. Memories are tagged (insight, preference, context), scored by usage, and injected into every AI conversation.

### Analysis Presets (`/analyze`)
Pre-built AI analysis templates that chain MCP tools automatically:
- **sentiment** — News + social sentiment for any token
- **whale-watch** — Onchain whale movements and exchange flows
- **defi-yield** — Top DeFi yields with rug risk assessment
- **polymarket-scan** — Mispriced prediction market contracts
- **polymarket-deep** — Deep dive on a single event

### Market Analysis (`/analyze`)
Full technical analysis with **RSI**, **MACD**, **Bollinger Bands**, **SMA 20/50**, trend detection, and the **Fear & Greed Index**. The AI uses these indicators automatically when you ask for trading advice.

### Portfolio Analytics (`/analytics`)
Advanced metrics: **Sharpe ratio**, **max drawdown**, **win rate**, **profit factor**, allocation charts, and trade statistics. Ask the AI "how am I performing?" for a natural language summary.

### Automation (`/auto`)
Tell the AI "buy $100 of BTC every day" and it creates a recurring rule. 4 trigger types: schedule-based (hourly, daily, weekly), price-condition, portfolio-metric, and indicator-based (RSI, MACD, Bollinger, SMA crossovers). Every fire requires your confirmation.

### TWAP Strategies (`/strategy`)
Split large orders into time-weighted slices. `/strategy twap ETH buy $2000 4h` creates 8 slices over 4 hours — each with confirmation and risk checks.

### Risk Guardrails (`/risk`)
Set max order size, position limits, and daily loss caps. Risk checks apply to **every** trade — manual, AI, trigger, strategy, and automation. No exceptions.

### Multi-Exchange Support (`/connect`)
Connect real exchanges via API keys: **Binance**, **Coinbase**, **Hyperliquid**, **Alpaca** (stocks), **Polymarket**. Unified balance and position views across all connected accounts.

### Prediction Markets (`/polymarket`, `/bet`, `/odds`)
Scan Polymarket for mispriced contracts, place bets, check odds and line movements for sports betting — all routed through the AI for context-aware analysis.

### Stocks & Equities (`/stock`, `/screen`)
AI-powered stock analysis and screening. `/stock AAPL` for fundamentals + news. `/screen high dividend tech under $50` for natural language filtering. Powered by Alpaca MCP.

### Onchain (`/wallet`, `/swap`, `/gas`)
Check wallet balances, execute DEX swaps (Jupiter/LiFi), and monitor gas prices across chains — all via MCP servers.

### Notifications (`/notify`)
Get **desktop notifications** (macOS/Linux) and **webhook alerts** (Slack/Discord/custom) when prices hit targets, trades execute, or automations fire.

### Conditional Triggers (`/trigger`)
"If BTC drops below $60K, sell 0.5 BTC" — triggers persist across restarts and check every 30 seconds.

### Price Alerts (`/alert`)
Background alerts that fire when a symbol crosses your target price, with terminal bell and optional desktop notification.

### Trade Journal (`/history`)
Every trade is logged with source (manual, AI, trigger, strategy, automation), rationale, and timestamps.

### MCP Integrations (`/mcp`)
Connect 15+ external tools via Model Context Protocol — DeFi data, exchange APIs, on-chain analytics, news search. Install with `/mcp add <name>` or `/mcp quick` for all free servers at once.

### Streaming AI + Markdown
Responses stream token-by-token with full markdown rendering — bold, code blocks, tables, lists. Powered by [Glamour](https://github.com/charmbracelet/glamour).

### One-Command Setup
`/config init` provisions a PaperNick account instantly — $100K paper money, no signup required.

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

Requires Go 1.25+.

## Quick Start

```
# Auto-provision a PaperNick account (no signup required)
/config init

# Set your Anthropic key for AI chat
/config set anthropic_key sk-ant-...

# Install free market data servers
/mcp quick

# Start trading
what's the price of BTC?
/buy BTC 0.01
/consensus ETH
/backtest run rsi-reversal BTC
```

## Configure

```
/config init                                  # auto-provision API key
/config set api_key <your-papernick-key>      # or set manually
/config set anthropic_key <your-key>          # Claude AI (tools + streaming)
/config set openrouter_key <your-key>         # GPT-4o, Gemini, DeepSeek, Llama
/config set minimax_key <your-key>            # free model
/config test                                  # verify connection
```

Or set keys via environment:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
export OPENROUTER_API_KEY=sk-or-...
export MINIMAX_API_KEY=...
```

Config is stored at `~/.nickai/config.json`.

## AI Models

Switch between AI providers with `/model`:

| Model | Provider | Streaming | Tools | Free |
|---|---|---|---|---|
| `claude-sonnet` | Anthropic | Yes | Yes | No |
| `claude-haiku` | Anthropic | Yes | Yes | No |
| `claude-opus` | Anthropic | Yes | Yes | No |
| `gpt-4o` | OpenRouter | No | No | No |
| `gemini-flash` | OpenRouter | No | No | No |
| `deepseek-v3` | OpenRouter | No | No | No |
| `deepseek-r1` | OpenRouter | No | No | Yes |
| `llama-3.3` | OpenRouter | No | No | Yes |
| `minimax` | MiniMax | No | No | Yes |

Custom OpenRouter models: `/model openai/gpt-4o-mini` — any model slug with `/` is sent directly to OpenRouter.

Tool use (trading, portfolio, MCP) requires an Anthropic model. Non-Anthropic models are chat-only.

Multi-LLM consensus (`/consensus`) routes through OpenRouter and queries up to 10 models in parallel — no extra configuration needed.

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
| `/alert BTC > 100000` | Background price alert |
| `/trigger add BTC < 60000 sell 0.5` | Conditional trade |

### Analysis & AI
| Command | Description |
|---|---|
| `/analyze BTC` | Technical analysis (RSI, MACD, Bollinger) |
| `/analyze presets` | List analysis presets |
| `/analyze run sentiment ETH` | Run an analysis preset |
| `/consensus BTC` | Multi-LLM consensus (10 models) |
| `/consensus models` | Available consensus models |
| `/backtest presets` | Browse preset strategies |
| `/backtest run rsi-reversal BTC` | Run a backtest preset |
| `/memory` | View saved AI memories |
| `/memory clear` | Clear all memories |

### Portfolio
| Command | Description |
|---|---|
| `/status` | Positions & cash balance |
| `/orders` | Recent orders & trades |
| `/pnl` | Profit & loss summary |
| `/analytics` | Sharpe ratio, drawdown, allocation |
| `/history` | Trade journal with source attribution |
| `/snapshot` | Combined portfolio dashboard |
| `/market` | Full market overview (10 assets) |

### Automation & Strategy
| Command | Description |
|---|---|
| `/auto list` | View automation rules |
| `/auto pause <id>` | Pause a rule |
| `/auto remove <id>` | Remove a rule |
| `/strategy twap ETH buy $2000 4h` | TWAP execution strategy |
| `/strategy list` | View active strategies |
| `/risk set max-order 5000` | Set risk guardrails |
| `/risk show` | View risk limits |
| `/notify set desktop on` | Enable desktop notifications |
| `/notify set webhook <url>` | Enable webhook alerts |

### Multi-Vertical
| Command | Description |
|---|---|
| `/connect <exchange>` | Connect an exchange (Binance, Coinbase, etc.) |
| `/connect list` | Show connected exchanges |
| `/balances` | Unified balance view across exchanges |
| `/positions` | Open positions across exchanges |
| `/funding` | Perpetual funding rates |
| `/stock AAPL` | Stock analysis |
| `/screen <filters>` | AI stock screener |
| `/wallet balance <addr>` | Check wallet balances |
| `/swap SOL USDC 10` | Token swap (DEX) |
| `/gas` | Gas price estimates |

### Prediction Markets & Betting
| Command | Description |
|---|---|
| `/polymarket scan` | Scan top prediction markets |
| `/polymarket analyze <event>` | Deep dive on an event |
| `/markets` | Trending prediction markets |
| `/markets <query>` | Search markets |
| `/bet <market> <side> <amt>` | Place a prediction bet |
| `/odds Lakers vs Celtics` | Betting odds lookup |
| `/lines Super Bowl` | Line movement tracker |

### Setup & Tools
| Command | Description |
|---|---|
| `/config init` | Auto-provision API key |
| `/config` | Manage settings & keys |
| `/model` | Switch AI model |
| `/theme` | Switch color theme |
| `/mcp list` | Connected MCP servers |
| `/mcp add <name>` | Install MCP server |
| `/mcp search` | Browse server directory |
| `/mcp quick` | Install all free servers |
| `/credential list` | Exchange API keys |
| `/workflow list` | Automation workflows |
| `/man <command>` | Detailed manual page |
| `/guide` | Interactive walkthrough |
| `/help` | Show all commands |

Or just type naturally — the AI checks prices, runs analysis, executes trades, creates automations, and manages your portfolio.

## MCP Integrations

NickAI supports the [Model Context Protocol](https://modelcontextprotocol.io) — plug in external tools and the AI uses them automatically.

```
/mcp search          # browse the curated server directory
/mcp add ccxt        # install a server
/mcp list            # see connected servers & tools
/mcp quick           # install all free servers at once
```

### Available Servers

| Server | What it does | Auth |
|---|---|---|
| `ccxt` | Trade on 100+ crypto exchanges | API keys |
| `alpaca` | Stocks, ETFs, options, crypto | API keys |
| `binance` | Dedicated Binance integration | API keys |
| `polymarket` | Prediction market data & trading | API keys |
| `defillama` | DeFi TVL, yields, volumes | Free |
| `tradingview` | Technical analysis, screeners | Free |
| `brave` | News search and sentiment | Free |
| `onchain` | ERC20 tokens, transactions | Free |
| `web3` | Multi-chain (ETH, SOL, BTC) | Free |
| `evm` | 30+ EVM blockchains | Free |
| `solana` | 40+ Solana actions — tokens, DeFi, NFTs | RPC URL |
| `jupiter` | Solana DEX trades via Jupiter | Private key |
| `lifi` | Cross-chain bridge and swap | Free |

Servers run as subprocesses communicating over stdio. Config stored at `~/.nickai/mcp.json`.

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
| `Ctrl+K` | Command palette — fuzzy search all 59 commands |
| `Ctrl+T` | Theme picker with color swatches |
| `Ctrl+O` | Model selector (switch LLM provider) |
| `F1` / `?` (normal mode) | Help overlay — 4-column keybinding reference |

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
- All API communication over HTTPS (TLS 1.2+ enforced)
- Credentials stored with `0600` permissions, directories `0700`, masked in output
- Atomic file writes (write-to-temp-then-rename) prevent corruption on crash
- Per-store mutex prevents race conditions on concurrent access
- Tool results capped at 16KB, execution timeout 30s
- `crypto/rand` for all ID generation
- No telemetry, no auto-updates, no inbound network surface

See [SECURITY.md](SECURITY.md) for the full security policy and audit details.

## Stack

- [Bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — Styling
- [Glamour](https://github.com/charmbracelet/glamour) — Markdown rendering
- [PaperNick API](https://paper.getnick.ai) — Paper trading with real-time Pyth prices
- [Anthropic Claude](https://anthropic.com) — AI agent with streaming tool use
- [OpenRouter](https://openrouter.ai) — Multi-LLM chat + consensus (GPT-4o, Gemini, DeepSeek, Llama)
- [MiniMax](https://www.minimaxi.com) — Free LLM option
- [Binance](https://binance.com) — OHLCV data for backtesting
- [Alternative.me](https://alternative.me) — Fear & Greed Index
