package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nickai/cli/internal/risk"
)

// testPlaceOrderInput is the input shape expected by place_order.
type testPlaceOrderInput struct {
	Symbol   string  `json:"symbol"`
	Side     string  `json:"side"`
	Quantity float64 `json:"quantity"`
	Type     string  `json:"type"`
	Price    float64 `json:"price,omitempty"`
}

// makeTestOrderRegistry creates a registry with a simplified place_order tool
// that validates input without making network calls. This mirrors the
// validation logic in makePlaceOrderDirect and makePlaceOrderWithConfirm.
func makeTestOrderRegistry(riskFn RiskLimitsFunc) *Registry {
	r := NewRegistry()
	r.Register(ToolEntry{
		Name:        "place_order",
		Description: "Place a trade order (test version).",
		Source:      "builtin",
		Execute: func(_ context.Context, rawInput json.RawMessage) (string, error) {
			var input testPlaceOrderInput
			if err := json.Unmarshal(rawInput, &input); err != nil {
				return ErrorJSON("invalid input: " + err.Error()), nil
			}

			// Validate symbol.
			if input.Symbol == "" {
				return ErrorJSON("symbol is required"), nil
			}

			// Validate side.
			side := strings.ToLower(input.Side)
			if side != "buy" && side != "sell" {
				return ErrorJSON("side must be buy or sell"), nil
			}

			// Validate quantity.
			if input.Quantity <= 0 {
				return ErrorJSON("quantity must be positive"), nil
			}

			// Validate price.
			if input.Price < 0 {
				return ErrorJSON("price must not be negative"), nil
			}

			// Normalize symbol.
			symbol := strings.ToUpper(input.Symbol)
			if !strings.HasSuffix(symbol, "USDT") &&
				!strings.HasSuffix(symbol, "USDC") &&
				!strings.HasSuffix(symbol, "USD") {
				symbol += "USDT"
			}

			// Risk check.
			if riskFn != nil {
				limits := riskFn()
				if limits != nil && !limits.IsEmpty() {
					checkPrice := input.Price
					result := risk.CheckOrder(limits, nil, symbol, side, input.Quantity, checkPrice)
					if !result.Allowed {
						return ErrorJSON("Risk limit: " + result.Reason), nil
					}
				}
			}

			return ToJSON(map[string]any{
				"status": "filled",
				"symbol": symbol,
				"side":   side,
			}), nil
		},
	})
	return r
}

func executeOrderTool(r *Registry, input testPlaceOrderInput) string {
	raw, _ := json.Marshal(input)
	return r.ExecuteTool("place_order", raw)
}

func TestPlaceOrder_MissingSymbol(t *testing.T) {
	r := makeTestOrderRegistry(nil)
	result := executeOrderTool(r, testPlaceOrderInput{
		Symbol:   "",
		Side:     "buy",
		Quantity: 1,
		Type:     "market",
	})
	if !strings.Contains(result, "symbol is required") {
		t.Errorf("expected symbol required error, got: %s", result)
	}
}

func TestPlaceOrder_InvalidSide(t *testing.T) {
	r := makeTestOrderRegistry(nil)
	result := executeOrderTool(r, testPlaceOrderInput{
		Symbol:   "BTC",
		Side:     "short",
		Quantity: 1,
		Type:     "market",
	})
	if !strings.Contains(result, "side must be buy or sell") {
		t.Errorf("expected invalid side error, got: %s", result)
	}
}

func TestPlaceOrder_ZeroQuantity(t *testing.T) {
	r := makeTestOrderRegistry(nil)
	result := executeOrderTool(r, testPlaceOrderInput{
		Symbol:   "ETH",
		Side:     "buy",
		Quantity: 0,
		Type:     "market",
	})
	if !strings.Contains(result, "quantity must be positive") {
		t.Errorf("expected zero quantity error, got: %s", result)
	}
}

func TestPlaceOrder_SymbolNormalization(t *testing.T) {
	r := makeTestOrderRegistry(nil)
	result := executeOrderTool(r, testPlaceOrderInput{
		Symbol:   "btc",
		Side:     "buy",
		Quantity: 0.5,
		Type:     "market",
		Price:    60000,
	})

	// Should succeed and normalize to BTCUSDT.
	if strings.Contains(result, "error") {
		t.Fatalf("unexpected error: %s", result)
	}
	if !strings.Contains(result, "BTCUSDT") {
		t.Errorf("expected symbol to be normalized to BTCUSDT, got: %s", result)
	}

	// Symbol already with suffix should not get doubled.
	result = executeOrderTool(r, testPlaceOrderInput{
		Symbol:   "ethusdt",
		Side:     "sell",
		Quantity: 1,
		Type:     "market",
		Price:    3000,
	})
	if !strings.Contains(result, "ETHUSDT") {
		t.Errorf("expected symbol ETHUSDT, got: %s", result)
	}
	// Should NOT contain ETHUSDTUSDT.
	if strings.Contains(result, "ETHUSDTUSDT") {
		t.Errorf("symbol got double suffixed: %s", result)
	}
}

func TestPlaceOrder_RiskRejection(t *testing.T) {
	riskFn := func() *risk.RiskLimits {
		return &risk.RiskLimits{MaxOrderValue: 1000}
	}
	r := makeTestOrderRegistry(riskFn)

	result := executeOrderTool(r, testPlaceOrderInput{
		Symbol:   "BTC",
		Side:     "buy",
		Quantity: 1,
		Type:     "limit",
		Price:    60000, // $60000 order exceeds $1000 limit
	})
	if !strings.Contains(result, "Risk limit") {
		t.Errorf("expected risk rejection, got: %s", result)
	}
}

func TestPlaceOrder_NegativePrice(t *testing.T) {
	r := makeTestOrderRegistry(nil)
	result := executeOrderTool(r, testPlaceOrderInput{
		Symbol:   "SOL",
		Side:     "buy",
		Quantity: 10,
		Type:     "limit",
		Price:    -100,
	})
	if !strings.Contains(result, "price must not be negative") {
		t.Errorf("expected negative price error, got: %s", result)
	}
}
