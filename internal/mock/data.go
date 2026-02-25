package mock

type AgentStatus int

const (
	StatusRunning AgentStatus = iota
	StatusStopped
	StatusError
)

func (s AgentStatus) String() string {
	switch s {
	case StatusRunning:
		return "running"
	case StatusStopped:
		return "stopped"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

type Agent struct {
	Name     string
	Strategy string
	Status   AgentStatus
	PnL      string
	Uptime   string
}

type Template struct {
	Name        string
	Description string
	Author      string
	Stars       int
	Tags        []string
}

func Agents() []Agent {
	return []Agent{
		{
			Name:     "alpha-scalper",
			Strategy: "Grid Trading",
			Status:   StatusRunning,
			PnL:      "+12.4%",
			Uptime:   "3d 14h",
		},
		{
			Name:     "eth-momentum",
			Strategy: "Momentum Breakout",
			Status:   StatusRunning,
			PnL:      "+5.7%",
			Uptime:   "1d 8h",
		},
		{
			Name:     "btc-dca-bot",
			Strategy: "DCA Accumulator",
			Status:   StatusStopped,
			PnL:      "+2.1%",
			Uptime:   "—",
		},
		{
			Name:     "arb-hunter",
			Strategy: "Cross-DEX Arbitrage",
			Status:   StatusError,
			PnL:      "-0.3%",
			Uptime:   "0h 12m",
		},
	}
}

func Templates() []Template {
	return []Template{
		{
			Name:        "Grid Trader Pro",
			Description: "Automated grid trading with dynamic range adjustment",
			Author:      "nickai-labs",
			Stars:       842,
			Tags:        []string{"grid", "spot", "beginner"},
		},
		{
			Name:        "Momentum Alpha",
			Description: "Trend-following strategy using EMA crossovers and volume",
			Author:      "quant-collective",
			Stars:       631,
			Tags:        []string{"momentum", "futures", "intermediate"},
		},
		{
			Name:        "DCA Stacker",
			Description: "Dollar-cost averaging with smart timing based on RSI",
			Author:      "nickai-labs",
			Stars:       1203,
			Tags:        []string{"dca", "spot", "beginner"},
		},
		{
			Name:        "MEV Shield",
			Description: "Sandwich attack protection with private mempool routing",
			Author:      "defi-guard",
			Stars:       415,
			Tags:        []string{"mev", "defi", "advanced"},
		},
		{
			Name:        "Yield Optimizer",
			Description: "Auto-compound and rebalance across DeFi yield farms",
			Author:      "yield-maxi",
			Stars:       578,
			Tags:        []string{"yield", "defi", "intermediate"},
		},
	}
}
