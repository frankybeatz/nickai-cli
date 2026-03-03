# NickAI CLI Data Schema Reference

> Auto-generated documentation of all structs, persistence locations, JSON tags, and
> relationships across the nickai-cli codebase.

---

## Table of Contents

1. [Persistence Map](#persistence-map)
2. [Package: config](#package-config)
3. [Package: credential](#package-credential)
4. [Package: mcp](#package-mcp)
5. [Package: risk](#package-risk)
6. [Package: memory](#package-memory)
7. [Package: journal](#package-journal)
8. [Package: automation](#package-automation)
9. [Package: strategy](#package-strategy)
10. [Package: alert](#package-alert)
11. [Package: trigger](#package-trigger)
12. [Package: workflow](#package-workflow)
13. [Package: notify](#package-notify)
14. [Package: market](#package-market)
15. [Package: api](#package-api)
16. [Package: ai](#package-ai)
17. [Package: backtest](#package-backtest)
18. [Package: indicators](#package-indicators)
19. [Package: tools](#package-tools)
20. [Package: analytics](#package-analytics)
21. [Package: guidance](#package-guidance)
22. [Package: personality](#package-personality)
23. [Package: node/pb](#package-nodepb)
24. [Package: node](#package-node)
25. [Package: commands](#package-commands)
26. [Package: mock](#package-mock)
27. [Example JSON Snippets](#example-json-snippets)
28. [Relationship Diagram](#relationship-diagram)

---

## Persistence Map

All files are stored under `~/.nickai/` with 0600 permissions. Writes use atomic
temp-file-then-rename via `safefile.AtomicWrite` to prevent corruption. File-level
mutex locks are acquired for concurrent-safe read-modify-write operations.

| File | Format | Root Type | Package | Description |
|------|--------|-----------|---------|-------------|
| `config.json` | JSON object | `config.Config` | `config` | App settings, API keys, theme, model, vibe |
| `credentials.json` | JSON object | `credential.Store` | `credential` | Exchange API credentials (API key + secret) |
| `mcp.json` | JSON object | `mcp.MCPConfig` | `mcp` | MCP server launch configs (command, args, env) |
| `risk.json` | JSON object | `risk.RiskLimits` | `risk` | Portfolio risk guardrails |
| `memory.json` | JSON object | `memory.Store` | `memory` | AI memory entries (insights, preferences, context) |
| `journal.json` | JSON array | `[]journal.JournalEntry` | `journal` | Trade journal with AI rationale |
| `automations.json` | JSON array | `[]automation.AutoRule` | `automation` | Automation rules (schedule, condition, portfolio, indicator) |
| `strategies.json` | JSON array | `[]strategy.TWAPStrategy` | `strategy` | TWAP execution strategies |
| `alerts.json` | JSON array | `[]alert.Alert` | `alert` | Price alerts |
| `triggers.json` | JSON array | `[]trigger.Trigger` | `trigger` | Conditional trade triggers |
| `workflows.json` | JSON object | `workflow.Store` | `workflow` | Multi-node automation pipelines |
| `notify.json` | JSON object | `notify.Config` | `notify` | Notification preferences (desktop, webhook, sound) |
| `cache/candles/<SYM>_<INT>_<N>.json` | JSON object | `market.cacheEntry` | `market` | Cached OHLCV candle data from Binance |

**Key storage note:** Sensitive keys (api_key, anthropic_key, minimax_key, openrouter_key)
are stored in the OS keyring when available. The plaintext field in `config.json` is cleared
when the keyring is used. The `credential.KeyringGet/Set/Delete` helpers manage this.

---

## Package: config

**Source:** `internal/config/config.go`
**Persisted to:** `~/.nickai/config.json`

### Config

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| APIKey | `string` | `api_key,omitempty` | PaperNick API key (cleared when keyring active) |
| BaseURL | `string` | `base_url,omitempty` | API base URL (default: `https://paper.getnick.ai/api/v1`) |
| AnthropicKey | `string` | `anthropic_key,omitempty` | Anthropic API key (cleared when keyring active) |
| MinimaxKey | `string` | `minimax_key,omitempty` | MiniMax API key (cleared when keyring active) |
| Theme | `string` | `theme,omitempty` | Color theme name |
| Model | `string` | `model,omitempty` | Active AI model ID (e.g. `claude-sonnet`) |
| DataKeys | `map[string]string` | `data_keys,omitempty` | Premium data source API keys (e.g. `{"openrouter": "sk-..."}`) |
| Vibe | `string` | `vibe,omitempty` | Active personality vibe ID |

**Relationships:** Read by `api.NewClient`, `ai.NewAgent`. Keys resolved via
`credential.KeyringGet` -> config file -> env var fallback chain.

---

## Package: credential

**Source:** `internal/credential/store.go`
**Persisted to:** `~/.nickai/credentials.json`

### Credential

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Name | `string` | `name` | User-assigned credential name |
| Exchange | `string` | `exchange` | Exchange name (binance, coinbase, hyperliquid, alpaca, polymarket) |
| APIKey | `string` | `api_key` | Exchange API key |
| APISecret | `string` | `api_secret` | Exchange API secret |

### Store

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Credentials | `[]Credential` | `credentials` | List of exchange credentials |

**Relationships:** Used by `/connect`, `/credential` commands. Supported exchanges:
binance, coinbase, hyperliquid, alpaca, polymarket.

---

## Package: mcp

**Source:** `internal/mcp/config.go`, `client.go`, `registry.go`, `plugin.go`
**Persisted to:** `~/.nickai/mcp.json`

### MCPConfig

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| MCPServers | `map[string]MCPServerConfig` | `mcpServers` | Map of server name to launch config |

### MCPServerConfig

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Command | `string` | `command` | Executable to run (e.g. `npx`) |
| Args | `[]string` | `args,omitempty` | Command arguments |
| Env | `map[string]string` | `env,omitempty` | Environment variables |

### RegistryEntry (in-memory, not persisted)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Name | `string` | - | Short name for `/mcp add` |
| DisplayName | `string` | - | Human-readable name |
| Description | `string` | - | One-liner description |
| Repo | `string` | - | GitHub URL |
| Command | `string` | - | Launch command |
| Args | `[]string` | - | Default args |
| EnvKeys | `[]string` | - | Required environment variable names |
| EnvHints | `map[string]string` | - | Human-readable hint per env var |
| Tier | `TrustTier` | - | `"verified"` or `"community"` |
| Capabilities | `[]Capability` | - | `"read-data"`, `"trade"`, `"on-chain"`, `"analytics"` |
| Tags | `[]string` | - | Searchable tags |

### PluginEntry (in-memory, not persisted)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Name | `string` | - | Short name for `/plugin install` |
| Description | `string` | - | One-liner |
| Command | `string` | - | Launch command |
| Args | `[]string` | - | Default args |
| Env | `map[string]string` | - | Required env vars with hints |
| Tags | `[]string` | - | Searchable tags |
| RequiresNpx | `bool` | - | Whether plugin needs npx |

### MCPConnection (in-memory, runtime)

Managed by `ClientManager`. Each connection holds an `mcp.Client` handle, the server
name, and the list of discovered tools. Not persisted.

### FailedConnection (in-memory, runtime)

Holds server name and error message for servers that failed to connect at startup.

**Relationships:** `MCPConfig` is loaded on startup by `ClientManager`. `RegistryEntry`
and `PluginEntry` are used by `/mcp add`, `/mcp search`, `/plugin install`. Tools
discovered from MCP servers are registered into `tools.Registry`.

---

## Package: risk

**Source:** `internal/risk/store.go`
**Persisted to:** `~/.nickai/risk.json`

### RiskLimits

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| MaxPositionPct | `float64` | `max_position_pct,omitempty` | Max single-asset concentration (e.g. 10 = 10%) |
| DailyLossPct | `float64` | `daily_loss_pct,omitempty` | Max daily loss before blocking trades (e.g. 5 = 5%) |
| MaxOrderValue | `float64` | `max_order_value,omitempty` | Max single order value in USD (e.g. 5000) |

### CheckResult (in-memory)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Allowed | `bool` | - | Whether the order passes risk checks |
| Reason | `string` | - | Human-readable rejection reason |

**Relationships:** `CheckOrder` validates against `api.Portfolio`. Injected into
`ai.Agent` system prompt via `SetRiskInfo`.

---

## Package: memory

**Source:** `internal/memory/store.go`
**Persisted to:** `~/.nickai/memory.json`

### Entry

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| ID | `string` | `id` | Unique identifier |
| Type | `MemoryType` | `type` | `"insight"`, `"preference"`, or `"context"` |
| Content | `string` | `content` | Memory content text |
| Tags | `[]string` | `tags,omitempty` | Searchable tags |
| CreatedAt | `time.Time` | `created_at` | Creation timestamp |
| AccessedAt | `time.Time` | `accessed_at` | Last access timestamp |
| Score | `int` | `score` | Relevance score (higher = more important) |

### Store

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Entries | `[]Entry` | `entries` | List of memory entries |

**Type enum:** `MemoryType` is a `string` with constants: `"insight"` (trade learnings),
`"preference"` (user habits), `"context"` (key facts user shared).

**Relationships:** Injected into `ai.Agent` system prompt via `SetMemoryInfo`. AI tools
`save_memory` and `recall_memory` write and search entries. Pruned by score + access time.

---

## Package: journal

**Source:** `internal/journal/store.go`
**Persisted to:** `~/.nickai/journal.json`

### JournalEntry

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| ID | `string` | `id` | Unique identifier |
| OrderID | `string` | `order_id` | Corresponding order ID from PaperNick |
| Symbol | `string` | `symbol` | Trading pair (e.g. `BTCUSDT`) |
| Side | `string` | `side` | `"buy"` or `"sell"` |
| Quantity | `float64` | `quantity` | Trade quantity |
| Price | `float64` | `price` | Execution price |
| Rationale | `string` | `rationale,omitempty` | AI-generated trade reasoning |
| Source | `string` | `source` | `"ai"`, `"manual"`, `"trigger"`, or `"strategy"` |
| Timestamp | `time.Time` | `timestamp` | Trade timestamp |

**Relationships:** Created by `tools.Registry` via `JournalCh` channel after order
execution. Read by `analytics.Calculate` and the `/history` command. Used by
`tools.BuiltinTools.get_trade_journal`.

---

## Package: automation

**Source:** `internal/automation/store.go`
**Persisted to:** `~/.nickai/automations.json`

### AutoRule

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| ID | `string` | `id` | Unique identifier |
| Description | `string` | `description` | Human-readable description |
| Type | `RuleType` | `type` | `"schedule"`, `"condition"`, `"portfolio"`, or `"indicator"` |
| Schedule | `string` | `schedule,omitempty` | Schedule string (e.g. `"daily"`, `"4h"`) |
| IntervalSec | `int` | `interval_sec,omitempty` | Interval in seconds |
| Symbol | `string` | `symbol,omitempty` | Target symbol (condition rules) |
| Operator | `string` | `operator,omitempty` | Comparison operator (condition rules) |
| Target | `float64` | `target,omitempty` | Price target (condition rules) |
| MetricName | `string` | `metric_name,omitempty` | Portfolio metric name (portfolio rules) |
| Threshold | `float64` | `threshold,omitempty` | Metric threshold (portfolio rules) |
| IndicatorConditions | `[]IndicatorCondition` | `indicator_conditions,omitempty` | Technical indicator conditions |
| SourceStrategy | `string` | `source_strategy,omitempty` | Originating backtest strategy name |
| Action | `string` | `action` | Action type (e.g. `"buy"`, `"sell"`) |
| ActionSymbol | `string` | `action_symbol` | Symbol to trade |
| ActionValue | `float64` | `action_value` | Trade quantity/value |
| ActionType | `string` | `action_type` | `"market"` or `"limit"` |
| Status | `string` | `status` | `"active"`, `"paused"`, or `"completed"` |
| LastFired | `time.Time` | `last_fired,omitempty` | Last execution timestamp |
| FireCount | `int` | `fire_count` | Number of times fired |
| MaxFires | `int` | `max_fires,omitempty` | Max allowed fires (0 = unlimited) |
| CreatedAt | `time.Time` | `created_at` | Creation timestamp |
| NextCheck | `time.Time` | `next_check,omitempty` | Next scheduled check time |

### IndicatorCondition

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Indicator | `string` | `indicator` | Indicator name (rsi, macd, macd_histogram, etc.) |
| Operator | `string` | `operator` | `"<"`, `">"`, `"crosses_above"`, `"crosses_below"` |
| Value | `float64` | `value` | Threshold value |

### IndicatorSnapshot (in-memory)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| RSI | `float64` | - | RSI(14) value |
| MACD | `float64` | - | MACD line |
| MACDSignal | `float64` | - | MACD signal line |
| MACDHistogram | `float64` | - | MACD histogram |
| BollingerUpper | `float64` | - | Upper Bollinger Band |
| BollingerLower | `float64` | - | Lower Bollinger Band |
| SMA20 | `float64` | - | 20-period SMA |
| SMA50 | `float64` | - | 50-period SMA |
| EMA12 | `float64` | - | 12-period EMA |
| EMA26 | `float64` | - | 26-period EMA |
| Price | `float64` | - | Current price |

**Type enum:** `RuleType` is a `string` with constants: `"schedule"`, `"condition"`,
`"portfolio"`, `"indicator"`.

**Relationships:** Created by `tools.BuiltinTools.create_automation` and
`tools.BuiltinTools.activate_strategy`. Evaluated by the TUI's automation tick loop.
`IndicatorCondition` is structurally identical to `backtest.Condition` (minus `CompareWith`).

---

## Package: strategy

**Source:** `internal/strategy/store.go`
**Persisted to:** `~/.nickai/strategies.json`

### TWAPStrategy

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| ID | `string` | `id` | Unique identifier |
| Symbol | `string` | `symbol` | Trading pair |
| Side | `string` | `side` | `"buy"` or `"sell"` |
| TotalValue | `float64` | `total_value` | Total USD value to execute |
| Duration | `string` | `duration` | Human-readable duration (e.g. `"4h"`, `"30m"`) |
| IntervalSec | `int` | `interval_sec` | Seconds between slices |
| SliceCount | `int` | `slice_count` | Total number of slices |
| SliceValue | `float64` | `slice_value` | USD value per slice |
| Executed | `int` | `executed` | Number of slices completed |
| Status | `string` | `status` | `"active"`, `"completed"`, or `"cancelled"` |
| CreatedAt | `time.Time` | `created_at` | Creation timestamp |
| NextSliceAt | `time.Time` | `next_slice_at` | Next slice execution time |
| OrderIDs | `[]string` | `order_ids,omitempty` | Order IDs of completed slices |

**Relationships:** Created by `tools.BuiltinTools.create_twap`. Slice execution places
orders via `api.PapernickClient.PlaceOrder` and logs to `journal.Add`.

---

## Package: alert

**Source:** `internal/alert/store.go`
**Persisted to:** `~/.nickai/alerts.json`

### Alert

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Symbol | `string` | `symbol` | Trading pair |
| Operator | `string` | `operator` | `">"` or `"<"` |
| Target | `float64` | `target` | Price threshold |
| Created | `time.Time` | `created_at` | Creation timestamp |

**Relationships:** Checked by the TUI's alert tick loop against live prices from
`api.PapernickClient.GetPrices`. Triggers desktop/webhook notifications via `notify.Send`.

---

## Package: trigger

**Source:** `internal/trigger/store.go`
**Persisted to:** `~/.nickai/triggers.json`

### Trigger

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| ID | `string` | `id` | Unique identifier |
| Symbol | `string` | `symbol` | Trading pair |
| Operator | `string` | `operator` | `">"` or `"<"` |
| Target | `float64` | `target` | Price threshold |
| Action | `Action` | `action` | Trade to execute when triggered |
| CreatedAt | `time.Time` | `created_at` | Creation timestamp |
| Fired | `bool` | `fired` | Whether the trigger has fired |

### Action

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Side | `string` | `side` | `"buy"` or `"sell"` |
| Quantity | `float64` | `quantity` | Trade quantity |
| Type | `string` | `type` | `"market"` or `"limit"` |
| Price | `float64` | `price,omitempty` | Limit price (for limit orders) |

**Relationships:** Unlike `alert.Alert` (notification-only), triggers execute trades
via `api.PapernickClient.PlaceOrder` when conditions are met. Trades are logged
to `journal.Add`.

---

## Package: workflow

**Source:** `internal/workflow/store.go`
**Persisted to:** `~/.nickai/workflows.json`

### Store

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Workflows | `[]Workflow` | `workflows` | List of workflow pipelines |

### Workflow

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Name | `string` | `name` | Unique workflow name |
| Nodes | `[]Node` | `nodes` | Ordered list of pipeline steps |
| Status | `string` | `status` | `"stopped"` or `"running"` |
| RunCount | `int` | `run_count` | Number of times executed |
| LastRun | `string` | `last_run,omitempty` | RFC3339 timestamp of last run |
| CreatedAt | `string` | `created_at` | RFC3339 timestamp of creation |
| Logs | `[]LogEntry` | `logs,omitempty` | Execution logs from last run |

### Node

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| ID | `string` | `id` | Unique node ID within the workflow |
| Type | `NodeType` | `type` | Node type (see enum below) |
| Name | `string` | `name` | Human-readable node name |
| Config | `map[string]any` | `config,omitempty` | Node-specific configuration |
| ConnectsTo | `[]string` | `connects_to,omitempty` | IDs of downstream nodes |

### LogEntry

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Timestamp | `string` | `timestamp` | Execution timestamp (`HH:MM:SS.mmm`) |
| NodeID | `string` | `node_id` | Which node generated this log |
| NodeName | `string` | `node_name` | Human-readable node name |
| Status | `string` | `status` | `"started"`, `"completed"`, `"skipped"`, or `"error"` |
| Message | `string` | `message,omitempty` | Log message |

**Type enum:** `NodeType` is a `string` with constants: `"trigger"`, `"schedule"`,
`"price_feed"`, `"data"`, `"analysis"`, `"llm"`, `"condition"`, `"filter"`, `"trade"`,
`"execution"`, `"notification"`, `"webhook"`.

**Relationships:** Workflows form a DAG via `Node.ConnectsTo`. Nodes are executed in
order. Can be created from JSON files via `Store.CreateFromFile`.

---

## Package: notify

**Source:** `internal/notify/dispatch.go`
**Persisted to:** `~/.nickai/notify.json`

### Config

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Desktop | `bool` | `desktop` | Enable native desktop notifications |
| WebhookURL | `string` | `webhook_url,omitempty` | Webhook URL for push notifications |
| Sound | `bool` | `sound` | Enable notification sounds |

**Relationships:** Used by alert and trigger systems to dispatch notifications.
Desktop notifications use `osascript` on macOS and `notify-send` on Linux.
Webhook payloads are `{"title", "body", "timestamp"}` POSTed as JSON.

---

## Package: market

**Source:** `internal/market/binance.go`, `feargreed.go`, `cache.go`
**Persisted to:** `~/.nickai/cache/candles/<SYM>_<INT>_<N>.json`

### Candle (no JSON tags -- internal use)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| OpenTime | `time.Time` | - | Candle open timestamp |
| Open | `float64` | - | Open price |
| High | `float64` | - | High price |
| Low | `float64` | - | Low price |
| Close | `float64` | - | Close price |
| Volume | `float64` | - | Volume |
| CloseTime | `time.Time` | - | Candle close timestamp |

Note: When serialized to the cache file, the `cacheEntry` struct wraps `[]Candle`
and Go's default JSON serialization uses the field names in lowercase.

### cacheEntry (internal, persisted to cache)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| CachedAt | `time.Time` | `cached_at` | When the data was cached |
| Symbol | `string` | `symbol` | Trading pair |
| Interval | `string` | `interval` | Candle interval (1h, 4h, 1d) |
| Total | `int` | `total` | Requested candle count |
| Candles | `[]Candle` | `candles` | OHLCV data |

**Cache staleness:** 1h for `1h` interval, 2h for `4h`, 6h for `1d`.

### FearGreedDay (no JSON tags -- internal use)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Timestamp | `time.Time` | - | Day timestamp |
| Value | `int` | - | Fear & Greed index (0-100) |
| Label | `string` | - | Classification (e.g. "Extreme Fear", "Greed") |

**Relationships:** `Candle` data feeds `backtest.RunWithCandles`,
`indicators.Analyze`, and chart rendering. `FearGreedDay` data is used by
backtest strategies with `fear_greed` indicator conditions.

---

## Package: api

**Source:** `internal/api/papernick.go`
**Not persisted** (API response types)

### User

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| ID | `string` | `id` | User ID |
| Email | `string` | `email` | Email address |
| Name | `string` | `name` | Display name |

### Position

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Symbol | `string` | `symbol` | Trading pair |
| Quantity | `float64` | `quantity` | Total quantity |
| ReservedQuantity | `float64` | `reservedQuantity` | Reserved (in open orders) |
| AvailableQuantity | `float64` | `availableQuantity` | Available for trading |
| Value | `float64` | `value` | Current USD value |
| AvailableValue | `float64` | `availableValue` | Available USD value |

### Portfolio

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| UserID | `string` | `userId` | Owner user ID |
| Cash | `float64` | `cash` | Total cash balance |
| ReservedCash | `float64` | `reservedCash` | Cash reserved for open orders |
| AvailableCash | `float64` | `availableCash` | Cash available for trading |
| Assets | `[]Position` | `assets` | Portfolio positions |
| TotalValue | `float64` | `totalValue` | Total portfolio value (cash + assets) |

### Cash

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Cash | `float64` | `cash` | Total cash |
| ReservedCash | `float64` | `reservedCash` | Reserved cash |
| AvailableCash | `float64` | `availableCash` | Available cash |

### Order

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| ID | `string` | `id` | Order ID |
| Symbol | `string` | `symbol` | Trading pair |
| Side | `string` | `side` | `"buy"` or `"sell"` |
| Type | `string` | `type` | `"market"` or `"limit"` |
| Quantity | `float64` | `quantity` | Order quantity |
| Price | `float64` | `price` | Requested price |
| FilledPrice | `float64` | `filledPrice` | Actual fill price |
| Status | `string` | `status` | Order status |
| FilledAt | `string` | `filledAt` | Fill timestamp |
| OrderClass | `string` | `orderClass` | Order classification |

### Price

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Symbol | `string` | `symbol` | Trading pair |
| Price | `float64` | `price` | Current price |

### Symbol

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Symbol | `string` | `symbol` | Trading pair identifier |
| Name | `string` | `name` | Human-readable name |

### PlaceOrderRequest

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Symbol | `string` | `symbol` | Trading pair |
| Quantity | `float64` | `quantity` | Order quantity |
| Side | `string` | `side` | `"buy"` or `"sell"` |
| Type | `string` | `type` | `"market"` or `"limit"` |
| Price | `float64` | `price,omitempty` | Limit price (omitted for market orders) |

### CreateAccountUser

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| ID | `string` | `id` | New user ID |
| APIKey | `string` | `apiKey` | Generated API key |
| Name | `string` | `name` | Account name |
| Description | `string` | `description` | Account description |
| Cash | `float64` | `cash` | Starting cash balance |

### CreateAccountResponse

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| User | `CreateAccountUser` | `user` | Created user object |

### APIError

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| StatusCode | `int` | - | HTTP status code |
| Message | `string` | `message` | Error message from API |
| Body | `string` | - | Raw response body |

**Relationships:** `PapernickClient` is the primary trading API client. Used by
`tools.BuiltinTools` (get_prices, get_portfolio, place_order), `risk.CheckOrder`,
`analytics.CalcAllocation`. Auth via `X-API-Key` header.

---

## Package: ai

**Source:** `internal/ai/agent.go`, `openrouter.go`, `consensus.go`
**Not persisted** (runtime state)

### ModelOption (in-memory)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| ID | `string` | - | Model identifier (e.g. `claude-sonnet`) |
| Name | `string` | - | Display name (e.g. `Claude Sonnet 4.6`) |
| Provider | `Provider` | - | `"anthropic"`, `"openrouter"`, or `"minimax"` |
| Free | `bool` | - | Whether the model is free tier |

### Agent (in-memory, runtime)

Core AI agent managing conversation history and tool-use loops. Not persisted.
Key internal fields: `history []chatMessage`, `modelID`, `provider`, `registry *tools.Registry`.

### Anthropic API Wire Types (internal)

- **chatMessage:** `{role, content}` where content is `string | []contentBlock | []toolResultBlock`
- **contentBlock:** `{type, text, id, name, input}` -- text or tool_use blocks
- **toolResultBlock:** `{type:"tool_result", tool_use_id, content, is_error}`
- **apiRequest:** `{model, max_tokens, system, tools, messages, temperature}`
- **apiResponse:** `{id, type, role, content, stop_reason}`
- **streamAPIRequest:** same as apiRequest + `stream: true`
- **sseEvent:** `{type, index, content_block, delta}` -- SSE stream events

### OpenRouter Types (internal)

- **orRequest:** `{model, messages, temperature, stream}`
- **orMessage:** `{role, content}`
- **orResponse:** `{choices: [{message}]}`
- **orStreamChunk:** `{choices: [{delta: {content}, finish_reason}]}`

### ModelVerdict

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Model | `string` | `model` | Model identifier |
| Verdict | `string` | `verdict` | `"BUY"`, `"SELL"`, or `"HOLD"` |
| Confidence | `string` | `confidence` | `"High"`, `"Medium"`, or `"Low"` |
| Reasoning | `string` | `reasoning` | 1-2 sentence explanation |
| Error | `string` | `error,omitempty` | Error message if model failed |
| Duration | `time.Duration` | `duration_ms` | Response latency |

### ConsensusResult

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Verdicts | `[]ModelVerdict` | `verdicts` | Individual model verdicts |
| Consensus | `string` | `consensus` | `"BUY"`, `"SELL"`, `"HOLD"`, or `"NO_CONSENSUS"` |
| Agreement | `string` | `agreement` | Agreement ratio (e.g. `"3/4"`) |
| Symbol | `string` | `symbol` | Analyzed symbol |
| Price | `float64` | `price` | Price at time of analysis |

### ConsensusConfig

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Models | `[]string` | `models` | List of model identifiers to query |
| Threshold | `float64` | `threshold` | Agreement threshold (e.g. 0.67) |

**Relationships:** `Agent` uses `tools.Registry` for tool execution, `api.PapernickClient`
for trading, and multiple LLM providers. `ConsensusResult` is rendered by the `/consensus`
TUI command. Tier 1 models (4 frontier), Tier 2 (4 diversity), Tier 3 (2 budget/free).

---

## Package: backtest

**Source:** `internal/backtest/engine.go`, `montecarlo.go`, `walkforward.go`, `optimizer.go`, `presets.go`
**Not persisted** (computed results, but serialized to JSON for AI analysis)

### Strategy

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Name | `string` | `name,omitempty` | Strategy name |
| Symbol | `string` | `symbol` | Trading pair |
| Side | `string` | `side,omitempty` | `"long"` (default), `"short"`, or `"both"` |
| EntryRules | `[]Condition` | `entry_conditions` | Entry signal conditions (AND logic) |
| ExitRules | `[]Condition` | `exit_conditions` | Exit signal conditions |
| StopLossPct | `float64` | `stop_loss_pct,omitempty` | Stop loss percentage |
| TakeProfitPct | `float64` | `take_profit_pct,omitempty` | Take profit percentage |
| PositionSize | `float64` | `position_size,omitempty` | Position size fraction (0-1, default 1.0) |
| Period | `string` | `period,omitempty` | Lookback period (e.g. `"180d"`, `"1y"`) |
| SlippageBps | `float64` | `slippage_bps,omitempty` | Slippage in basis points |
| CommissionBps | `float64` | `commission_bps,omitempty` | Commission in basis points per trade |
| ExitLogic | `string` | `exit_logic,omitempty` | `"and"` (default) or `"or"` for exit conditions |

### Condition

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Indicator | `string` | `indicator` | Indicator name (see list below) |
| Operator | `string` | `operator` | `"<"`, `">"`, `"crosses_above"`, `"crosses_below"` |
| Value | `float64` | `value` | Static threshold |
| CompareWith | `string` | `compare_with,omitempty` | Compare against another indicator instead of Value |

**Supported indicators:** `rsi`, `macd`, `macd_histogram`, `macd_signal`,
`bollinger_upper`, `bollinger_lower`, `sma20`, `sma50`, `ema12`, `ema26`, `price`,
`fear_greed`, `trend`, `momentum`, `vol_regime`, `drawdown`, `dir_volume`.

### Result

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Strategy | `string` | `strategy` | Strategy name |
| Symbol | `string` | `symbol` | Trading pair |
| Period | `string` | `period` | Lookback period |
| Trades | `[]Trade` | `trades` | Individual trade records |
| TotalTrades | `int` | `total_trades` | Number of trades |
| WinRate | `float64` | `win_rate` | Win rate percentage |
| TotalReturn | `float64` | `total_return` | Total return percentage |
| SharpeRatio | `float64` | `sharpe_ratio` | Annualized Sharpe ratio |
| MaxDrawdown | `float64` | `max_drawdown` | Maximum drawdown percentage |
| ProfitFactor | `float64` | `profit_factor` | Gross profit / gross loss |
| BestTrade | `float64` | `best_trade` | Best trade PnL percentage |
| WorstTrade | `float64` | `worst_trade` | Worst trade PnL percentage |
| EquityCurve | `[]float64` | `equity_curve` | Equity curve (normalized, starts at 1.0) |
| SortinoRatio | `float64` | `sortino_ratio` | Sortino ratio |
| CalmarRatio | `float64` | `calmar_ratio` | Calmar ratio |
| OmegaRatio | `float64` | `omega_ratio` | Omega ratio |
| RecoveryFactor | `float64` | `recovery_factor` | Recovery factor |
| Expectancy | `float64` | `expectancy` | Expected PnL per trade |
| TailRatio | `float64` | `tail_ratio` | Right tail / left tail ratio |
| VaR95 | `float64` | `var_95` | 95th percentile Value at Risk |
| CVaR95 | `float64` | `cvar_95` | 95th percentile Conditional VaR |
| MaxDDDuration | `int` | `max_dd_duration` | Max drawdown duration in bars |
| TimeInMarketPct | `float64` | `time_in_market_pct` | Percentage of time in a position |
| AvgTradeBars | `float64` | `avg_trade_bars` | Average trade duration in bars |
| HODLReturn | `float64` | `hodl_return` | Buy-and-hold return for comparison |
| DCAReturn | `float64` | `dca_return` | DCA return for comparison |
| MonteCarlo | `*MonteCarloResult` | `monte_carlo,omitempty` | Monte Carlo simulation results |

### Trade

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| EntryTime | `time.Time` | `entry_time` | Entry timestamp |
| ExitTime | `time.Time` | `exit_time` | Exit timestamp |
| EntryPrice | `float64` | `entry_price` | Entry price |
| ExitPrice | `float64` | `exit_price` | Exit price |
| PnLPct | `float64` | `pnl_pct` | Profit/loss percentage |
| Reason | `string` | `reason` | `"exit_signal"`, `"stop_loss"`, `"take_profit"`, `"period_end"` |

### MonteCarloResult

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Simulations | `int` | `simulations` | Number of simulations run (typically 1000) |
| OriginalSharpe | `float64` | `original_sharpe` | Sharpe ratio of the original backtest |
| OriginalMaxDD | `float64` | `original_max_dd` | Max drawdown of the original backtest |
| PValue | `float64` | `p_value` | Fraction of sims with Sharpe >= original |
| MedianSharpe | `float64` | `median_sharpe` | Median Sharpe across simulations |
| MedianMaxDD | `float64` | `median_max_dd` | Median max drawdown across simulations |
| DD95 | `float64` | `dd_95` | 95th percentile max drawdown |
| DD99 | `float64` | `dd_99` | 99th percentile max drawdown |
| SharpeLower95 | `float64` | `sharpe_lower_95` | 2.5th percentile Sharpe (95% CI lower) |
| SharpeUpper95 | `float64` | `sharpe_upper_95` | 97.5th percentile Sharpe (95% CI upper) |

### WFAConfig (in-memory)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Windows | `int` | - | Number of IS/OOS windows (default 5) |
| OOSRatio | `float64` | - | Fraction for OOS (default 0.3) |
| Anchored | `bool` | - | Anchored (IS grows from start) vs rolling |

### WFAResult (in-memory)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Windows | `[]WFAWindow` | - | Per-window results |
| CombinedOOS | `*Result` | - | Concatenated OOS equity curve result |
| Efficiency | `float64` | - | OOS_Sharpe / IS_Sharpe (>0.5 = generalizes) |
| Robust | `bool` | - | True if Efficiency > 0.5 |

### WFAWindow (in-memory)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| WindowNum | `int` | - | Window number (1-indexed) |
| ISStart | `int` | - | In-sample start candle index |
| ISEnd | `int` | - | In-sample end candle index |
| OOSStart | `int` | - | Out-of-sample start candle index |
| OOSEnd | `int` | - | Out-of-sample end candle index |
| ISResult | `*Result` | - | In-sample backtest result |
| OOSResult | `*Result` | - | Out-of-sample backtest result |

### ParamRange (in-memory)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Name | `string` | - | Parameter name (e.g. `"stop_loss_pct"`, `"rsi_entry"`) |
| Values | `[]float64` | - | Explicit list of values to try |

### OptimizeConfig (in-memory)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Params | `[]ParamRange` | - | Parameter search ranges |
| Metric | `string` | - | Metric to maximize: `"sharpe"`, `"sortino"`, `"total_return"`, `"calmar"` |
| MaxCombos | `int` | - | Safety limit (default 1000) |

### OptimizeResult (in-memory)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| BestParams | `map[string]float64` | - | Best parameter combination |
| BestMetric | `float64` | - | Best metric value |
| BestResult | `*Result` | - | Full backtest result for best params |
| TotalCombos | `int` | - | Total combinations evaluated |
| TopN | `[]OptimizeEntry` | - | Top 10 results |
| Duration | `time.Duration` | - | Total optimization time |

### OptimizeEntry (in-memory)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Params | `map[string]float64` | - | Parameter combination |
| Metric | `float64` | - | Metric value |
| Result | `*Result` | - | Full backtest result |

### PresetStrategy (in-memory)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Strategy | `Strategy` | - | Pre-configured strategy |
| Description | `string` | - | Human-readable description |

12 presets: `rsi-reversal`, `macd-crossover`, `bollinger-bounce`, `golden-cross`,
`momentum`, `fear-and-greed`, `dip-buyer`, `rsi-short`, `macd-short`,
`and-tre-mom-dir`, `and-tre-mom`, `calm-trend`.

### AnalysisPreset (in-memory)

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Name | `string` | - | Preset name |
| Description | `string` | - | Human-readable description |
| Prompt | `string` | - | Structured AI prompt |
| MCPTools | `[]string` | - | Required MCP server names |

5 presets: `polymarket-scan`, `polymarket-deep`, `sentiment-check`, `whale-watch`, `defi-yield`.

**Relationships:** `Strategy` uses `market.Candle` data and `indicators` for
signals. `Result` is serialized to JSON for `/backtest analyze` (sent to Claude).
`MonteCarloResult` embedded in `Result`. `PresetStrategy` wraps `Strategy`.

---

## Package: indicators

**Source:** `internal/indicators/ta.go`
**Not persisted** (computed on the fly)

### Analysis

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Symbol | `string` | `symbol` | Trading pair |
| Price | `float64` | `price` | Current price |
| RSI | `float64` | `rsi` | RSI(14) value |
| RSISignal | `string` | `rsi_signal` | `"overbought"`, `"oversold"`, or `"neutral"` |
| MACD | `float64` | `macd` | MACD line |
| MACDSignal | `float64` | `macd_signal` | MACD signal line |
| MACDHistogram | `float64` | `macd_histogram` | MACD histogram |
| MACDTrend | `string` | `macd_trend` | `"bullish"` or `"bearish"` |
| BollingerUpper | `float64` | `bollinger_upper` | Upper Bollinger Band |
| BollingerLower | `float64` | `bollinger_lower` | Lower Bollinger Band |
| BollingerPos | `string` | `bollinger_position` | `"above"`, `"middle"`, or `"below"` |
| SMA20 | `float64` | `sma_20` | 20-period SMA |
| SMA50 | `float64` | `sma_50` | 50-period SMA |
| Trend | `string` | `trend` | `"bullish"`, `"bearish"`, or `"neutral"` |
| FearGreed | `int` | `fear_greed_index` | Fear & Greed Index (0-100) |
| FearGreedLabel | `string` | `fear_greed_label` | Classification string |
| Summary | `string` | `summary` | AI-friendly analysis summary |

### Streaming Indicators (in-memory, O(1) per update)

- **StreamEMA:** period, k, value, count, sum
- **StreamRSI:** period, avgGain, avgLoss, prev, count
- **StreamSMA:** period, circular buffer, idx, sum, count
- **StreamMACD:** fast EMA + slow EMA + signal EMA
- **StreamBollinger:** SMA + circular buffer for stddev

**Relationships:** `Analysis` is returned by `tools.BuiltinTools.analyze_market`.
Streaming indicators are used by `backtest.computeSnapshots` for O(1) per-candle
backtest simulation.

---

## Package: tools

**Source:** `internal/tools/registry.go`, `builtin.go`
**Not persisted** (runtime registry)

### ToolEntry

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Name | `string` | - | Tool name (e.g. `get_prices`) |
| Description | `string` | - | Tool description for LLM |
| InputSchema | `json.RawMessage` | - | JSON Schema for input parameters |
| Execute | `ToolFunc` | - | Executor function |
| Source | `string` | - | `"builtin"` or MCP server name |

### AnthropicToolDef

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Name | `string` | `name` | Tool name |
| Description | `string` | `description` | Tool description |
| InputSchema | `json.RawMessage` | `input_schema` | JSON Schema |

### ConfirmRequest

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| ToolName | `string` | - | Name of the tool requesting confirmation |
| Input | `json.RawMessage` | - | Raw tool input |
| Display | `string` | - | Pre-rendered confirmation text for the user |

### ConfirmResponse

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Approved | `bool` | - | Whether the user approved |

### Registry (in-memory)

| Field | Type | Description |
|-------|------|-------------|
| entries | `map[string]*ToolEntry` | Tool lookup map |
| order | `[]string` | Insertion-order tool names |
| ConfirmCh | `chan ConfirmRequest` | Tool -> UI confirmation channel |
| ResponseCh | `chan ConfirmResponse` | UI -> tool response channel |
| JournalCh | `chan journal.JournalEntry` | Trade journal entries channel (buffered 10) |

**Relationships:** `Registry` holds both builtin tools (get_prices, place_order, etc.)
and MCP-discovered tools. `ai.Agent` calls `Registry.ExecuteTool` during tool-use loops.
Tool results are truncated to 16KB to prevent context window overflow.

---

## Package: analytics

**Source:** `internal/analytics/metrics.go`
**Not persisted** (computed on the fly)

### Metrics

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| SharpeRatio | `float64` | - | Annualized Sharpe ratio |
| MaxDrawdownPct | `float64` | - | Maximum drawdown percentage |
| WinRate | `float64` | - | Win rate percentage |
| ProfitFactor | `float64` | - | Gross profit / gross loss |
| TotalTrades | `int` | - | Total number of trades |
| WinCount | `int` | - | Number of winning trades |
| LossCount | `int` | - | Number of losing trades |
| TotalPnL | `float64` | - | Total P&L in USD |
| BestTrade | `float64` | - | Best trade P&L in USD |
| WorstTrade | `float64` | - | Worst trade P&L in USD |
| AvgWin | `float64` | - | Average winning trade |
| AvgLoss | `float64` | - | Average losing trade |

### Allocation

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Symbol | `string` | - | Asset symbol (or `"CASH"`) |
| Value | `float64` | - | USD value |
| Percent | `float64` | - | Percentage of total portfolio |

**Relationships:** `Calculate` takes `[]journal.JournalEntry` + current prices.
`CalcAllocation` takes `*api.Portfolio`. Both are used by the `/analytics` command
and the `get_analytics` tool.

---

## Package: guidance

**Source:** `internal/guidance/stage.go`
**Not persisted** (computed on the fly from runtime state)

### Stage (type alias: `string`)

Constants: `"fresh"`, `"configured"`, `"ai_ready"`, `"equipped"`, `"trading"`,
`"analyzing"`, `"advanced"`.

### ActionCard

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| Icon | `string` | - | Emoji icon |
| Title | `string` | - | Card title |
| Desc | `string` | - | Card description |
| Command | `string` | - | Suggested command to run |

### StageContext

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| HasAPIKey | `bool` | - | Whether PaperNick API key is set |
| HasAIKey | `bool` | - | Whether Anthropic API key is set |
| MCPCount | `int` | - | Number of connected MCP servers |
| TradeCount | `int` | - | Number of trades executed |
| HasAnalyzed | `bool` | - | Whether user has used `/analyze` |
| HasBacktested | `bool` | - | Whether user has used `/backtest` |
| MemoryCount | `int` | - | Number of AI memory entries |
| PortfolioValue | `float64` | - | Total portfolio value |
| CashBalance | `float64` | - | Cash balance |
| TopPositions | `[]string` | - | Top position symbol names |

**Relationships:** `DetectStage` determines the user's journey stage.
`ActionsForStage` returns contextual next-step suggestions. Injected into
`ai.Agent` system prompt via `SetGuidanceContext`.

---

## Package: personality

**Source:** `internal/personality/vibes.go`
**Not persisted** (built-in list; active vibe stored in `config.Config.Vibe`)

### Vibe

| Field | Type | JSON Tag | Description |
|-------|------|----------|-------------|
| ID | `string` | - | Vibe identifier (e.g. `"degen"`) |
| Name | `string` | - | Display name (e.g. `"Degen Nick"`) |
| Emoji | `string` | - | Emoji icon |
| Tagline | `string` | - | Short tagline |
| Prompt | `string` | - | System prompt personality section |
| Greetings | `[]string` | - | Random greeting messages |

6 vibes: `degen`, `quant`, `zen`, `hype`, `sensei`, `degen-bets`.

**Relationships:** Selected vibe's `Prompt` is injected into `ai.Agent.systemPrompt`.
Active vibe ID is persisted in `config.Config.Vibe`.

---

## Package: node/pb

**Source:** `internal/node/pb/node.go`, `service.go`
**Not persisted** (gRPC wire types, JSON codec)

### Ping

| Type | Fields |
|------|--------|
| `PingRequest` | *(empty)* |
| `PingResponse` | `Version string`, `UptimeSeconds int64` |

### Strategy Types

| Type | Key Fields |
|------|------------|
| `StrategyCondition` | `Indicator`, `Operator`, `Value`, `CompareWith` |
| `StrategySpec` | `Name`, `Symbol`, `EntryRules`, `ExitRules`, `StopLossPct`, `TakeProfitPct`, `PositionSize`, `Interval` |
| `StrategyInfo` | `ID`, `Spec`, `Status` (enum: RUNNING/STOPPED/ERRORED), `DeployedAt`, `Error` |
| `DeployStrategyRequest` | `Spec *StrategySpec` |
| `DeployStrategyResponse` | `ID string` |
| `ListStrategiesRequest` | *(empty)* |
| `ListStrategiesResponse` | `Strategies []*StrategyInfo` |
| `StopStrategyRequest` | `ID string` |
| `StopStrategyResponse` | `Stopped bool` |

### Price Streaming

| Type | Key Fields |
|------|------------|
| `StreamPricesRequest` | `Symbols []string` |
| `PriceTick` | `Symbol`, `Price`, `Volume24H`, `Timestamp` |

### Backtest Types

| Type | Key Fields |
|------|------------|
| `BacktestCondition` | `Indicator`, `Operator`, `Value`, `CompareWith` |
| `BacktestSpec` | `Name`, `Symbol`, `EntryRules`, `ExitRules`, `StopLossPct`, `TakeProfitPct`, `PositionSize`, `Period`, `SlippageBps`, `CommissionBps` |
| `BacktestTrade` | `EntryTime`, `ExitTime`, `EntryPrice`, `ExitPrice`, `PnLPct`, `Reason` |
| `BacktestResult` | `Strategy`, `Symbol`, `Period`, `Trades`, `TotalTrades`, `WinRate`, `TotalReturn`, `SharpeRatio`, `MaxDrawdown`, `ProfitFactor`, `BestTrade`, `WorstTrade`, `EquityCurve` |
| `SubmitBacktestRequest` | `Spec *BacktestSpec` |
| `SubmitBacktestResponse` | `JobID string` |
| `GetBacktestResultRequest` | `JobID string` |
| `GetBacktestResultResponse` | `Status` (enum: PENDING/RUNNING/COMPLETED/FAILED), `Result`, `Error` |

### Alert Types

| Type | Key Fields |
|------|------------|
| `AlertSpec` | `Symbol`, `Operator`, `Target` |
| `AlertInfo` | `ID`, `Spec`, `CreatedAt`, `Triggered` |
| `CreateAlertRequest` | `Spec *AlertSpec` |
| `CreateAlertResponse` | `ID string` |
| `ListAlertsRequest` | *(empty)* |
| `ListAlertsResponse` | `Alerts []*AlertInfo` |

### Status

| Type | Key Fields |
|------|------------|
| `GetStatusRequest` | *(empty)* |
| `GetStatusResponse` | `Version`, `UptimeSeconds`, `RunningStrategies`, `ActiveAlerts`, `ConnectedSymbols`, `MemoryBytes`, `Goroutines` |

**Service interface:** `NickNodeServer` with 10 RPCs: `Ping`, `DeployStrategy`,
`ListStrategies`, `StopStrategy`, `StreamPrices` (server-streaming), `SubmitBacktest`,
`GetBacktestResult`, `CreateAlert`, `ListAlerts`, `GetStatus`.

**Wire format:** JSON codec (`encoding/json`) over gRPC, not protobuf binary.
Proto file at `proto/nick/v1/node.proto` (for future `protoc` generation).

---

## Package: node

**Source:** `internal/node/server.go`, `client.go`
**Not persisted** (runtime state)

### Server (in-memory)

Implements `pb.NickNodeServer`. Manages running strategies, backtest jobs, and
alerts in memory with mutex-protected maps.

### Client

| Field | Type | Description |
|-------|------|-------------|
| conn | `*grpc.ClientConn` | gRPC connection |
| client | `pb.NickNodeClient` | gRPC client stub |
| addr | `string` | Server address (default: `localhost:9400`) |

### RunningStrategy (in-memory)

| Field | Type | Description |
|-------|------|-------------|
| Info | `*pb.StrategyInfo` | Strategy metadata and status |
| cancel | `context.CancelFunc` | Cancellation handle |

### backtestJob (in-memory)

| Field | Type | Description |
|-------|------|-------------|
| ID | `string` | Job identifier |
| Spec | `*pb.BacktestSpec` | Job specification |
| Status | `pb.BacktestJobStatus` | Current status |
| Result | `*pb.BacktestResult` | Results (when completed) |
| Error | `string` | Error message (when failed) |

**Relationships:** `node.Client` wraps `pb.NickNodeClient` for TUI usage.
`node.Server` uses `backtest.RunWithCandles` for offloaded backtest jobs.
Binary: `cmd/node/main.go` builds `nickai-node`.

---

## Package: commands

**Source:** `internal/commands/defs.go`, `router.go`
**Not persisted** (compile-time registry)

### CommandType (type alias: `int`, iota enum)

50+ constants: `TypeChat`, `TypeHelp`, `TypeStatus`, `TypeOrders`, `TypePrice`,
`TypeBuy`, `TypeSell`, `TypeConfig`, `TypeClear`, `TypeQuit`, `TypeCredential`,
`TypeWorkflow`, `TypeLogs`, `TypeMan`, `TypeWatch`, `TypeSnapshot`, `TypeMarket`,
`TypePnl`, `TypeHistory`, `TypeAlert`, `TypeChart`, `TypeTheme`, `TypeModel`,
`TypeMCP`, `TypeTrigger`, `TypeRisk`, `TypeStrategy`, `TypeNotify`, `TypeAnalytics`,
`TypeAnalyze`, `TypeAuto`, `TypeBacktest`, `TypePolymarket`, `TypeGuide`, `TypeMemory`,
`TypeConsensus`, `TypeConnect`, `TypeBalances`, `TypePositions`, `TypeMarkets`,
`TypeBet`, `TypeWallet`, `TypeSwap`, `TypeGas`, `TypeStock`, `TypeScreen`,
`TypeOdds`, `TypeLines`, `TypeFunding`, `TypeDashboard`, `TypeVibe`, `TypeExport`,
`TypePlugin`, `TypeNode`, `TypeUnknown`.

### CommandDef

| Field | Type | Description |
|-------|------|-------------|
| Type | `CommandType` | Command type enum |
| Primary | `string` | Primary command (e.g. `"/buy"`) |
| Aliases | `[]string` | Alternative names (e.g. `["/b"]`) |
| Description | `string` | Short description |
| Category | `string` | `"Trading"`, `"Analysis"`, `"Tools"`, `"Setup"`, `"Multi-Vertical"` |
| SubCommands | `[]SubCommandDef` | Child commands |

### SubCommandDef

| Field | Type | Description |
|-------|------|-------------|
| Name | `string` | Subcommand name (e.g. `"list"`) |
| Description | `string` | Subcommand description |

### Result (in-memory)

| Field | Type | Description |
|-------|------|-------------|
| Type | `CommandType` | Parsed command type |
| SubCommand | `string` | Parsed subcommand (e.g. `"scan"`) |
| Input | `string` | Original user input |
| Args | `[]string` | Remaining arguments |
| IsCommand | `bool` | True if input started with `/` |

**Relationships:** `Registry` is the canonical list of all commands. `BuildCommandMap`
generates the routing map. `PaletteEntries` generates the Ctrl+K palette list.
`Route` parses user input into `Result`. Commands with subcommand parsing enabled:
`TypeConnect`, `TypeWallet`, `TypeMarkets`, `TypeNode`.

---

## Package: mock

**Source:** `internal/mock/data.go`
**Not persisted** (demo/testing data)

### Agent

| Field | Type | Description |
|-------|------|-------------|
| Name | `string` | Agent name |
| Strategy | `string` | Strategy description |
| Status | `AgentStatus` | `StatusRunning`, `StatusStopped`, `StatusError` |
| PnL | `string` | P&L display string |
| Uptime | `string` | Uptime display string |

### Template

| Field | Type | Description |
|-------|------|-------------|
| Name | `string` | Template name |
| Description | `string` | Description |
| Author | `string` | Author name |
| Stars | `int` | Star count |
| Tags | `[]string` | Tags |

---

## Example JSON Snippets

### ~/.nickai/config.json

```json
{
  "base_url": "https://paper.getnick.ai/api/v1",
  "theme": "tokyo-night",
  "model": "claude-sonnet",
  "vibe": "degen",
  "data_keys": {
    "openrouter": "sk-or-..."
  }
}
```

Note: `api_key` and `anthropic_key` fields are absent when stored in OS keyring.

### ~/.nickai/credentials.json

```json
{
  "credentials": [
    {
      "name": "my-binance",
      "exchange": "binance",
      "api_key": "abc123...",
      "api_secret": "def456..."
    }
  ]
}
```

### ~/.nickai/mcp.json

```json
{
  "mcpServers": {
    "polymarket": {
      "command": "npx",
      "args": ["-y", "graph-polymarket-mcp"]
    },
    "brave-search": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-brave-search"],
      "env": {
        "BRAVE_API_KEY": "BSA..."
      }
    }
  }
}
```

### ~/.nickai/risk.json

```json
{
  "max_position_pct": 20,
  "daily_loss_pct": 5,
  "max_order_value": 10000
}
```

### ~/.nickai/memory.json

```json
{
  "entries": [
    {
      "id": "a1b2c3d4",
      "type": "preference",
      "content": "User prefers 2-5% position sizes and RSI-based entries",
      "tags": ["risk", "strategy"],
      "created_at": "2026-03-01T10:00:00Z",
      "accessed_at": "2026-03-03T14:30:00Z",
      "score": 5
    }
  ]
}
```

### ~/.nickai/journal.json

```json
[
  {
    "id": "j-001",
    "order_id": "ord-abc",
    "symbol": "BTCUSDT",
    "side": "buy",
    "quantity": 0.01,
    "price": 95000,
    "rationale": "RSI oversold at 28, MACD bullish crossover. Good R:R.",
    "source": "ai",
    "timestamp": "2026-03-02T15:30:00Z"
  }
]
```

### ~/.nickai/triggers.json

```json
[
  {
    "id": "trig-001",
    "symbol": "ETHUSDT",
    "operator": "<",
    "target": 3000,
    "action": {
      "side": "buy",
      "quantity": 1.0,
      "type": "market"
    },
    "created_at": "2026-03-01T12:00:00Z",
    "fired": false
  }
]
```

### ~/.nickai/automations.json

```json
[
  {
    "id": "auto-001",
    "description": "Daily DCA into BTC",
    "type": "schedule",
    "schedule": "daily",
    "interval_sec": 86400,
    "action": "buy",
    "action_symbol": "BTCUSDT",
    "action_value": 100,
    "action_type": "market",
    "status": "active",
    "fire_count": 3,
    "created_at": "2026-02-28T08:00:00Z",
    "next_check": "2026-03-04T08:00:00Z"
  }
]
```

### ~/.nickai/workflows.json

```json
{
  "workflows": [
    {
      "name": "morning-scan",
      "nodes": [
        {"id": "n1", "type": "schedule", "name": "Daily 9 AM", "connects_to": ["n2"]},
        {"id": "n2", "type": "data", "name": "Fetch BTC price", "connects_to": ["n3"]},
        {"id": "n3", "type": "analysis", "name": "Run TA", "connects_to": ["n4"]},
        {"id": "n4", "type": "notification", "name": "Send alert"}
      ],
      "status": "stopped",
      "run_count": 0,
      "created_at": "2026-03-01T10:00:00Z"
    }
  ]
}
```

### ~/.nickai/notify.json

```json
{
  "desktop": true,
  "webhook_url": "https://hooks.slack.com/...",
  "sound": false
}
```

---

## Relationship Diagram

```
config.Config ──────────────> credential (keyring integration)
     |
     ├── api.PapernickClient (baseURL, apiKey)
     ├── ai.Agent (anthropicKey, model, vibe)
     └── mcp.ClientManager (loads mcp.json)

ai.Agent
     ├── tools.Registry (builtin + MCP tools)
     ├── api.PapernickClient (get_prices, place_order)
     ├── personality.Vibe (system prompt)
     ├── memory.Store (context injection)
     ├── risk.RiskLimits (prompt suffix)
     └── guidance.StageContext (journey stage)

tools.Registry
     ├── api.PapernickClient (trading tools)
     ├── journal.JournalEntry (trade logging via JournalCh)
     ├── indicators.Analysis (analyze_market)
     ├── backtest.Strategy/Result (backtest_strategy)
     ├── automation.AutoRule (create_automation, activate_strategy)
     ├── strategy.TWAPStrategy (create_twap)
     ├── analytics.Metrics (get_analytics)
     └── memory.Store (save_memory, recall_memory)

backtest.Result
     ├── backtest.Trade[]
     ├── backtest.MonteCarloResult (embedded)
     ├── market.Candle[] (input data)
     └── indicators.Stream* (O(1) computation)

node.Server (gRPC)
     ├── pb.StrategySpec (deployed strategies)
     ├── pb.BacktestSpec -> backtest.RunWithCandles
     ├── pb.AlertSpec (server-side alerts)
     └── pb.PriceTick (streaming prices)

alert.Alert ──> notify.Send (notification dispatch)
trigger.Trigger ──> api.PlaceOrder + journal.Add (trade execution)
automation.AutoRule ──> api.PlaceOrder + journal.Add (automated trades)
strategy.TWAPStrategy ──> api.PlaceOrder + journal.Add (sliced execution)
```
