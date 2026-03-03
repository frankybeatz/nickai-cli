package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/nickai/cli/internal/analytics"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/indicators"
	"github.com/nickai/cli/internal/journal"
	"github.com/nickai/cli/internal/market"
)

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
