package guidance

import "fmt"

// Stage represents where the user is in their NickAI journey.
type Stage string

const (
	StageFresh      Stage = "fresh"      // No API key
	StageConfigured Stage = "configured" // Has API key, no AI key
	StageAIReady    Stage = "ai_ready"   // Has AI key, no MCP
	StageEquipped   Stage = "equipped"   // Has MCP, 0 trades
	StageTrading    Stage = "trading"    // Has trades, no analysis
	StageAnalyzing  Stage = "analyzing"  // Used analysis, no backtest
	StageAdvanced   Stage = "advanced"   // Used most features
)

// ActionCard is a suggested next action shown to the user.
type ActionCard struct {
	Icon    string
	Title   string
	Desc    string
	Command string
}

// ChecklistItem is a guided onboarding task shown in the welcome experience.
type ChecklistItem struct {
	Label   string
	Command string
	Done    bool
}

// StageContext holds the state needed to determine the user's journey stage.
type StageContext struct {
	HasAPIKey      bool
	HasAIKey       bool
	MCPCount       int
	TradeCount     int
	HasAnalyzed    bool
	HasBacktested  bool
	MemoryCount    int
	PortfolioValue float64
	CashBalance    float64
	TopPositions   []string // symbol names
}

// DetectStage returns the user's current journey stage.
func DetectStage(ctx StageContext) Stage {
	if !ctx.HasAPIKey {
		return StageFresh
	}
	if !ctx.HasAIKey {
		return StageConfigured
	}
	if ctx.MCPCount == 0 {
		return StageAIReady
	}
	if ctx.TradeCount == 0 {
		return StageEquipped
	}
	if !ctx.HasAnalyzed {
		return StageTrading
	}
	if !ctx.HasBacktested {
		return StageAnalyzing
	}
	return StageAdvanced
}

// ActionsForStage returns 2-3 action cards for the given stage.
func ActionsForStage(stage Stage, ctx StageContext) []ActionCard {
	switch stage {
	case StageFresh:
		return []ActionCard{
			{Icon: "⚡", Title: "Create your account", Desc: "$100K paper money. Zero risk. Full degen.", Command: "/config init"},
		}
	case StageConfigured:
		return []ActionCard{
			{Icon: "🧠", Title: "Unlock Nick", Desc: "Add your Anthropic key and let's cook", Command: "/config set anthropic_key <key>"},
			{Icon: "🎭", Title: "Pick your vibe", Desc: "degen, quant, zen, hype, sensei, or degen-bets", Command: "/vibe"},
			{Icon: "💰", Title: "Check the vibe", Desc: "See what BTC is doing rn", Command: "/price BTC"},
		}
	case StageAIReady:
		return []ActionCard{
			{Icon: "🔌", Title: "Get the tools", Desc: "Free MCP servers = live data = alpha", Command: "/mcp quick"},
			{Icon: "💬", Title: "Talk to Nick", Desc: "\"what should I ape into?\"", Command: "what should I buy?"},
		}
	case StageEquipped:
		return []ActionCard{
			{Icon: "📊", Title: "Scout the market", Desc: "Nick will pull the data and give you the play", Command: "/price BTC ETH SOL"},
			{Icon: "💰", Title: "Send your first trade", Desc: "Paper money, full send, zero consequences", Command: "buy 0.1 BTC"},
			{Icon: "🔍", Title: "Run the TA", Desc: "RSI, MACD, Bollinger — the whole toolkit", Command: "/analyze BTC"},
		}
	case StageTrading:
		return []ActionCard{
			{Icon: "📈", Title: "Analyze your bags", Desc: "Are your positions based or cringe?", Command: "/analyze BTC"},
			{Icon: "🤝", Title: "Multi-AI consensus", Desc: "10 models vote. Majority rules. No emotion.", Command: "/consensus BTC"},
			{Icon: "📋", Title: "Check your P&L", Desc: "Are we up or are we learning?", Command: "/pnl"},
		}
	case StageAnalyzing:
		return []ActionCard{
			{Icon: "🧪", Title: "Backtest it", Desc: "Would this strategy have printed? Find out.", Command: "/backtest presets"},
			{Icon: "⚙️", Title: "Automate it", Desc: "Nick trades while you touch grass", Command: "/auto list"},
			{Icon: "🛡", Title: "Set risk limits", Desc: "Degen with discipline > rekt degen", Command: "/risk"},
		}
	case StageAdvanced:
		return advancedActions(ctx)
	default:
		return nil
	}
}

func advancedActions(ctx StageContext) []ActionCard {
	cards := []ActionCard{}

	if ctx.CashBalance > 0 && ctx.PortfolioValue > 0 {
		pct := (ctx.CashBalance / ctx.PortfolioValue) * 100
		if pct > 30 && pct <= 100 {
			cards = append(cards, ActionCard{
				Icon:    "💰",
				Title:   fmt.Sprintf("%.0f%% cash sitting idle", pct),
				Desc:    "Ask Nick to scan for entries",
				Command: "scan for good entries",
			})
		}
	}

	cards = append(cards, ActionCard{
		Icon:    "🎯",
		Title:   "Prediction markets",
		Desc:    "Find mispriced contracts on Polymarket",
		Command: "/polymarket scan",
	})

	if len(ctx.TopPositions) > 0 {
		sym := ctx.TopPositions[0]
		cards = append(cards, ActionCard{
			Icon:    "📊",
			Title:   "Deep dive: " + sym,
			Desc:    "Full technical + sentiment analysis",
			Command: "/analyze " + sym,
		})
	}

	if len(cards) > 3 {
		cards = cards[:3]
	}
	return cards
}

// NextStepAfterCommand returns contextual hints after a command completes.
// Uses full context for smarter, stage-aware suggestions.
func NextStepAfterCommand(cmd string, ctx StageContext) []string {
	switch cmd {
	// Market data
	case "price":
		if !ctx.HasAnalyzed {
			return []string{"/analyze <sym> — AI deep-dive"}
		}
		return []string{"/analyze <sym>", "/buy <sym>"}
	case "market", "watch":
		return []string{"/analyze <sym>", "/alert <sym> > <price>"}
	case "chart":
		return []string{"/analyze <sym>", "/backtest run rsi-reversal <sym>"}

	// Analysis
	case "analyze":
		if ctx.TradeCount == 0 {
			return []string{"/buy <sym> — make your first trade", "/consensus <sym> — multi-AI vote"}
		}
		if !ctx.HasBacktested {
			return []string{"/backtest run rsi-reversal <sym>", "/consensus <sym>"}
		}
		return []string{"/buy <sym>", "/backtest run rsi-reversal <sym>"}
	case "consensus":
		return []string{"/buy <sym> — act on it", "/analyze <sym> — go deeper"}
	case "analytics":
		return []string{"/pnl", "/risk set max-order <val>"}

	// Trading
	case "buy", "sell":
		if ctx.TradeCount <= 1 {
			return []string{"/orders — verify your trade", "/status — see portfolio"}
		}
		return []string{"/pnl — check P&L", "/alert <sym> > <price>"}
	case "orders":
		return []string{"/pnl — profit/loss", "/history — full trade log"}
	case "pnl":
		return []string{"/analytics — deeper metrics", "/risk — set guardrails"}
	case "history":
		return []string{"/pnl", "/analytics"}
	case "status", "portfolio":
		if len(ctx.TopPositions) > 0 {
			return []string{"/pnl", "/analyze " + ctx.TopPositions[0]}
		}
		return []string{"/price BTC ETH SOL", "/analyze BTC"}
	case "snapshot":
		return []string{"/dashboard — live view", "/pnl"}
	case "balances", "positions":
		return []string{"/pnl", "/status"}

	// Backtesting & automation
	case "backtest":
		return []string{"/backtest activate <preset> <sym>", "/auto list"}
	case "auto", "automation":
		return []string{"/auto list", "/risk set daily-loss 5"}
	case "trigger":
		return []string{"/trigger list", "/alert <sym> > <price>"}
	case "alert":
		return []string{"/notify set desktop on", "/trigger add <sym> > <price> buy <qty>"}
	case "strategy":
		return []string{"/strategy list", "/auto list"}
	case "risk":
		return []string{"/buy <sym> <qty> — trade with guardrails"}

	// Multi-vertical
	case "stock":
		return []string{"/screen — stock screener", "/chart <sym>"}
	case "screen":
		return []string{"/stock <ticker>", "/analyze <sym>"}
	case "bet", "polymarket":
		return []string{"/polymarket scan", "/odds <event>"}
	case "odds", "lines":
		return []string{"/bet", "/polymarket scan"}
	case "swap":
		return []string{"/gas", "/balances"}
	case "gas":
		return []string{"/swap <from> <to> <amt>"}
	case "funding":
		return []string{"/analyze <sym>", "/connect hyperliquid"}
	case "wallet":
		return []string{"/balances", "/swap"}

	// Setup
	case "mcp":
		if ctx.MCPCount == 0 {
			return []string{"/mcp quick — install free tools"}
		}
		return []string{"/mcp list", "/analyze <sym>"}
	case "connect":
		return []string{"/balances", "/mcp list"}
	case "config":
		if !ctx.HasAIKey {
			return []string{"/config set anthropic_key <key>"}
		}
		return []string{"/status", "/price BTC"}
	case "model":
		return []string{"/consensus <sym> — test the model"}
	case "memory":
		return []string{"/status", "/analyze <sym>"}

	// Dashboard
	case "dashboard":
		return []string{"Esc — back to chat"}

	default:
		return nil
	}
}

// StageOrdinal returns the 1-based journey position for a stage.
func StageOrdinal(stage Stage) int {
	switch stage {
	case StageFresh:
		return 1
	case StageConfigured:
		return 2
	case StageAIReady:
		return 3
	case StageEquipped:
		return 4
	case StageTrading:
		return 5
	case StageAnalyzing:
		return 6
	case StageAdvanced:
		return 7
	default:
		return 1
	}
}

// StageLabel returns a human-readable stage name.
func StageLabel(stage Stage) string {
	switch stage {
	case StageFresh:
		return "Fresh Setup"
	case StageConfigured:
		return "Configured"
	case StageAIReady:
		return "AI Ready"
	case StageEquipped:
		return "Tool Equipped"
	case StageTrading:
		return "Trading"
	case StageAnalyzing:
		return "Analyzing"
	case StageAdvanced:
		return "Advanced"
	default:
		return "Fresh Setup"
	}
}

// OnboardingChecklist returns the core guided journey tasks with completion state.
func OnboardingChecklist(ctx StageContext) []ChecklistItem {
	return []ChecklistItem{
		{
			Label:   "Connect your PaperNick account",
			Command: "/config init",
			Done:    ctx.HasAPIKey,
		},
		{
			Label:   "Unlock AI assistant",
			Command: "/config set anthropic_key <key>",
			Done:    ctx.HasAIKey,
		},
		{
			Label:   "Install live MCP data tools",
			Command: "/mcp quick",
			Done:    ctx.MCPCount > 0,
		},
		{
			Label:   "Place first paper trade",
			Command: "/buy BTC 0.01",
			Done:    ctx.TradeCount > 0,
		},
		{
			Label:   "Run technical analysis",
			Command: "/analyze BTC",
			Done:    ctx.HasAnalyzed,
		},
		{
			Label:   "Backtest one strategy",
			Command: "/backtest run rsi-reversal BTC",
			Done:    ctx.HasBacktested,
		},
	}
}

// JourneyProgress returns completed and total onboarding tasks.
func JourneyProgress(ctx StageContext) (done int, total int) {
	items := OnboardingChecklist(ctx)
	for _, item := range items {
		if item.Done {
			done++
		}
	}
	return done, len(items)
}
