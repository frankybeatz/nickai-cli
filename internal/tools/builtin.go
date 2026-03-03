package tools

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/risk"
)

// RiskLimitsFunc returns current risk limits. Set by the UI layer so tools
// can check limits without a direct dependency on the Model.
type RiskLimitsFunc func() *risk.RiskLimits

// RegisterBuiltins adds built-in trading tools to the registry.
// place_order requires user confirmation via the registry's confirmation channels.
func RegisterBuiltins(reg *Registry, client *api.PapernickClient, riskFn RiskLimitsFunc) {
	reg.Register(ToolEntry{
		Name:        "get_prices",
		Description: "Get current prices for cryptocurrency symbols. Returns symbol and current price.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"symbols": {
					"type": "array",
					"items": {"type": "string"},
					"description": "List of symbol tickers, e.g. [\"BTC\", \"ETH\", \"SOL\"]"
				}
			},
			"required": ["symbols"]
		}`),
		Execute: makeGetPrices(client),
		Source:  "builtin",
	})

	reg.Register(ToolEntry{
		Name:        "get_portfolio",
		Description: "Get the user's current portfolio including cash balance, total value, and all open positions with quantities and values.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {},
			"required": []
		}`),
		Execute: makeGetPortfolio(client),
		Source:  "builtin",
	})

	reg.Register(ToolEntry{
		Name:        "get_orders",
		Description: "Get the user's recent order history including filled, pending, and cancelled orders.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {},
			"required": []
		}`),
		Execute: makeGetOrders(client),
		Source:  "builtin",
	})

	reg.Register(ToolEntry{
		Name:        "place_order",
		Description: "Place a trade order. Symbol should include quote currency (e.g. BTCUSDT). For market orders, omit price. For limit orders, include price.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"symbol": {
					"type": "string",
					"description": "Trading pair symbol, e.g. BTCUSDT, ETHUSDT"
				},
				"side": {
					"type": "string",
					"enum": ["buy", "sell"],
					"description": "Order side"
				},
				"quantity": {
					"type": "number",
					"description": "Quantity to trade"
				},
				"type": {
					"type": "string",
					"enum": ["market", "limit"],
					"description": "Order type"
				},
				"price": {
					"type": "number",
					"description": "Limit price (required for limit orders, omit for market)"
				}
			},
			"required": ["symbol", "side", "quantity", "type"]
		}`),
		Execute: makePlaceOrderWithConfirm(reg, client, riskFn),
		Source:  "builtin",
	})

	reg.Register(ToolEntry{
		Name:        "rebalance_portfolio",
		Description: "Calculate trades needed to rebalance portfolio to target allocations. Returns a trade plan (does NOT execute). Call place_order for each trade to execute.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"targets": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"symbol": {"type": "string", "description": "Asset symbol, e.g. BTC, ETH, SOL"},
							"percent": {"type": "number", "description": "Target allocation percentage, e.g. 50 for 50%"}
						},
						"required": ["symbol", "percent"]
					},
					"description": "Target allocations. Percentages should sum to <= 100."
				}
			},
			"required": ["targets"]
		}`),
		Execute: makeRebalancePortfolio(client),
		Source:  "builtin",
	})

	reg.Register(ToolEntry{
		Name:        "get_trade_journal",
		Description: "Get the trade journal with all recorded trades, AI rationale, win rate, and P&L stats.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {},
			"required": []
		}`),
		Execute: makeGetTradeJournal(client),
		Source:  "builtin",
	})

	reg.Register(ToolEntry{
		Name:        "create_twap",
		Description: "Create a TWAP (Time-Weighted Average Price) strategy to scale into/out of a position over time. Splits a total dollar value into slices executed at regular intervals.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"symbol": {
					"type": "string",
					"description": "Asset symbol, e.g. ETH, BTC"
				},
				"side": {
					"type": "string",
					"enum": ["buy", "sell"],
					"description": "Order side"
				},
				"total_value": {
					"type": "number",
					"description": "Total dollar value to trade, e.g. 2000 for $2000"
				},
				"duration": {
					"type": "string",
					"description": "Time to spread execution over, e.g. 4h, 1h, 30m"
				}
			},
			"required": ["symbol", "side", "total_value", "duration"]
		}`),
		Execute: makeCreateTWAP(),
		Source:  "builtin",
	})

	reg.Register(ToolEntry{
		Name:        "get_analytics",
		Description: "Get portfolio analytics including Sharpe ratio, max drawdown, win rate, profit factor, and allocation breakdown.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {},
			"required": []
		}`),
		Execute: makeGetAnalytics(client),
		Source:  "builtin",
	})

	reg.Register(ToolEntry{
		Name:        "analyze_market",
		Description: "Run technical analysis on a cryptocurrency. Returns RSI, MACD, Bollinger Bands, trend, Fear & Greed index, and a summary recommendation.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"symbol": {
					"type": "string",
					"description": "Asset symbol, e.g. BTC, ETH, SOL"
				}
			},
			"required": ["symbol"]
		}`),
		Execute: makeAnalyzeMarket(client),
		Source:  "builtin",
	})

	reg.Register(ToolEntry{
		Name:        "backtest_strategy",
		Description: "Backtest a trading strategy against historical data. Define entry/exit conditions using technical indicators (rsi, macd, macd_histogram, macd_signal, bollinger_upper, bollinger_lower, sma20, sma50, ema12, ema26, price, fear_greed) with operators (<, >, crosses_above, crosses_below). Returns trade list, win rate, total return, Sharpe ratio, max drawdown, and equity curve.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"preset": {
					"type": "string",
					"description": "Name of a preset strategy (rsi-reversal, macd-crossover, bollinger-bounce, golden-cross, momentum, fear-and-greed, dip-buyer). If set, other fields are optional overrides."
				},
				"symbol": {
					"type": "string",
					"description": "Asset symbol, e.g. BTC, ETH, SOL"
				},
				"entry_conditions": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"indicator": {"type": "string", "description": "Indicator name: rsi, macd, macd_histogram, macd_signal, bollinger_upper, bollinger_lower, sma20, sma50, ema12, ema26, price, fear_greed"},
							"operator": {"type": "string", "description": "Comparison: <, >, crosses_above, crosses_below"},
							"value": {"type": "number", "description": "Threshold value"}
						},
						"required": ["indicator", "operator", "value"]
					},
					"description": "Conditions that must ALL be true to enter a position"
				},
				"exit_conditions": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"indicator": {"type": "string"},
							"operator": {"type": "string"},
							"value": {"type": "number"}
						},
						"required": ["indicator", "operator", "value"]
					},
					"description": "Conditions that must ALL be true to exit a position"
				},
				"stop_loss_pct": {
					"type": "number",
					"description": "Stop loss percentage, e.g. 5 for 5%"
				},
				"take_profit_pct": {
					"type": "number",
					"description": "Take profit percentage, e.g. 15 for 15%"
				},
				"position_size": {
					"type": "number",
					"description": "Position size as fraction (0.0-1.0), default 1.0"
				},
				"period": {
					"type": "string",
					"description": "Backtest period, e.g. 180d, 6m, 1y. Default 180d."
				}
			},
			"required": ["symbol"]
		}`),
		Execute: makeBacktestStrategy(),
		Source:  "builtin",
	})

	reg.Register(ToolEntry{
		Name:        "create_automation",
		Description: "Create an automated trading rule. Supports schedule-based (e.g. buy $100 BTC daily), condition-based (e.g. buy when BTC < 50000), and portfolio-based (e.g. sell when drawdown > 5%) rules.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"description": {
					"type": "string",
					"description": "Human-readable description of the rule"
				},
				"type": {
					"type": "string",
					"enum": ["schedule", "condition", "portfolio"],
					"description": "Rule type"
				},
				"schedule": {
					"type": "string",
					"description": "Schedule string: hourly, daily, weekly, 4h, 30m"
				},
				"symbol": {
					"type": "string",
					"description": "Symbol for condition rules"
				},
				"operator": {
					"type": "string",
					"enum": [">", "<"],
					"description": "Comparison operator for condition rules"
				},
				"target": {
					"type": "number",
					"description": "Target price for condition rules"
				},
				"metric_name": {
					"type": "string",
					"description": "Portfolio metric for portfolio rules: total_value, drawdown_pct"
				},
				"threshold": {
					"type": "number",
					"description": "Threshold for portfolio metric rules"
				},
				"action": {
					"type": "string",
					"enum": ["buy", "sell", "sell_all"],
					"description": "Trade action to take"
				},
				"action_symbol": {
					"type": "string",
					"description": "Symbol to trade"
				},
				"action_value": {
					"type": "number",
					"description": "Dollar value to trade"
				},
				"action_type": {
					"type": "string",
					"enum": ["market", "limit"],
					"description": "Order type (default: market)"
				},
				"max_fires": {
					"type": "integer",
					"description": "Maximum times to fire (0 = unlimited)"
				}
			},
			"required": ["description", "type", "action", "action_symbol", "action_value"]
		}`),
		Execute: makeCreateAutomation(),
		Source:  "builtin",
	})

	reg.Register(ToolEntry{
		Name:        "save_memory",
		Description: "Save a memory for future sessions. Use to remember user preferences, trade outcomes, key insights.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"type": {"type": "string", "enum": ["insight", "preference", "context"], "description": "insight (trade learnings), preference (user habits), context (key facts)"},
				"content": {"type": "string", "description": "The memory to save"},
				"tags": {"type": "array", "items": {"type": "string"}, "description": "Tags for categorization"}
			},
			"required": ["type", "content"]
		}`),
		Execute: makeSaveMemory(),
		Source:  "builtin",
	})

	reg.Register(ToolEntry{
		Name:        "recall_memory",
		Description: "Search saved memories by keyword. Returns user preferences, trade outcomes, insights from past sessions.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {"type": "string", "description": "Search query"}
			},
			"required": ["query"]
		}`),
		Execute: makeRecallMemory(),
		Source:  "builtin",
	})

	reg.Register(ToolEntry{
		Name:        "activate_strategy",
		Description: "Activate a backtest strategy as a live monitoring rule. Converts entry conditions into indicator-based automation checked every 60 seconds.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"strategy_name": {"type": "string", "description": "Strategy name"},
				"symbol": {"type": "string", "description": "Symbol to monitor"},
				"entry_conditions": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"indicator": {"type": "string"},
							"operator": {"type": "string"},
							"value": {"type": "number"}
						},
						"required": ["indicator", "operator", "value"]
					},
					"description": "Indicator conditions to monitor"
				},
				"action": {"type": "string", "enum": ["buy", "sell"], "description": "Action when conditions met"},
				"action_value": {"type": "number", "description": "Dollar value to trade"},
				"max_fires": {"type": "integer", "description": "Max times to fire (default 1)"}
			},
			"required": ["symbol", "entry_conditions", "action", "action_value"]
		}`),
		Execute: makeActivateStrategy(),
		Source:  "builtin",
	})
}

// --- Shared helpers ---

// formatOrderDisplay builds a one-line summary for the confirmation card.
func formatOrderDisplay(symbol, side string, qty float64, orderType string, price float64) string {
	s := strings.ToUpper(side) + " " + formatQty(qty) + " " + symbol + " (" + orderType + ")"
	if price > 0 {
		s += " @ " + formatMoney(price)
		total := qty * price
		s += " ≈ " + formatMoney(total)
	}
	return s
}

func formatQty(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func formatMoney(v float64) string {
	// Format with 2 decimal places and comma separators.
	neg := ""
	if v < 0 {
		neg = "-"
		v = -v
	}
	whole := int64(v)
	frac := int64((v - float64(whole)) * 100 + 0.5)
	if frac >= 100 {
		whole++
		frac -= 100
	}
	// Add commas to whole part.
	s := strconv.FormatInt(whole, 10)
	if len(s) > 3 {
		var parts []string
		for len(s) > 3 {
			parts = append([]string{s[len(s)-3:]}, parts...)
			s = s[:len(s)-3]
		}
		parts = append([]string{s}, parts...)
		s = strings.Join(parts, ",")
	}
	return neg + "$" + s + "." + fmt.Sprintf("%02d", frac)
}

// randomToolID generates a random alphanumeric string using crypto/rand.
func randomToolID(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	crand.Read(b)
	for i := range b {
		b[i] = chars[b[i]%byte(len(chars))]
	}
	return string(b)
}

// generateSyntheticHistory creates a random-walk price series ending at basePrice.
func generateSyntheticHistory(basePrice float64, n int) []float64 {
	data := make([]float64, n)
	data[n-1] = basePrice
	volatility := basePrice * 0.003
	for i := n - 2; i >= 0; i-- {
		delta := (float64(time.Now().UnixNano()%1000)/500 - 1) * volatility
		data[i] = data[i+1] + delta
		time.Sleep(time.Nanosecond)
	}
	return data
}
