# VHS Demo — https://github.com/charmbracelet/vhs
# Run: vhs demo.tape
#
# Prerequisites:
#   - brew install vhs ffmpeg ttyd
#   - /config init (account provisioned)
#   - /config set anthropic_key sk-ant-...
#   - go build succeeds

Output demo.gif

Set Shell "zsh"
Set FontSize 14
Set Width 1200
Set Height 750
Set Padding 16
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms
Set PlaybackSpeed 1

# ─────────────────────────────────────────────
# ACT 0 — Build (hidden)
# ─────────────────────────────────────────────
Hide
Type "go build -o /tmp/nickai-demo . && clear"
Enter
Sleep 3s
Show

# ─────────────────────────────────────────────
# ACT 1 — Launch & Boot Sequence
# ─────────────────────────────────────────────
# The boot animation is ~4s: ASCII logo reveal → tagline typeout → status checks
Type "/tmp/nickai-demo"
Enter
Sleep 6s

# ─────────────────────────────────────────────
# ACT 2 — One-Command Setup
# ─────────────────────────────────────────────
Type "/config init"
Enter
Sleep 3s

# ─────────────────────────────────────────────
# ACT 3 — AI Conversation (streaming)
# ─────────────────────────────────────────────
# Natural question → streamed markdown response with tool use
Type "what's the price of BTC and ETH right now?"
Enter
Sleep 12s

# Follow-up with deeper analysis — AI calls analyze_market tool
Type "should I buy ETH? give me a quick analysis"
Enter
Sleep 14s

# ─────────────────────────────────────────────
# ACT 4 — Technical Analysis
# ─────────────────────────────────────────────
# Full indicator suite: RSI, MACD, Bollinger, SMA, Fear & Greed, sparkline
Type "/analyze BTC"
Enter
Sleep 4s

# Let the viewer absorb the card
Sleep 2s

# ─────────────────────────────────────────────
# ACT 5 — Execute a Trade
# ─────────────────────────────────────────────
Type "/buy ETH 0.5"
Enter
Sleep 2s

# Confirmation card is showing — press y to confirm
Type "y"
Sleep 3s

# Check what we own now
Type "/status"
Enter
Sleep 3s

# ─────────────────────────────────────────────
# ACT 6 — Portfolio Analytics
# ─────────────────────────────────────────────
# Sharpe ratio, drawdown, win rate, allocation bar chart
Type "/analytics"
Enter
Sleep 4s

# P&L with progress bar
Type "/pnl"
Enter
Sleep 3s

# ─────────────────────────────────────────────
# ACT 7 — Risk Guardrails
# ─────────────────────────────────────────────
Type "/risk set max-order 5000"
Enter
Sleep 2s

Type "/risk set daily-loss 1000"
Enter
Sleep 2s

Type "/risk show"
Enter
Sleep 3s

# ─────────────────────────────────────────────
# ACT 8 — Automation & Triggers
# ─────────────────────────────────────────────
# Price alert
Type "/alert BTC > 100000"
Enter
Sleep 2s

# Conditional trigger
Type "/trigger add ETH < 3000 buy 1.0"
Enter
Sleep 2.5s

# Desktop notifications
Type "/notify set desktop on"
Enter
Sleep 2s

# Test notification
Type "/notify test"
Enter
Sleep 3s

# ─────────────────────────────────────────────
# ACT 9 — Charts & Market Data
# ─────────────────────────────────────────────
# Sparkline chart
Type "/chart BTC"
Enter
Sleep 3s

# Multi-price quotes
Type "/price BTC ETH SOL"
Enter
Sleep 3s

# Full market overview — 10 assets
Type "/market"
Enter
Sleep 4s

# ─────────────────────────────────────────────
# ACT 10 — Dashboard
# ─────────────────────────────────────────────
Type "/snapshot"
Enter
Sleep 4s

# ─────────────────────────────────────────────
# ACT 11 — Command Palette (Ctrl+K)
# ─────────────────────────────────────────────
Ctrl+K
Sleep 1.5s

# Fuzzy search
Type "anal"
Sleep 1s

# Select analyze
Enter
Sleep 500ms

# Complete the command
Type " ETH"
Enter
Sleep 4s

# ─────────────────────────────────────────────
# ACT 12 — Theme Picker (Ctrl+T)
# ─────────────────────────────────────────────
Ctrl+T
Sleep 1.5s

# Browse themes — move to cyberpunk
Down
Sleep 400ms
Down
Sleep 400ms

# Select cyberpunk
Enter
Sleep 2s

# Show the market in the new theme
Type "/market"
Enter
Sleep 3s

# Switch to matrix theme
Ctrl+T
Sleep 1.5s
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms

# Select matrix
Enter
Sleep 2s

# Show chart in matrix green
Type "/chart ETH"
Enter
Sleep 3s

# Switch back to default theme
Ctrl+T
Sleep 1.5s
Enter
Sleep 1.5s

# ─────────────────────────────────────────────
# ACT 13 — Vim Mode & Overlays
# ─────────────────────────────────────────────
# Enter normal mode
Escape
Sleep 500ms

# Scroll up through history
Type "kkkkkk"
Sleep 1.5s

# Jump to bottom
Type "G"
Sleep 500ms

# Open help overlay
Type "?"
Sleep 3s

# Close help
Escape
Sleep 500ms

# Model selector
Ctrl+O
Sleep 2s

# Close it
Escape
Sleep 1s

# ─────────────────────────────────────────────
# ACT 14 — AI Creates an Automation
# ─────────────────────────────────────────────
# Back to insert mode
Type "i"
Sleep 300ms

Type "buy $50 of BTC every day"
Enter
Sleep 12s

# Show the created rule
Type "/auto list"
Enter
Sleep 3s

# ─────────────────────────────────────────────
# ACT 15 — Manual Page
# ─────────────────────────────────────────────
Type "/man analyze"
Enter
Sleep 3s

# ─────────────────────────────────────────────
# ACT 16 — Trade Journal
# ─────────────────────────────────────────────
Type "/history"
Enter
Sleep 3s

# ─────────────────────────────────────────────
# ACT 17 — Exit
# ─────────────────────────────────────────────
Escape
Sleep 300ms
Type ":q"
Enter
Sleep 1.5s
