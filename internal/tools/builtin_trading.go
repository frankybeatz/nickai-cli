package tools

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/journal"
	"github.com/nickai/cli/internal/risk"
	"github.com/nickai/cli/internal/telemetry"
)

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
					telemetry.Record("warn", "order", "risk_rejection", nil, map[string]string{"reason": result.Reason})
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
			telemetry.RecordError("order", "place_order", err)
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
