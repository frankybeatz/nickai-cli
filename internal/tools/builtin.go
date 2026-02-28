package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/nickai/cli/internal/api"
)

// RegisterBuiltins adds the 4 built-in trading tools to the registry.
// place_order requires user confirmation via the registry's confirmation channels.
func RegisterBuiltins(reg *Registry, client *api.PapernickClient) {
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
		Execute: makePlaceOrderWithConfirm(reg, client),
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

// makePlaceOrderWithConfirm wraps the order executor with a confirmation flow.
// It sends a ConfirmRequest on the registry channel and blocks until the UI responds.
func makePlaceOrderWithConfirm(reg *Registry, client *api.PapernickClient) ToolFunc {
	direct := makePlaceOrderDirect(client)
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

func makePlaceOrderDirect(client *api.PapernickClient) ToolFunc {
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
		return ToJSON(order), nil
	}
}
