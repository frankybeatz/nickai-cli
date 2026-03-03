# NickAI CLI — Elite Product Audit

**Date:** 2026-03-03
**Agents:** 5 (AI/ML, UI/UX, Backend, Logic/Testing, Architecture)
**Codebase:** 20,272 lines of Go across 25 internal packages
**Score:** ~62% Elite

---

## Domain Scores

| Domain | Score | Findings |
|--------|-------|----------|
| AI/ML | 58% | 25 |
| Backend | 55% | 35 |
| Logic/Testing | 55% | 31 |
| UI/UX | 68% | 30 |
| Architecture | 60% | 25 |
| **Total** | **~62%** | **146** |

---

## Phase 1: Safety & Correctness

### 1.1 Non-Atomic File Writes (CRITICAL)
- **All 12 stores** use `os.WriteFile()` directly
- Crash mid-write = corrupted JSON = total data loss
- **Fix:** Write-to-temp-then-rename pattern:
```go
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, data, perm); err != nil {
        return err
    }
    return os.Rename(tmp, path)
}
```
- **Files:** config/config.go:76, credential/store.go:82, trigger/store.go:70, automation/store.go:126, strategy/store.go:71, alert/store.go:60, journal/store.go:64, memory/store.go:80, risk/store.go:66, notify/dispatch.go:62, workflow/store.go:104, mcp/config.go:69

### 1.2 Tool Result Truncation (CRITICAL)
- Tool results from `ExecuteTool()` appended to history with no size limit
- MCP tool can return megabytes, blows context window
- **Fix:** `const maxToolResultBytes = 16384` — truncate after ExecuteTool returns
- **Files:** agent.go:486-491, agent.go:758-765

### 1.3 Conversation History Pruning (CRITICAL)
- `a.history` grows unbounded, only reset on model switch
- After ~20-30 multi-tool exchanges, hits context limit → hard 400 error
- **Fix:** Token-aware sliding window. Before every API call, estimate tokens. If > 80% of context window, drop oldest user/assistant pairs. Always preserve most recent N turns.
- **File:** agent.go (new method `pruneHistory()`)

### 1.4 `crypto/rand` for ID Generation (HIGH)
- `randomToolID` in builtin.go uses `time.Now().UnixNano()` — produces collisions
- `randomID` in chat.go uses `math/rand` — not cryptographically secure
- **Fix:** Replace both with `crypto/rand`:
```go
import "crypto/rand"
func randomID(n int) string {
    const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
    b := make([]byte, n)
    rand.Read(b)
    for i := range b { b[i] = chars[b[i]%byte(len(chars))] }
    return string(b)
}
```
- **Files:** tools/builtin.go:1096-1104, ui/chat.go:3791-3798

### 1.5 Fix 3 Broken Backtest Presets (CRITICAL)
- bollinger-bounce and dip-buyer: `price < 0` never true for crypto
- golden-cross: `sma20 crosses_above 0` — SMA is always > 0
- **Fix:** Implement cross-indicator comparison in engine (e.g., `IndicatorRef` field) or replace with working static thresholds
- **Files:** backtest/presets.go:47,58,101 — engine.go:283 (evalCondition)

### 1.6 429 Rate Limit Handling (CRITICAL)
- Retry logic retries 5xx but treats 429 as non-retryable
- **Fix:** Add 429 handler before generic 4xx block, parse Retry-After header
- **File:** agent.go:670-683

### 1.7 Tool Execution Timeout (CRITICAL)
- `ExecuteTool` uses `context.Background()` — no deadline, stuck tool = frozen app
- **Fix:** `context.WithTimeout(context.Background(), 30*time.Second)`
- **File:** tools/registry.go:118

### 1.8 Fix `automation.Add` Error Swallowing (HIGH)
- `rules, _ := Load()` — if Load fails, existing rules overwritten with empty list
- **Fix:** Propagate error: `if err != nil { return err }`
- **File:** automation/store.go:131

### 1.9 Per-Store Mutex (CRITICAL)
- Zero stores have concurrency protection
- Load-Modify-Save pattern = TOCTOU race condition
- Triggers can double-fire, journal entries can be lost
- **Fix:** Add `sync.Mutex` per store guarding entire Load-Modify-Save cycle
- **Files:** All 12 stores

### 1.10 Fix go.mod Version (HIGH)
- `go 1.25.0` doesn't exist — build fails on standard Go
- **Fix:** Set to actual Go version (e.g., `go 1.23.0`)
- **File:** go.mod:3

---

## Phase 2: Reliability & API

### 2.1 Streaming Retry Logic (HIGH)
- `callAnthropicStream` has zero retry logic vs 3 retries for non-streaming
- Streaming is the primary code path from the TUI
- **Fix:** Wrap in retry loop matching `callAnthropic`
- **File:** agent.go:796-914

### 2.2 Remove TLS 1.2 Cap (HIGH)
- `TLSClientConfig: &tls.Config{MaxVersion: tls.VersionTLS12}` prevents TLS 1.3
- No legitimate reason — TLS 1.3 is faster and more secure
- **Fix:** Change to `MinVersion: tls.VersionTLS12` (floor not ceiling) or remove entirely
- **Files:** agent.go:295, agent.go:826

### 2.3 Non-Anthropic Models Warn About No Tool Use (MEDIUM)
- OpenRouter/MiniMax can't call tools but system prompt describes 12+ tools
- AI hallucinates prices, can't execute trades
- **Fix:** Different system prompt for non-tool-capable models, UI warning
- **Files:** agent.go:504,575

### 2.4 Thread `context.Context` for Cancellation (HIGH)
- None of the HTTP requests use context — Ctrl+C can't cancel in-flight calls
- Related to needing `os.Exit(0)` for force-quit
- **Fix:** `Chat(ctx, msg)`, `ChatStream(ctx, msg, ch)`, use `http.NewRequestWithContext`
- **Files:** agent.go (all API methods)

### 2.5 Set Anthropic Temperature (MEDIUM)
- Anthropic API defaults to temperature=1.0 when not specified
- OpenRouter uses 0.3 — wildly different randomness per provider
- For trading assistant, high temperature = inconsistent advice
- **Fix:** Add `Temperature float64` to apiRequest, set 0.3
- **Files:** agent.go:183-189, agent.go:534-540

### 2.6 MCP Health Checks + Reconnection (MEDIUM)
- `ConnectAll` connects once at startup, no reconnection on failure
- Dead MCP connection = broken tools for rest of session
- **Fix:** Add `Reconnect(name)`, periodic health check (ListTools every 5 min)
- **File:** mcp/client.go:44-57

### 2.7 Add CI Workflow (HIGH)
- Only release.yml exists — no PR checks, no go vet, no go test
- **Fix:** Add `.github/workflows/ci.yml` with go vet + go test + go build
- **File:** .github/workflows/ci.yml (new)

### 2.8 Fix Makefile Test Target (MEDIUM)
- `make test` runs go vet and go build but NOT go test
- 118 test functions never run by build system
- **Fix:** Replace `go build ./...` with `go test ./...`
- **File:** Makefile:18-21

### 2.9 Consistent File Permissions (MEDIUM)
- 8 of 12 stores use 0644 (world-readable), rest use 0600
- Journal, memory, automations contain sensitive trading data
- **Fix:** Standardize all to 0600, directory to 0700
- **Files:** All 12 stores

### 2.10 Webhook Error Propagation (HIGH)
- `_, _ = client.Post(url, ...)` — webhook failures silently swallowed
- User thinks alerts work but notifications vanish
- **Fix:** Return error, show status flash on failure
- **File:** notify/dispatch.go:118-122

---

## Phase 3: UX Polish

### 3.1 Fix OSC Escape Code Regex (CRITICAL)
- `oscLeakRe` too broad — matches valid hex input like `/DEAD/BEEF`
- Cleanup runs AFTER textInput.Update — garbled text flickers for one frame
- **Fix:** Require ESC or `]` prefix in regex. Intercept BEFORE textInput.Update
- **File:** chat.go:62, chat.go:1674-1683

### 3.2 Viewport Auto-Scroll Only at Bottom (HIGH)
- `updateViewport()` unconditionally calls `GotoBottom()` every 80ms during streaming
- User can't scroll up to read history while AI responds
- **Fix:** Track if user scrolled up, only auto-scroll if already at bottom:
```go
atBottom := m.viewport.AtBottom()
m.viewport.SetContent(content)
if atBottom { m.viewport.GotoBottom() }
```
- **File:** chat.go:5075

### 3.3 Search `n`/`N` Navigation + Highlighting (HIGH)
- Search finds first match only, no next/prev, no highlight, no count
- Fundamental vim expectation violated
- **Fix:** Store all match indices, add n/N bindings to Normal mode, show "[3/17]"
- **File:** chat.go:1991-2003, chat.go:1723-1793

### 3.4 Fix Vim `a`/`I`/`o` Semantics (MEDIUM)
- `a` doesn't advance cursor (should append AFTER current position)
- `I` clears input (should CursorStart)
- `o` clears input (should just enter insert mode)
- **Fix:** `a` → `SetCursor(Position+1)`, `I` → `CursorStart()`, `o` → just Focus()
- **File:** chat.go:1749-1768

### 3.5 Theme Change Propagates to TextInput (HIGH)
- TextInput styles set once at construction, never updated on theme switch
- Input bar uses old theme colors after switching
- **Fix:** Update PromptStyle/TextStyle/PlaceholderStyle after ApplyTheme()
- **File:** chat.go (theme handler), styles.go

### 3.6 Scrollable Model/Theme Dialogs (MEDIUM)
- Model dialog has 9+ entries, theme has 10 — no scroll on short terminals
- Below ~20 rows, entries are clipped and unreachable
- **Fix:** Add ScrollOffset support (like palette dialog already has)
- **File:** overlay.go:238-315

### 3.7 Context-Aware Placeholder Text (LOW)
- Always "Ask NickAI anything or type / for commands..."
- Could show "Run /config init to get started..." when unconfigured
- **Fix:** Dynamic placeholder based on state
- **File:** chat.go:335

### 3.8 `padRight` Use Visual Width (HIGH)
- Uses `len(s)` which counts bytes — breaks with ANSI codes and Unicode
- Causes misaligned columns throughout UI
- **Fix:** `lipgloss.Width(s)` instead of `len(s)`
- **File:** cards.go:1452-1457

### 3.9 Enable Mouse Wheel Scrolling (MEDIUM)
- `MouseWheelEnabled = false` — trackpad users can't scroll
- **Fix:** `m.viewport.MouseWheelEnabled = true`
- **File:** chat.go:1143

### 3.10 Remove Dead `title` Param from overlayFrame (LOW)
- Parameter accepted, rendered, then discarded with `_ = titleRendered`
- All callers pass empty string
- **Fix:** Remove parameter
- **File:** overlay.go:36-59

---

## Phase 4: Architecture

### 4.1 Extract chat.go (5,473 lines → 10+ files)
- 33 handler methods, 50+ field struct, every internal package imported
- **Target:** model.go, update.go, view.go, vim_*.go, handlers/*.go, messages.go
- **File:** chat.go

### 4.2 Single Command Registry
- Commands defined in 5 places: knownCommands, allCommands, paletteCommands, helpDialog, manpages
- **Fix:** Single `CommandDef` struct, all consumers derive from one slice
- **Files:** chat.go:64, router.go:81, overlay.go:396, overlay.go:121, manpages.go

### 4.3 Extract Guard Functions
- "No API key" message copy-pasted 17 times
- Symbol normalization triple-TrimSuffix pasted 6 times
- **Fix:** `requireAgent()` method, `BaseSymbol()` function
- **File:** chat.go (17+ locations)

### 4.4 Test Coverage: analytics + tools
- Zero tests for Sharpe ratio, drawdown, win rate, profit factor
- Zero tests for tool registry, ExecuteTool, all tool executors
- **Fix:** Create metrics_test.go, registry_test.go
- **Files:** analytics/, tools/

### 4.5 Fix Sharpe Ratio Annualization
- Backtest hardcodes `sqrt(252)` regardless of candle interval (1h, 4h, 1d)
- Wrong by up to 5x for hourly candles
- Analytics treats per-trade returns as daily — statistically meaningless
- **Fix:** Pass candles-per-year factor; use consistent sample variance
- **Files:** engine.go:448, metrics.go:104-121

### 4.6 Backtest Slippage + Commission
- Assumes perfect fills at close price with zero fees
- **Fix:** Add SlippageBps/CommissionBps to Strategy struct, apply at entry/exit
- **File:** engine.go:234-246

### 4.7 Remove Binaries from Git
- `cli` (10.7MB) and `nickai` (22MB) checked into repo
- **Fix:** `git rm --cached cli nickai`, add to .gitignore
- **Files:** cli, nickai, .gitignore

### 4.8 Structured Logging
- Zero logging infrastructure, no debug mode
- **Fix:** Add `log/slog` with file handler, enabled by `--debug` or `NICKAI_DEBUG=1`
- **File:** New internal/logging/ or flag in main.go

### 4.9 Credential Encryption
- Exchange API keys stored as plaintext JSON
- **Fix:** OS keyring integration (`github.com/zalando/go-keyring`)
- **File:** credential/store.go

### 4.10 Canonical NormalizeSymbol
- Two implementations: length-based (broken) and suffix-based (correct)
- `api.NormalizeSymbol` uses `len > 5` heuristic — fails for MATIC, SHIB, etc.
- **Fix:** Delete length-based, use suffix-based everywhere
- **Files:** api/papernick.go:216, market/binance.go:25

---

## Additional Findings by Domain

### AI/ML (remaining)
- Stale Anthropic API version (2023-06-01) — missing prompt caching, 1.5 years behind
- Dynamic prompt suffixes not sanitized — prompt injection via save_memory
- System prompt massive and unstructured — XML delimiters would help
- New streaming HTTP client created per call — should reuse
- Consensus model list contains speculative future models (gpt-5.2, grok-4.1)
- Consensus tie-breaking is non-deterministic (Go map iteration order)
- `sanitizeHistory` misses type assertion edge cases
- `io.ReadAll` without size limit on response bodies
- Retry uses fixed 1s delay instead of exponential backoff with jitter
- `strings.Title` deprecated in consensus.go
- Consensus has no per-model timeout — one slow model blocks all
- `maxTokens = 4096` is low for complex tool chains

### Backend (remaining)
- `credential.Get` returns copy not pointer (Go loop variable pitfall)
- MCP config stores env var VALUES in plaintext JSON
- Memory `ForPrompt` truncation can cut mid-UTF8 character
- Workflow `Run()` never persists state to disk
- Empty automation conditions = always fires (vacuous truth)
- Inconsistent Sharpe variance (population vs sample) between backtest and analytics

### Logic (remaining)
- Equity curve ignores unrealized P&L during open positions
- MACD calculation is O(n^2), backtest warmup makes engine O(n^3)
- `generateSyntheticHistory` uses biased non-random walk
- Place order confirmation can deadlock with concurrent tool calls
- FetchFearGreed silently swallows parse errors (defaults to 50)
- RSI default 50 during warmup can create phantom crossings
- Risk check symbol matching inconsistent (BTCUSDT vs BTC)
- Unknown indicator/operator silently returns false — no error
- Period-end forced trade close not distinguished in metrics
- TestRouteEmpty assertion is a no-op (tests 0 != 0)
- FuzzySort can suggest aliases instead of canonical commands

### UI/UX (remaining)
- Welcome screen tagline re-rolls on every ticker refresh (flickers)
- Tab completion has two incompatible systems (legacy + suggestions)
- Ctrl+C writes raw escape sequences, bypasses Bubbletea cleanup
- `compositeOverlay` transparent lines leak through dialogs
- Command mode limited command set, no `:N` line jump
- Help dialog 4-column layout overflows on 80-col terminals
- Dashboard mode has no keyboard shortcut or toggle indicator
- Suggestions box can overwrite top bar on short terminals
- Boot animation MCP check unreliable (always shows missing)
- `knownSymbols` only 9 hardcoded symbols for tab completion
- Streaming content not word-wrapped until stream completes
- No incremental search preview while typing `/pattern`

### Architecture (remaining)
- `renderResult()` duplicates routing logic from main Update switch
- `Model` struct 50+ fields — should be grouped into sub-structs
- `updateConfirmMode` 371 lines with 5 nested confirmation branches
- Version defined in 4 places (main.go, styles.go, Makefile, mcp/client.go)
- `os.Exit(0)` on Ctrl+C skips session save/cleanup
- Config layering inverted (config file checked before env vars)
- No package-level documentation on any of 25 packages
- `streamToAI` boilerplate duplicated across ~12 handlers
- `cli/commands.go` duplicates agent setup 3 times
- Welcome.go uses variadic `memCount` as positional args
