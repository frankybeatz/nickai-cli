package backtest

// PresetStrategy is a curated backtest strategy users can run instantly.
type PresetStrategy struct {
	Strategy    Strategy
	Description string
}

// AnalysisPreset is a live AI-driven analysis template.
type AnalysisPreset struct {
	Name        string
	Description string
	Prompt      string   // structured AI prompt
	MCPTools    []string // required MCP servers
}

// backtestPresets are curated strategies for historical backtesting.
var backtestPresets = []PresetStrategy{
	{
		Description: "Buy when RSI signals oversold, sell when overbought",
		Strategy: Strategy{
			Name:          "rsi-reversal",
			EntryRules:    []Condition{{Indicator: "rsi", Operator: "<", Value: 30}},
			ExitRules:     []Condition{{Indicator: "rsi", Operator: ">", Value: 70}},
			StopLossPct:   5,
			TakeProfitPct: 15,
			PositionSize:  1.0,
			Period:        "180d",
			SlippageBps:   10,
			CommissionBps: 10,
		},
	},
	{
		Description: "Enter on MACD bullish crossover, exit on bearish",
		Strategy: Strategy{
			Name:          "macd-crossover",
			EntryRules:    []Condition{{Indicator: "macd_histogram", Operator: "crosses_above", Value: 0}},
			ExitRules:     []Condition{{Indicator: "macd_histogram", Operator: "crosses_below", Value: 0}},
			StopLossPct:   8,
			TakeProfitPct: 0,
			PositionSize:  1.0,
			Period:        "180d",
			SlippageBps:   10,
			CommissionBps: 10,
		},
	},
	{
		Description: "Buy at lower Bollinger Band, take profit at +10%",
		Strategy: Strategy{
			Name:          "bollinger-bounce",
			EntryRules:    []Condition{{Indicator: "price", Operator: "<", CompareWith: "bollinger_lower"}},
			ExitRules:     []Condition{},
			StopLossPct:   5,
			TakeProfitPct: 10,
			PositionSize:  1.0,
			Period:        "180d",
			SlippageBps:   10,
			CommissionBps: 10,
		},
	},
	{
		Description: "Enter when SMA20 crosses above SMA50 (golden cross)",
		Strategy: Strategy{
			Name:          "golden-cross",
			EntryRules:    []Condition{{Indicator: "sma20", Operator: "crosses_above", CompareWith: "sma50"}},
			ExitRules:     []Condition{{Indicator: "sma20", Operator: "crosses_below", CompareWith: "sma50"}},
			StopLossPct:   10,
			TakeProfitPct: 0,
			PositionSize:  1.0,
			Period:        "180d",
			SlippageBps:   10,
			CommissionBps: 10,
		},
	},
	{
		Description: "Multi-signal momentum: RSI > 50 + MACD bullish",
		Strategy: Strategy{
			Name: "momentum",
			EntryRules: []Condition{
				{Indicator: "rsi", Operator: ">", Value: 50},
				{Indicator: "macd_histogram", Operator: ">", Value: 0},
			},
			ExitRules:     []Condition{{Indicator: "rsi", Operator: "<", Value: 40}},
			StopLossPct:   7,
			TakeProfitPct: 20,
			PositionSize:  1.0,
			Period:        "180d",
			SlippageBps:   10,
			CommissionBps: 10,
		},
	},
	{
		Description: "Buy extreme fear + oversold RSI (contrarian)",
		Strategy: Strategy{
			Name: "fear-and-greed",
			EntryRules: []Condition{
				{Indicator: "rsi", Operator: "<", Value: 30},
				{Indicator: "fear_greed", Operator: "<", Value: 25},
			},
			ExitRules:     []Condition{{Indicator: "rsi", Operator: ">", Value: 60}},
			StopLossPct:   8,
			TakeProfitPct: 20,
			PositionSize:  1.0,
			Period:        "180d",
			SlippageBps:   10,
			CommissionBps: 10,
		},
	},
	{
		Description: "Buy dips with Bollinger + extreme fear confirmation",
		Strategy: Strategy{
			Name: "dip-buyer",
			EntryRules: []Condition{
				{Indicator: "price", Operator: "<", CompareWith: "bollinger_lower"},
				{Indicator: "fear_greed", Operator: "<", Value: 30},
			},
			ExitRules:     []Condition{},
			StopLossPct:   5,
			TakeProfitPct: 12,
			PositionSize:  1.0,
			Period:        "180d",
			SlippageBps:   10,
			CommissionBps: 10,
		},
	},
	// ── Short strategies ──
	{
		Description: "Short when RSI overbought, cover when oversold",
		Strategy: Strategy{
			Name:          "rsi-short",
			Side:          "short",
			EntryRules:    []Condition{{Indicator: "rsi", Operator: ">", Value: 70}},
			ExitRules:     []Condition{{Indicator: "rsi", Operator: "<", Value: 30}},
			StopLossPct:   5,
			TakeProfitPct: 15,
			PositionSize:  1.0,
			Period:        "180d",
			SlippageBps:   10,
			CommissionBps: 10,
		},
	},
	{
		Description: "Short on MACD bearish crossover, cover on bullish",
		Strategy: Strategy{
			Name:          "macd-short",
			Side:          "short",
			EntryRules:    []Condition{{Indicator: "macd_histogram", Operator: "crosses_below", Value: 0}},
			ExitRules:     []Condition{{Indicator: "macd_histogram", Operator: "crosses_above", Value: 0}},
			StopLossPct:   8,
			TakeProfitPct: 0,
			PositionSize:  1.0,
			Period:        "180d",
			SlippageBps:   10,
			CommissionBps: 10,
		},
	},
	// ── Research-derived strategies ──
	{
		Description: "Risk-on when trend + momentum + volume align (AND logic, hysteresis exit)",
		Strategy: Strategy{
			Name: "and-tre-mom-dir",
			EntryRules: []Condition{
				{Indicator: "trend", Operator: ">", Value: 0.05},
				{Indicator: "momentum", Operator: ">", Value: 0.05},
				{Indicator: "dir_volume", Operator: ">", Value: 0.05},
			},
			ExitRules: []Condition{
				{Indicator: "trend", Operator: "<", Value: -0.05},
				{Indicator: "momentum", Operator: "<", Value: -0.05},
				{Indicator: "dir_volume", Operator: "<", Value: -0.05},
			},
			ExitLogic:     "or",
			StopLossPct:   0,
			TakeProfitPct: 0,
			PositionSize:  1.0,
			Period:        "1y",
			SlippageBps:   10,
			CommissionBps: 10,
		},
	},
	{
		Description: "Risk-on when trend + momentum both positive (2-feature AND)",
		Strategy: Strategy{
			Name: "and-tre-mom",
			EntryRules: []Condition{
				{Indicator: "trend", Operator: ">", Value: 0.05},
				{Indicator: "momentum", Operator: ">", Value: 0.05},
			},
			ExitRules: []Condition{
				{Indicator: "trend", Operator: "<", Value: -0.05},
				{Indicator: "momentum", Operator: "<", Value: -0.05},
			},
			ExitLogic:     "or",
			StopLossPct:   0,
			TakeProfitPct: 0,
			PositionSize:  1.0,
			Period:        "1y",
			SlippageBps:   10,
			CommissionBps: 10,
		},
	},
	{
		Description: "Avoid drawdowns + high vol regimes, stay in calm uptrends",
		Strategy: Strategy{
			Name: "calm-trend",
			EntryRules: []Condition{
				{Indicator: "trend", Operator: ">", Value: 0.05},
				{Indicator: "vol_regime", Operator: ">", Value: 0},
				{Indicator: "drawdown", Operator: ">", Value: -0.1},
			},
			ExitRules: []Condition{
				{Indicator: "trend", Operator: "<", Value: -0.05},
				{Indicator: "vol_regime", Operator: "<", Value: -0.3},
			},
			ExitLogic:     "or",
			StopLossPct:   0,
			TakeProfitPct: 0,
			PositionSize:  1.0,
			Period:        "1y",
			SlippageBps:   10,
			CommissionBps: 10,
		},
	},
}

// analysisPresets are live AI-driven analysis templates.
var analysisPresets = []AnalysisPreset{
	{
		Name:        "polymarket-scan",
		Description: "Scan top Polymarket events, find mispriced contracts",
		MCPTools:    []string{"polymarket", "brave-search"},
		Prompt: `Scan the top Polymarket events using the polymarket tools. For each event:
1. Get the current market odds
2. Search for related news using brave-search
3. Assess whether the market odds seem mispriced based on news context
4. Flag contracts where the gap between market odds and your assessed probability is largest
Present results in a table: Event | Market Odds | Your Estimate | Gap | Reasoning
Always note that prediction markets carry risk.`,
	},
	{
		Name:        "polymarket-deep",
		Description: "Deep analysis of a specific Polymarket event",
		MCPTools:    []string{"polymarket", "brave-search"},
		Prompt: `Do a deep analysis of this Polymarket event: %s
1. Fetch the current market data (odds, volume, liquidity)
2. Search for 5+ recent news articles about this topic
3. Analyze the news sentiment and key factors
4. Compare market implied probability vs your assessed probability
5. Give a verdict: is this contract overpriced, underpriced, or fair?
Always note that prediction markets carry risk.`,
	},
	{
		Name:        "sentiment-check",
		Description: "Search news + social for current sentiment on a token",
		MCPTools:    []string{"brave-search"},
		Prompt: `Search for the latest news and social sentiment for %s:
1. Search for recent news articles (last 7 days)
2. Identify key themes: bullish, bearish, or neutral
3. Note any upcoming events (earnings, upgrades, partnerships)
4. Give an overall sentiment score: Very Bearish / Bearish / Neutral / Bullish / Very Bullish
5. Highlight any contrarian signals`,
	},
	{
		Name:        "whale-watch",
		Description: "Check on-chain activity for large movements",
		MCPTools:    []string{"onchain", "web3"},
		Prompt: `Check on-chain activity for %s:
1. Look for large recent transactions (whale movements)
2. Check exchange inflows/outflows if available
3. Note any unusual smart contract interactions
4. Summarize: are whales accumulating or distributing?`,
	},
	{
		Name:        "defi-yield",
		Description: "Scan top DeFi yields and evaluate risk",
		MCPTools:    []string{"defillama"},
		Prompt: `Scan DeFi yield opportunities using defillama tools:
1. Find the top 10 yield opportunities across all chains
2. For each: protocol, chain, APY, TVL
3. Flag any suspiciously high APYs (potential rug risk)
4. Recommend the top 3 risk-adjusted opportunities
5. Note impermanent loss risk for LP positions`,
	},
}

// GetPresets returns all backtest preset strategies.
func GetPresets() []PresetStrategy {
	return backtestPresets
}

// GetPreset returns a preset by name, or nil if not found.
func GetPreset(name string) *PresetStrategy {
	for i := range backtestPresets {
		if backtestPresets[i].Strategy.Name == name {
			return &backtestPresets[i]
		}
	}
	return nil
}

// GetAnalysisPresets returns all analysis presets.
func GetAnalysisPresets() []AnalysisPreset {
	return analysisPresets
}

// GetAnalysisPreset returns an analysis preset by name, or nil if not found.
func GetAnalysisPreset(name string) *AnalysisPreset {
	for i := range analysisPresets {
		if analysisPresets[i].Name == name {
			return &analysisPresets[i]
		}
	}
	return nil
}
