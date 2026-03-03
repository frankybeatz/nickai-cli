# Deployment Guide — nickai-cli

## Quick Start

### Binary Install

Download the latest release from [GitHub Releases](https://github.com/frankybeatz/nickai-cli/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/frankybeatz/nickai-cli/releases/latest/download/nickai-darwin-arm64.tar.gz | tar xz
sudo mv nickai /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/frankybeatz/nickai-cli/releases/latest/download/nickai-darwin-amd64.tar.gz | tar xz
sudo mv nickai /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/frankybeatz/nickai-cli/releases/latest/download/nickai-linux-amd64.tar.gz | tar xz
sudo mv nickai /usr/local/bin/
```

Or via Homebrew:

```bash
brew tap frankybeatz/nickai
brew install nickai
```

### From Source

```bash
git clone https://github.com/frankybeatz/nickai-cli.git
cd nickai-cli
make build        # builds nickai binary
make node         # builds nickai-node binary
make install      # installs to GOPATH/bin
```

## Docker

### Build and Run

```bash
# Build image
make docker

# Run with docker compose
make docker-run

# Or manually
docker build -t nickai .
docker run -d \
  --name nickai-node \
  -p 127.0.0.1:9400:9400 \
  -v nickai-data:/home/nickai/.nickai \
  nickai
```

### Docker Compose

```bash
docker compose up -d      # start
docker compose logs -f    # follow logs
docker compose down       # stop
```

The compose file mounts a named volume at `/home/nickai/.nickai` for persistent configuration and data.

## Configuration

All configuration lives in `~/.nickai/`:

| File | Purpose |
|------|---------|
| `config.json` | API keys, model selection, theme |
| `mcp.json` | MCP server configurations |
| `credentials.json` | Exchange API credentials |
| `risk.json` | Risk management limits |
| `memory.json` | AI memory entries |
| `telemetry.json` | Telemetry configuration |
| `telemetry_events.json` | Telemetry event log |

### Required Keys

Set up your PaperNick API key on first launch (the TUI will guide you), or:

```bash
nickai config set api_key YOUR_KEY
```

For AI features:

```bash
nickai config set anthropic_key YOUR_KEY    # Claude models
nickai config set openrouter_key YOUR_KEY   # OpenRouter models
```

## Nick Node Architecture

Nick Node (`nickai-node`) is a separate always-on process for:

- Persistent strategy execution (TWAP, automations)
- Live price streaming via gRPC
- Background backtest job offloading
- Alert dispatch

### Running Nick Node

```bash
# Binary
nickai-node --addr 127.0.0.1:9400

# Remote bind requires auth token
NICKAI_NODE_TOKEN=change-me nickai-node --addr 0.0.0.0:9400

# Docker
docker compose up -d

# Connect from TUI
/node connect localhost:9400
/node status
```

### gRPC API

Nick Node exposes 10 RPCs on port 9400:

- `Ping` — health check
- `DeployStrategy` / `ListStrategies` / `StopStrategy`
- `StreamPrices` — server-streaming live prices
- `SubmitBacktest` / `GetBacktestResult`
- `CreateAlert` / `ListAlerts`
- `GetStatus`

By default, node binds to loopback only. If you bind to a non-loopback address, you must set `--token` or `NICKAI_NODE_TOKEN`.

## CI/CD Pipeline

### GitHub Actions

The project uses GoReleaser for automated releases. On tag push:

1. `go mod tidy` + `go vet ./...`
2. Builds `nickai` and `nickai-node` for darwin/linux (amd64/arm64) + windows/amd64
3. Creates GitHub Release with archives and checksums
4. Updates Homebrew tap

### Dependabot

Automated dependency updates via `.github/dependabot.yml`:
- Go modules: weekly on Monday
- GitHub Actions: weekly on Monday

## Development

```bash
make build    # build nickai binary
make node     # build nickai-node binary
make test     # run go vet + go test
make run      # run TUI locally
make demo     # generate demo GIF (requires vhs)
make release  # build all platform binaries
make clean    # remove build artifacts
```

### Running Tests

```bash
go test ./...                    # all tests
go test ./internal/backtest/...  # specific package
go test -race ./...              # with race detector
```
