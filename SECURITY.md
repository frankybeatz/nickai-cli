# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.3.x   | Yes       |
| < 0.3   | No        |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly.

**Do not open a public issue.** Instead, email **security@getnick.ai** with:

- Description of the vulnerability
- Steps to reproduce
- Impact assessment
- Suggested fix (if any)

You should receive an acknowledgment within 48 hours. We aim to release a patch within 7 days of confirmation.

## Security Model

NickAI CLI is a **local-only terminal application**. It does not run a server, accept inbound connections, or expose any network surface.

### Credentials

- API keys are stored locally at `~/.nickai/config.json` and `~/.nickai/credentials.json` with `0600` file permissions (owner-read/write only)
- Keys are never logged, committed, or transmitted except to their intended API endpoints over HTTPS
- Keys are masked in all display output

### Network

- All API communication uses HTTPS (TLS)
- PaperNick API: `https://paper.getnick.ai/api/v1`
- Anthropic API: `https://api.anthropic.com/v1/messages`
- No other outbound connections are made

### Dependencies

- Only 3 direct dependencies, all from [Charmbracelet](https://github.com/charmbracelet) (widely trusted, open-source TUI libraries)
- Dependencies are pinned in `go.sum` with cryptographic checksums
- Audited with `govulncheck` — zero known vulnerabilities

### What This App Does NOT Do

- Does not access real funds (paper trading only)
- Does not run background processes or daemons
- Does not collect telemetry or analytics
- Does not auto-update or download code at runtime

## Audit

Last audited: 2026-02-27

| Check | Tool | Result |
|-------|------|--------|
| Known CVEs in dependencies | `govulncheck` | 0 vulnerabilities |
| Static analysis | `go vet` | Clean |
| Module integrity | `go mod verify` | All checksums verified |
| Hardcoded secrets | Manual review | None found |
| Supply chain | Manual review | 3 direct deps, all trusted |
