# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
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

### Credentials

- API keys are stored locally at `~/.nickai/config.json` and `~/.nickai/credentials.json` with `0600` file permissions (owner-read/write only)
- Keys are never logged, committed, or transmitted except to their intended API endpoints over HTTPS
- Keys are masked in all display output

### Network

- All API communication uses HTTPS (TLS)
- PaperNick API: `https://paper.getnick.ai/api/v1`
- Anthropic API: `https://api.anthropic.com/v1/messages`
- MCP servers may make additional outbound connections to their respective APIs (exchange endpoints, RPC nodes, etc.)

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

Servers with `trade` or `on-chain` capabilities can move real money when connected to a live exchange or wallet. **Only install these if you understand the risk.**

### How MCP Servers Run

- Each server runs as a **local subprocess** communicating over stdio
- Servers are started when NickAI launches and stopped when it exits
- Servers run with the same OS permissions as your user account — they are **not sandboxed**
- All tools a server exposes are automatically registered and available to the AI

### MCP Credential Handling

- When you `/mcp add` a server, required API keys are read from your environment variables and saved to `~/.nickai/mcp.json`
- Keys are passed to the server subprocess as environment variables at startup
- Keys are stored in plaintext — protect `~/.nickai/mcp.json` with appropriate file permissions

### Guardrails Against Malicious or Misbehaving Servers

| Guardrail | Description |
|-----------|-------------|
| **Curated registry** | Only pre-reviewed servers can be installed — no arbitrary URLs or packages |
| **Trust tiers** | Verified vs. community labeling helps you assess risk |
| **Capability labels** | Clearly indicates which servers can trade or sign transactions |
| **Required env vars** | Installation blocks until you explicitly provide API keys |
| **No auto-install** | Servers are never installed without your explicit `/mcp add` command |
| **Startup isolation** | Each server is a separate subprocess — a crash in one does not affect others |
| **Connection timeout** | Servers that fail to initialize within 15 seconds are skipped |
| **Failure tolerance** | A failing server does not block the rest of NickAI from starting |

### Known Limitations

Be aware of the following when using MCP integrations:

- **No confirmation on MCP trades.** Built-in trades require confirmation; MCP server trades (e.g., via CCXT or Alpaca) execute when the AI calls the tool. Review the AI's plan before approving tool use.
- **Risk limits apply to built-in trades only.** The `/risk` guardrails (max order size, daily loss cap, position limits) do not currently apply to MCP-initiated trades.
- **No tool-level filtering.** When you add a server, all of its tools are registered. You cannot selectively enable or disable individual tools.
- **No OS-level sandboxing.** MCP servers run with your user permissions and can access files, network, and environment variables.
- **Capabilities are advisory.** Trust tiers and capability labels are metadata for your decision-making — they are not enforced at runtime.

### Recommendations

1. **Start with read-only servers.** Install `defillama`, `tradingview`, or `brave-search` first — these only have `read-data` capability.
2. **Use paper/testnet keys first.** When connecting exchange servers (CCXT, Alpaca, Binance), use testnet API keys before switching to live keys.
3. **Review AI tool calls.** Pay attention when the AI proposes calling an MCP tool — especially one with `trade` capability.
4. **Limit API key permissions.** Most exchanges let you create API keys with restricted permissions (e.g., read-only, trade-only, no withdrawals). Use the minimum permissions needed.
5. **Keep MCP config secure.** Run `chmod 600 ~/.nickai/mcp.json` to restrict file access to your user only.

## Audit

Last audited: 2026-02-28

| Check | Tool | Result |
|-------|------|--------|
| Known CVEs in dependencies | `govulncheck` | 0 vulnerabilities |
| Static analysis | `go vet` | Clean |
| Module integrity | `go mod verify` | All checksums verified |
| Hardcoded secrets | Manual review | None found |
| Supply chain | Manual review | Core deps all trusted |
