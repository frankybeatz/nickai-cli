# VHS Demo — https://github.com/charmbracelet/vhs
# Run: vhs demo.tape

Output demo.gif

Set Shell "zsh"
Set FontSize 14
Set Width 1200
Set Height 700
Set Padding 20
Set Theme "Catppuccin Mocha"

# Build first so startup is fast.
Hide
Type "go build -o /tmp/nickai-demo . && clear"
Enter
Sleep 2s
Show

# Launch.
Type "/tmp/nickai-demo"
Enter
Sleep 8s

# Feature 1: Auto-provisioning — frictionless onboarding.
Type "/config init"
Enter
Sleep 3s

# Feature 2: Streaming AI — tokens appear word-by-word.
Type "what is bitcoin? keep it to 3 sentences"
Enter
Sleep 10s

# Feature 3: Streaming with tool use — auto price lookup.
Type "check the price of BTC and ETH and tell me which is a better buy"
Enter
Sleep 12s

# Show markdown rendering in a structured response.
Type "give me a quick bull vs bear case for SOL as a markdown list"
Enter
Sleep 12s

# Show command palette (Ctrl+K).
Ctrl+K
Sleep 2s

# Filter to "price".
Type "price"
Sleep 1.5s

# Select it.
Enter
Sleep 500ms

# Add arguments and run.
Type " BTC ETH SOL"
Enter
Sleep 2.5s

# Show a sparkline chart.
Type "/chart BTC"
Enter
Sleep 2.5s

# Show snapshot dashboard.
Type "/snapshot"
Enter
Sleep 3s

# Open theme picker (Ctrl+T).
Ctrl+T
Sleep 1.5s

# Navigate to cyberpunk.
Down
Sleep 300ms
Down
Sleep 300ms

# Select it.
Enter
Sleep 2s

# Show market overview in new theme.
Type "/market"
Enter
Sleep 3s

# Switch to normal mode and open help overlay.
Escape
Sleep 500ms
Type "?"
Sleep 3s

# Close help, back to insert.
Type "i"
Sleep 500ms

# Model selector (Ctrl+O).
Ctrl+O
Sleep 2s

# Close it.
Escape
Sleep 1s

# Set an alert.
Type "/alert BTC > 100000"
Enter
Sleep 2s

# Scroll up in normal mode.
Escape
Sleep 300ms
Type "kkkk"
Sleep 1s

# Back to bottom.
Type "G"
Sleep 500ms

# Quit.
Type ":q"
Enter
Sleep 1s
