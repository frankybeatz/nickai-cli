# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.4.x   | Yes       |
| 0.3.x   | Yes       |
| < 0.3   | No        |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly.

**Do not open a public issue.** Instead, email **security@getnick.ai** or use [GitHub private vulnerability reporting](https://github.com/frankybeatz/nickai-cli/security/advisories/new) with:

- Description of the vulnerability
- Steps to reproduce
- Impact assessment
- Suggested fix (if any)

You should receive an acknowledgment within 48 hours. We aim to release a patch within 7 days of confirmation.

## Security Model

NickAI CLI is a **local terminal application**. The core app does not run a server, accept inbound connections, or expose any network surface.

With **MCP integrations**, NickAI can connect to external services — including live exchanges — so real funds may be at risk depending on which servers you enable. Read the [MCP Security](#mcp-security) section carefully before connecting trading servers.

### Data Protection

- **Atomic file writes** — All 12 data stores use write-to-temp-then-rename to prevent corruption on crash or power loss
- **Per-store mutex** — Concurrent access to any store file is serialized to prevent race conditions and data loss
- **File permissions** — All config, credentials, and data files stored with `0600` (owner-read/write only), directories `0700`
- **No plaintext logging of secrets** — Debug logs (`--debug`) never include API keys or credentials

### Credentials

- API keys stored locally at `~/.nickai/config.json` and `~/.nickai/credentials.json` with `0600` permissions
- Keys are never logged, committed, or transmitted except to their intended API endpoints over HTTPS
- Keys are masked in all display output (first 4 + last 4 characters shown)
- `crypto/rand` used for all ID generation (no `math/rand` anywhere in the codebase)

### Network

- All API communication uses HTTPS with TLS 1.2+ minimum enforced
- PaperNick API: `https://paper.getnick.ai/api/v1`
- Anthropic API: `https://api.anthropic.com/v1/messages`
- OpenRouter API: `https://openrouter.ai/api/v1/chat/completions`
- MCP servers may make additional outbound connections to their respective APIs
- URL parameters are properly encoded to prevent injection

### AI Safety

- **Tool result truncation** — MCP tool responses capped at 16KB to prevent context window overflow
- **Tool execution timeout** — 30-second deadline on every tool call prevents hung operations
- **Context cancellation** — All HTTP requests use `context.Context` so in-flight calls can be cancelled
- **Conversation pruning** — History automatically pruned when approaching 80% of context window limits
- **Rate limit handling** — 429 responses trigger automatic backoff with Retry-After header parsing
- **Retry logic** — Both streaming and non-streaming API calls retry on transient errors (5xx, connection resets)

### Dependencies

- Core dependencies are from [Charmbracelet](https://github.com/charmbracelet) (widely trusted, open-source TUI libraries)
- Dependencies are pinned in `go.sum` with cryptographic checksums
- Audited with `govulncheck` — zero known vulnerabilities

### Built-in Trading (PaperNick)

- Built-in `/buy` and `/sell` commands use the PaperNick paper-trading API — **no real funds**
- Every built-in trade requires **user confirmation** before execution
- Risk guardrails (`/risk`) enforce max order size, position limits, and daily loss caps on all built-in trades

### What the Core App Does NOT Do

- Does not collect telemetry or analytics
- Does not auto-update or download code at runtime
- Does not open any inbound ports or accept connections

## MCP Security

[Model Context Protocol](https://modelcontextprotocol.io) integrations extend NickAI with external tools. Some of these tools **can access real funds** (e.g., CCXT, Alpaca, Jupiter). This section explains our guardrails and your responsibilities.

### Curated Registry

MCP servers can **only** be installed from a curated, hardcoded registry shipped with each release. You cannot add arbitrary servers. Every entry in the registry is reviewed before inclusion.

Each server has a **trust tier**:

| Tier | Meaning |
|------|---------|
| **Verified** | Audited by NickAI maintainers, known-safe |
| **Community** | Popular and open-source, but unaudited |

Trust tiers are displayed during `/mcp add` and in `/mcp list` so you can make an informed choice.

### Capability Labels

Each server declares its capabilities:

| Capability | What it means |
|------------|---------------|
| `read-data` | Market data, prices, analytics (read-only) |
| `trade` | Can place orders and execute trades |
| `on-chain` | Can sign and submit blockchain transactions |
| `analytics` | Aggregated metrics and analysis |

Servers with `trade` or `on-chain` capabilities trigger **additional guardrails** — see below.

### Trade-Capable Server Guardrails

When an MCP server has `trade` or `on-chain` capability, every tool call is wrapped with:

1. **User confirmation** — A confirmation prompt is shown before execution (same pattern as built-in `/buy`/`/sell`)
2. **Best-effort risk checks** — If the tool call includes `symbol`, `side`, `quantity`, and `price` fields, the system checks against your `/risk` limits before prompting
3. **Audit logging** — All tool calls are logged when `--debug` is enabled

### How MCP Servers Run

- Each server runs as a **local subprocess** communicating over stdio (not shell-executed — arguments passed as a list)
- Servers are started when NickAI launches and stopped when it exits
- Servers run with the same OS permissions as your user account — they are **not sandboxed**
- All tools a server exposes are automatically registered and available to the AI
- **Health checks** run every 5 minutes — dead connections are detected and automatically reconnected

### MCP Credential Handling

- When you `/mcp add` a server, required API keys are read from your environment variables and saved to `~/.nickai/mcp.json`
- Keys are passed to the server subprocess as environment variables at startup
- Keys are stored in plaintext in `~/.nickai/mcp.json` with `0600` permissions
- The `~/.nickai/` directory is created with `0700` permissions (owner-only access)

### Guardrails Against Malicious or Misbehaving Servers

| Guardrail | Description |
|-----------|-------------|
| **Curated registry** | Only pre-reviewed servers can be installed — no arbitrary URLs or packages |
| **Trust tiers** | Verified vs. community labeling helps you assess risk |
| **Capability labels** | Clearly indicates which servers can trade or sign transactions |
| **Trade confirmation** | Servers with `trade`/`on-chain` capability require user approval on every call |
| **Risk limit checks** | Best-effort risk checks applied to MCP trade tools (same limits as built-in trades) |
| **Required env vars** | Installation blocks until you explicitly provide API keys |
| **No auto-install** | Servers are never installed without your explicit `/mcp add` command |
| **Startup isolation** | Each server is a separate subprocess — a crash in one does not affect others |
| **Connection timeout** | Servers that fail to initialize within 30 seconds are skipped |
| **Health monitoring** | Dead connections detected every 5 minutes and automatically reconnected |
| **Failure tolerance** | A failing server does not block the rest of NickAI from starting |

### Known Limitations

Be aware of the following when using MCP integrations:

- **No tool-level filtering.** When you add a server, all of its tools are registered. You cannot selectively enable or disable individual tools.
- **No OS-level sandboxing.** MCP servers run with your user permissions and can access files, network, and environment variables.
- **Capabilities are advisory.** Trust tiers and capability labels are metadata for your decision-making — they are not enforced at runtime beyond the confirmation prompt.
- **Risk checks are best-effort.** MCP tool inputs vary by server — risk checks only work when standard field names (`symbol`, `side`, `quantity`, `price`) are used.

### Recommendations

1. **Start with read-only servers.** Install `defillama`, `tradingview`, or `brave-search` first — these only have `read-data` capability.
2. **Use paper/testnet keys first.** When connecting exchange servers (CCXT, Alpaca, Binance), use testnet API keys before switching to live keys.
3. **Review AI tool calls.** Pay attention when the AI proposes calling an MCP tool — especially one with `trade` capability.
4. **Limit API key permissions.** Most exchanges let you create API keys with restricted permissions (e.g., read-only, trade-only, no withdrawals). Use the minimum permissions needed.
5. **Keep MCP config secure.** The `~/.nickai/` directory is created with `0700` permissions by default. Verify with `ls -la ~/.nickai/`.

## Audit

Last audited: 2026-03-03

| Check | Tool | Result |
|-------|------|--------|
| Known CVEs in dependencies | `govulncheck` | 0 vulnerabilities |
| Static analysis | `go vet` | Clean |
| Race condition detection | `go test -race` | Clean |
| Module integrity | `go mod verify` | All checksums verified |
| Hardcoded secrets | Manual review | None found |
| Supply chain | Manual review | Core deps all trusted |
| File permissions | Manual review | All stores 0600, directories 0700 |
| Concurrency safety | Manual review | Per-store mutex on all load-modify-save |
| Input validation | Manual review | URL encoding, symbol normalization |
