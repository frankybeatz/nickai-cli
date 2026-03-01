package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/nickai/cli/internal/analytics"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/automation"
	"github.com/nickai/cli/internal/backtest"
	"github.com/nickai/cli/internal/indicators"
	"github.com/nickai/cli/internal/journal"
	"github.com/nickai/cli/internal/market"
	"github.com/nickai/cli/internal/risk"
	"github.com/nickai/cli/internal/strategy"
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
}

func makeGetPrices(client *api.PapernickClient) ToolFunc {
	return func(_ context.Context, rawInput json.RawMessage) (string, error) {
		var input struct {
			Symbols []string `json:"symbols"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}
		prices, err := client.GetPrices(input.Symbols)
		if err != nil {
			return ErrorJSON(err.Error()), nil
		}
		return ToJSON(prices), nil
	}
}

func makeGetPortfolio(client *api.PapernickClient) ToolFunc {
	return func(_ context.Context, _ json.RawMessage) (string, error) {
		portfolio, err := client.GetPortfolio()
		if err != nil {
			return ErrorJSON(err.Error()), nil
		}
		return ToJSON(portfolio), nil
	}
}

func makeGetOrders(client *api.PapernickClient) ToolFunc {
	return func(_ context.Context, _ json.RawMessage) (string, error) {
		orders, err := client.GetOrders()
		if err != nil {
			return ErrorJSON(err.Error()), nil
		}
		return ToJSON(orders), nil
	}
}

// makePlaceOrderWithConfirm wraps the order executor with risk checks and a confirmation flow.
// Risk check happens BEFORE the confirmation channel send — if it fails, the AI gets
// an error JSON so it understands why and can adjust.
func makePlaceOrderWithConfirm(reg *Registry, client *api.PapernickClient, riskFn RiskLimitsFunc) ToolFunc {
	direct := makePlaceOrderDirect(reg, client)
	return func(ctx context.Context, rawInput json.RawMessage) (string, error) {
		// Parse input to build a human-readable confirmation display.
		var input struct {
			Symbol   string  `json:"symbol"`
			Side     string  `json:"side"`
			Quantity float64 `json:"quantity"`
			Type     string  `json:"type"`
			Price    float64 `json:"price,omitempty"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}
		symbol := strings.ToUpper(input.Symbol)
		if !strings.HasSuffix(symbol, "USDT") &&
			!strings.HasSuffix(symbol, "USDC") &&
			!strings.HasSuffix(symbol, "USD") {
			symbol += "USDT"
		}

		// For market orders, fetch current price so the user knows the cost.
		displayPrice := input.Price
		if input.Type == "market" || input.Price == 0 {
			baseSymbol := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(symbol, "USDT"), "USDC"), "USD")
			if prices, err := client.GetPrices([]string{baseSymbol}); err == nil && len(prices) > 0 {
				displayPrice = prices[0].Price
			}
		}

		// Risk check BEFORE sending confirmation request.
		if riskFn != nil {
			limits := riskFn()
			if limits != nil && !limits.IsEmpty() {
				portfolio, _ := client.GetPortfolio()
				checkPrice := displayPrice
				if checkPrice == 0 {
					checkPrice = input.Price
				}
				result := risk.CheckOrder(limits, portfolio, symbol, input.Side, input.Quantity, checkPrice)
				if !result.Allowed {
					return ErrorJSON("Risk limit: " + result.Reason), nil
				}
			}
		}

		// Send confirmation request and block.
		reg.ConfirmCh <- ConfirmRequest{
			ToolName: "place_order",
			Input:    rawInput,
			Display:  formatOrderDisplay(symbol, input.Side, input.Quantity, input.Type, displayPrice),
		}
		resp := <-reg.ResponseCh

		if !resp.Approved {
			return ToJSON(map[string]string{
				"status": "cancelled",
				"reason": "User declined the trade",
			}), nil
		}
		return direct(ctx, rawInput)
	}
}

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

func makePlaceOrderDirect(reg *Registry, client *api.PapernickClient) ToolFunc {
	return func(_ context.Context, rawInput json.RawMessage) (string, error) {
		var input struct {
			Symbol   string  `json:"symbol"`
			Side     string  `json:"side"`
			Quantity float64 `json:"quantity"`
			Type     string  `json:"type"`
			Price    float64 `json:"price,omitempty"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}
		// Normalize symbol — append USDT if no quote currency.
		symbol := strings.ToUpper(input.Symbol)
		if !strings.HasSuffix(symbol, "USDT") &&
			!strings.HasSuffix(symbol, "USDC") &&
			!strings.HasSuffix(symbol, "USD") {
			symbol += "USDT"
		}
		order, err := client.PlaceOrder(api.PlaceOrderRequest{
			Symbol:   symbol,
			Side:     input.Side,
			Quantity: input.Quantity,
			Type:     input.Type,
			Price:    input.Price,
		})
		if err != nil {
			return ErrorJSON(err.Error()), nil
		}

		// Emit journal entry for AI-initiated trades.
		filledPrice := order.FilledPrice
		if filledPrice == 0 {
			filledPrice = order.Price
		}
		select {
		case reg.JournalCh <- journal.JournalEntry{
			ID:        randomToolID(8),
			OrderID:   order.ID,
			Symbol:    order.Symbol,
			Side:      order.Side,
			Quantity:  order.Quantity,
			Price:     filledPrice,
			Source:    "ai",
			Timestamp: time.Now(),
		}:
		default:
			// Don't block if channel is full.
		}

		return ToJSON(order), nil
	}
}

// makeRebalancePortfolio calculates trades needed to reach target allocations.
func makeRebalancePortfolio(client *api.PapernickClient) ToolFunc {
	return func(_ context.Context, rawInput json.RawMessage) (string, error) {
		var input struct {
			Targets []struct {
				Symbol  string  `json:"symbol"`
				Percent float64 `json:"percent"`
			} `json:"targets"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}

		// Validate targets sum <= 100%.
		totalPct := 0.0
		for _, t := range input.Targets {
			totalPct += t.Percent
		}
		if totalPct > 100 {
			return ErrorJSON(fmt.Sprintf("Target allocations sum to %.1f%% (must be <= 100%%)", totalPct)), nil
		}

		// Fetch portfolio and prices.
		portfolio, err := client.GetPortfolio()
		if err != nil {
			return ErrorJSON("failed to fetch portfolio: " + err.Error()), nil
		}

		symbols := make([]string, len(input.Targets))
		for i, t := range input.Targets {
			symbols[i] = strings.ToUpper(t.Symbol)
		}
		prices, err := client.GetPrices(symbols)
		if err != nil {
			return ErrorJSON("failed to fetch prices: " + err.Error()), nil
		}
		priceMap := make(map[string]float64)
		for _, p := range prices {
			// Map both normalized and base symbol.
			priceMap[p.Symbol] = p.Price
			base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(p.Symbol, "USDT"), "USDC"), "USD")
			priceMap[base] = p.Price
		}

		totalValue := portfolio.TotalValue
		if totalValue <= 0 {
			totalValue = portfolio.Cash
		}

		// Build position value map.
		positionValue := make(map[string]float64)
		for _, pos := range portfolio.Assets {
			base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(pos.Symbol, "USDT"), "USDC"), "USD")
			positionValue[base] = pos.Value
		}

		type tradePlan struct {
			Symbol   string  `json:"symbol"`
			Side     string  `json:"side"`
			Quantity float64 `json:"quantity"`
			Reason   string  `json:"reason"`
		}
		var trades []tradePlan

		for _, t := range input.Targets {
			sym := strings.ToUpper(t.Symbol)
			targetValue := totalValue * t.Percent / 100
			currentValue := positionValue[sym]
			diff := targetValue - currentValue

			price := priceMap[sym]
			if price <= 0 {
				continue
			}

			qty := math.Abs(diff) / price
			if math.Abs(diff) < 1 {
				continue // Skip tiny differences.
			}

			side := "buy"
			if diff < 0 {
				side = "sell"
			}

			trades = append(trades, tradePlan{
				Symbol:   sym + "USDT",
				Side:     side,
				Quantity: math.Round(qty*10000) / 10000, // 4 decimal places
				Reason:   fmt.Sprintf("Rebalance %s from %.1f%% to %.1f%% (diff %s%.0f)", sym, currentValue/totalValue*100, t.Percent, map[bool]string{true: "+", false: ""}[diff > 0], diff),
			})
		}

		if len(trades) == 0 {
			return ToJSON(map[string]string{
				"status":  "balanced",
				"message": "Portfolio is already at target allocations (within $1 threshold).",
			}), nil
		}

		return ToJSON(map[string]any{
			"status":      "rebalance_needed",
			"total_value": totalValue,
			"trades":      trades,
			"note":        "Call place_order for each trade to execute. Each will require user confirmation.",
		}), nil
	}
}

// makeGetTradeJournal returns all journal entries with stats.
func makeGetTradeJournal(client *api.PapernickClient) ToolFunc {
	return func(_ context.Context, _ json.RawMessage) (string, error) {
		entries, err := journal.All()
		if err != nil {
			return ErrorJSON("failed to load journal: " + err.Error()), nil
		}

		if len(entries) == 0 {
			return ToJSON(map[string]any{
				"entries":      []any{},
				"total_trades": 0,
				"message":      "No trades recorded yet.",
			}), nil
		}

		// Fetch current prices for win rate calculation.
		symbolSet := make(map[string]bool)
		for _, e := range entries {
			base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(e.Symbol, "USDT"), "USDC"), "USD")
			symbolSet[base] = true
		}
		symbols := make([]string, 0, len(symbolSet))
		for s := range symbolSet {
			symbols = append(symbols, s)
		}
		priceMap := make(map[string]float64)
		if prices, err := client.GetPrices(symbols); err == nil {
			for _, p := range prices {
				priceMap[p.Symbol] = p.Price
				base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(p.Symbol, "USDT"), "USDC"), "USD")
				priceMap[base] = p.Price
			}
		}

		wins := 0
		losses := 0
		totalPnL := 0.0
		for _, e := range entries {
			base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(e.Symbol, "USDT"), "USDC"), "USD")
			currentPrice := priceMap[base]
			if currentPrice <= 0 || e.Price <= 0 {
				continue
			}
			if e.Side == "buy" {
				pnl := (currentPrice - e.Price) * e.Quantity
				totalPnL += pnl
				if pnl > 0 {
					wins++
				} else {
					losses++
				}
			} else {
				pnl := (e.Price - currentPrice) * e.Quantity
				totalPnL += pnl
				if pnl > 0 {
					wins++
				} else {
					losses++
				}
			}
		}

		winRate := 0.0
		if wins+losses > 0 {
			winRate = float64(wins) / float64(wins+losses) * 100
		}

		return ToJSON(map[string]any{
			"entries":      entries,
			"total_trades": len(entries),
			"wins":         wins,
			"losses":       losses,
			"win_rate":     fmt.Sprintf("%.1f%%", winRate),
			"total_pnl":    fmt.Sprintf("$%.2f", totalPnL),
		}), nil
	}
}

// makeCreateTWAP creates a TWAP strategy and persists it.
func makeCreateTWAP() ToolFunc {
	return func(_ context.Context, rawInput json.RawMessage) (string, error) {
		var input struct {
			Symbol     string  `json:"symbol"`
			Side       string  `json:"side"`
			TotalValue float64 `json:"total_value"`
			Duration   string  `json:"duration"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}

		symbol := strings.ToUpper(input.Symbol)
		side := strings.ToLower(input.Side)
		if side != "buy" && side != "sell" {
			return ErrorJSON("side must be buy or sell"), nil
		}
		if input.TotalValue <= 0 {
			return ErrorJSON("total_value must be positive"), nil
		}

		dur, err := strategy.ParseDuration(input.Duration)
		if err != nil {
			return ErrorJSON(err.Error()), nil
		}

		sliceCount, intervalSec := strategy.CalcSlices(dur)
		sliceValue := input.TotalValue / float64(sliceCount)

		s := strategy.TWAPStrategy{
			ID:          randomToolID(8),
			Symbol:      symbol,
			Side:        side,
			TotalValue:  input.TotalValue,
			Duration:    input.Duration,
			IntervalSec: intervalSec,
			SliceCount:  sliceCount,
			SliceValue:  sliceValue,
			Executed:    0,
			Status:      "active",
			CreatedAt:   time.Now(),
			NextSliceAt: time.Now().Add(time.Duration(intervalSec) * time.Second),
		}

		if err := strategy.Add(s); err != nil {
			return ErrorJSON("failed to save strategy: " + err.Error()), nil
		}

		return ToJSON(map[string]any{
			"status":       "created",
			"id":           s.ID,
			"symbol":       symbol,
			"side":         side,
			"total_value":  fmt.Sprintf("$%.0f", input.TotalValue),
			"duration":     input.Duration,
			"slice_count":  sliceCount,
			"slice_value":  fmt.Sprintf("$%.2f", sliceValue),
			"interval":     fmt.Sprintf("%dm", intervalSec/60),
			"first_slice":  s.NextSliceAt.Format("3:04 PM"),
			"note":         "Strategy is active. Each slice will require confirmation before executing.",
		}), nil
	}
}

// makeGetAnalytics returns portfolio analytics metrics.
func makeGetAnalytics(client *api.PapernickClient) ToolFunc {
	return func(_ context.Context, _ json.RawMessage) (string, error) {
		entries, err := journal.All()
		if err != nil {
			return ErrorJSON("failed to load journal: " + err.Error()), nil
		}

		portfolio, err := client.GetPortfolio()
		if err != nil {
			return ErrorJSON("failed to load portfolio: " + err.Error()), nil
		}

		// Build price map.
		symbolSet := make(map[string]bool)
		for _, e := range entries {
			base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(e.Symbol, "USDT"), "USDC"), "USD")
			symbolSet[base] = true
		}
		for _, a := range portfolio.Assets {
			base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(a.Symbol, "USDT"), "USDC"), "USD")
			symbolSet[base] = true
		}
		symbols := make([]string, 0, len(symbolSet))
		for s := range symbolSet {
			symbols = append(symbols, s)
		}
		priceMap := make(map[string]float64)
		if len(symbols) > 0 {
			if prices, err := client.GetPrices(symbols); err == nil {
				for _, p := range prices {
					priceMap[p.Symbol] = p.Price
					base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(p.Symbol, "USDT"), "USDC"), "USD")
					priceMap[base] = p.Price
				}
			}
		}

		metrics := analytics.Calculate(entries, priceMap)
		allocs := analytics.CalcAllocation(portfolio)

		return ToJSON(map[string]any{
			"metrics":    metrics,
			"allocation": allocs,
			"portfolio": map[string]any{
				"total_value": portfolio.TotalValue,
				"cash":        portfolio.Cash,
			},
		}), nil
	}
}

// makeAnalyzeMarket runs technical analysis on a symbol.
func makeAnalyzeMarket(client *api.PapernickClient) ToolFunc {
	return func(_ context.Context, rawInput json.RawMessage) (string, error) {
		var input struct {
			Symbol string `json:"symbol"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}
		symbol := strings.ToUpper(input.Symbol)
		if symbol == "" {
			return ErrorJSON("symbol is required"), nil
		}

		prices, err := client.GetPrices([]string{symbol})
		if err != nil || len(prices) == 0 {
			return ErrorJSON("failed to fetch price for " + symbol), nil
		}
		currentPrice := prices[0].Price

		// Fetch real price history from Binance, fallback to synthetic.
		var history []float64
		if candles, err := market.FetchKlines(symbol, "1d", 50); err == nil && len(candles) > 0 {
			history = market.ClosePrices(candles)
		} else {
			history = generateSyntheticHistory(currentPrice, 50)
		}

		// Fetch Fear & Greed.
		fg, fgLabel, _ := indicators.FetchFearGreed()

		analysis := indicators.Analyze(symbol, currentPrice, history, fg, fgLabel)
		return ToJSON(analysis), nil
	}
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

// makeCreateAutomation creates an automation rule from AI input.
func makeCreateAutomation() ToolFunc {
	return func(_ context.Context, rawInput json.RawMessage) (string, error) {
		var input struct {
			Description string  `json:"description"`
			Type        string  `json:"type"`
			Schedule    string  `json:"schedule"`
			Symbol      string  `json:"symbol"`
			Operator    string  `json:"operator"`
			Target      float64 `json:"target"`
			MetricName  string  `json:"metric_name"`
			Threshold   float64 `json:"threshold"`
			Action      string  `json:"action"`
			ActionSym   string  `json:"action_symbol"`
			ActionVal   float64 `json:"action_value"`
			ActionType  string  `json:"action_type"`
			MaxFires    int     `json:"max_fires"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}

		ruleType := automation.RuleType(input.Type)
		if ruleType != automation.RuleSchedule && ruleType != automation.RuleCondition && ruleType != automation.RulePortfolio {
			return ErrorJSON("type must be schedule, condition, or portfolio"), nil
		}

		actionType := input.ActionType
		if actionType == "" {
			actionType = "market"
		}

		rule := automation.AutoRule{
			ID:           randomToolID(8),
			Description:  input.Description,
			Type:         ruleType,
			Schedule:     input.Schedule,
			Symbol:       strings.ToUpper(input.Symbol),
			Operator:     input.Operator,
			Target:       input.Target,
			MetricName:   input.MetricName,
			Threshold:    input.Threshold,
			Action:       input.Action,
			ActionSymbol: strings.ToUpper(input.ActionSym),
			ActionValue:  input.ActionVal,
			ActionType:   actionType,
			Status:       "active",
			MaxFires:     input.MaxFires,
			CreatedAt:    time.Now(),
		}

		// Parse schedule if applicable.
		if ruleType == automation.RuleSchedule && input.Schedule != "" {
			intervalSec, err := automation.ParseSchedule(input.Schedule)
			if err != nil {
				return ErrorJSON(err.Error()), nil
			}
			rule.IntervalSec = intervalSec
			rule.NextCheck = time.Now().Add(time.Duration(intervalSec) * time.Second)
		}

		if err := automation.Add(rule); err != nil {
			return ErrorJSON("failed to save automation: " + err.Error()), nil
		}

		return ToJSON(map[string]any{
			"status":      "created",
			"id":          rule.ID,
			"description": rule.Description,
			"type":        rule.Type,
			"action":      fmt.Sprintf("%s %s $%.0f", rule.Action, rule.ActionSymbol, rule.ActionValue),
			"schedule":    rule.Schedule,
			"note":        "Rule is active. Each fire will require user confirmation.",
		}), nil
	}
}

// makeBacktestStrategy creates the backtest executor.
func makeBacktestStrategy() ToolFunc {
	return func(_ context.Context, rawInput json.RawMessage) (string, error) {
		var input struct {
			Preset     string              `json:"preset"`
			Symbol     string              `json:"symbol"`
			Entry      []backtest.Condition `json:"entry_conditions"`
			Exit       []backtest.Condition `json:"exit_conditions"`
			StopLoss   float64             `json:"stop_loss_pct"`
			TakeProfit float64             `json:"take_profit_pct"`
			PosSize    float64             `json:"position_size"`
			Period     string              `json:"period"`
		}
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return ErrorJSON("invalid input: " + err.Error()), nil
		}

		var strat backtest.Strategy

		// If preset specified, load it as base.
		if input.Preset != "" {
			preset := backtest.GetPreset(input.Preset)
			if preset == nil {
				return ErrorJSON("unknown preset: " + input.Preset + ". Available: rsi-reversal, macd-crossover, bollinger-bounce, golden-cross, momentum, fear-and-greed, dip-buyer"), nil
			}
			strat = preset.Strategy
		}

		// Override with provided values.
		if input.Symbol != "" {
			strat.Symbol = strings.ToUpper(input.Symbol)
		}
		if len(input.Entry) > 0 {
			strat.EntryRules = input.Entry
		}
		if len(input.Exit) > 0 {
			strat.ExitRules = input.Exit
		}
		if input.StopLoss > 0 {
			strat.StopLossPct = input.StopLoss
		}
		if input.TakeProfit > 0 {
			strat.TakeProfitPct = input.TakeProfit
		}
		if input.PosSize > 0 {
			strat.PositionSize = input.PosSize
		}
		if input.Period != "" {
			strat.Period = input.Period
		}

		if strat.Symbol == "" {
			return ErrorJSON("symbol is required"), nil
		}
		if strat.Period == "" {
			strat.Period = "180d"
		}

		result, err := backtest.Run(strat)
		if err != nil {
			return ErrorJSON("backtest failed: " + err.Error()), nil
		}

		return ToJSON(result), nil
	}
}

// randomToolID generates a random alphanumeric string.
func randomToolID(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
