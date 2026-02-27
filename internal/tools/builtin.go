package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/nickai/cli/internal/api"
)

// RegisterBuiltins adds the 4 built-in trading tools to the registry.
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
		Execute: makePlaceOrder(client),
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

func makePlaceOrder(client *api.PapernickClient) ToolFunc {
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
