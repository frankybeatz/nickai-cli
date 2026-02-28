# NickAI CLI — Roadmap

## v0.5 — Live Trading & Core Features

- [ ] Paper / Live mode toggle (`/mode live`, `/mode paper`)
- [ ] Unified portfolio across all connected exchanges (`/status` aggregates)
- [ ] Risk guardrails (`/risk set max-position 10%`, daily loss limits, no-leverage mode)
- [ ] Smart order routing (compare prices across exchanges, execute on best)
- [ ] Exchange credential flow (connect exchanges via `/connect binance`)

## v0.6 — AI-Powered Trading

- [ ] Natural language strategies ("scale into ETH with $2000 over 4 hours" → TWAP)
- [x] Conditional triggers ("if BTC drops below 60k, sell half")
- [ ] Portfolio rebalancing ("rebalance to 50/30/20 BTC/ETH/SOL")
- [ ] AI trade journal (win rate, pattern analysis, suggestions)

## v0.7 — Real-Time & Market Intelligence

- [ ] WebSocket live price streaming (CCXT WebSocket or exchange-native)
- [ ] Cross-exchange arbitrage detection and alerts
- [ ] Live P&L ticker (updates every second)
- [ ] Full-screen Bloomberg-style dashboard mode

## v0.8 — MCP Directory & Ecosystem

- [x] `/mcp list` — show connected servers and their tools
- [x] `/mcp add <name>` — install from curated registry
- [x] `/mcp remove <name>` — disconnect a server
- [x] `/mcp search <query>` — browse available servers
- [ ] AI-powered MCP recommendations ("I need on-chain data" → suggests servers)
- [x] Permission model (trust tiers, confirmation on trade actions)
- [ ] Remote registry fetch (pull latest from GitHub or API)

## v0.9 — Strategy & Backtesting

- [ ] Backtesting engine (`nickai backtest --strategy btc-momentum --from 2025-01-01`)
- [ ] Strategy performance metrics (Sharpe, max drawdown, win rate)
- [ ] Real workflow execution (currently mock — wire up the pipeline engine)
- [ ] Session persistence & trade replay

## Future

- [ ] Plugin system (Go plugins or WASM for custom strategies)
- [ ] Multi-agent orchestration (risk manager, signal generator, executor)
- [ ] Notifications (Telegram, Discord, Slack, email)
- [ ] Test suite (unit tests on API client, command router, agent loop)
