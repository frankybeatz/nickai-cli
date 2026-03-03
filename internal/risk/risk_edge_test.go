package risk

import (
	"testing"

	"github.com/nickai/cli/internal/api"
)

func TestCheckOrder_ZeroPrice(t *testing.T) {
	limits := &RiskLimits{MaxOrderValue: 5000}
	portfolio := &api.Portfolio{TotalValue: 100000, Cash: 50000}

	// Zero price means orderValue = qty * 0 = 0, which is within any limit.
	result := CheckOrder(limits, portfolio, "BTCUSDT", "buy", 1, 0)
	if !result.Allowed {
		t.Errorf("expected zero-price order to be allowed (value is $0), got: %s", result.Reason)
	}
}

func TestCheckOrder_NilPortfolio(t *testing.T) {
	limits := &RiskLimits{
		MaxOrderValue:  5000,
		MaxPositionPct: 25,
		DailyLossPct:   5,
	}

	// Nil portfolio should not panic. MaxOrderValue still applies.
	result := CheckOrder(limits, nil, "ETHUSDT", "buy", 0.05, 3000) // $150
	if !result.Allowed {
		t.Errorf("expected small order with nil portfolio to be allowed, got: %s", result.Reason)
	}

	// Large order should still be blocked by MaxOrderValue.
	result = CheckOrder(limits, nil, "ETHUSDT", "buy", 5, 3000) // $15000
	if result.Allowed {
		t.Error("expected large order to be blocked by MaxOrderValue even with nil portfolio")
	}
}

func TestCheckOrder_MultipleViolations(t *testing.T) {
	// Set limits that would trigger both MaxOrderValue AND MaxPositionPct.
	limits := &RiskLimits{
		MaxOrderValue:  1000,
		MaxPositionPct: 10,
	}
	portfolio := &api.Portfolio{
		TotalValue: 100000,
		Cash:       50000,
		Assets: []api.Position{
			{Symbol: "BTCUSDT", Value: 9000, Quantity: 0.15},
		},
	}

	// Order is $6000 (exceeds $1000 cap), and would push BTC to 15% (exceeds 10%).
	result := CheckOrder(limits, portfolio, "BTCUSDT", "buy", 0.1, 60000)
	if result.Allowed {
		t.Error("expected order to be blocked")
	}
	// Should report the first violation found (MaxOrderValue is checked first).
	if result.Reason == "" {
		t.Error("expected a reason for the rejection")
	}
}

func TestCheckOrder_AtExactLimit(t *testing.T) {
	limits := &RiskLimits{MaxOrderValue: 5000}
	portfolio := &api.Portfolio{TotalValue: 100000, Cash: 50000}

	// Order value exactly at the limit: 0.1 * 50000 = $5000.
	result := CheckOrder(limits, portfolio, "ETHUSDT", "buy", 0.1, 50000)
	if !result.Allowed {
		t.Errorf("expected order exactly at max order value to be allowed, got: %s", result.Reason)
	}

	// One dollar over: 0.1 * 50010 = $5001.
	result = CheckOrder(limits, portfolio, "ETHUSDT", "buy", 0.1, 50010)
	if result.Allowed {
		t.Error("expected order slightly over max order value to be blocked")
	}
}

func TestCheckOrder_SellBypassesConcentration(t *testing.T) {
	limits := &RiskLimits{MaxPositionPct: 10}
	portfolio := &api.Portfolio{
		TotalValue: 100000,
		Cash:       20000,
		Assets: []api.Position{
			{Symbol: "BTCUSDT", Value: 50000, Quantity: 0.8},
		},
	}

	// Selling should bypass MaxPositionPct check even though BTC is 50% of portfolio.
	result := CheckOrder(limits, portfolio, "BTCUSDT", "sell", 0.5, 60000)
	if !result.Allowed {
		t.Errorf("expected sell to bypass concentration check, got: %s", result.Reason)
	}

	// Buying the same amount should be blocked.
	result = CheckOrder(limits, portfolio, "BTCUSDT", "buy", 0.5, 60000)
	if result.Allowed {
		t.Error("expected buy to be blocked by concentration limit")
	}
}

func TestCheckOrder_EmptyLimits(t *testing.T) {
	limits := &RiskLimits{}
	portfolio := &api.Portfolio{TotalValue: 100000, Cash: 50000}

	// Empty limits (all zeros) should allow everything.
	result := CheckOrder(limits, portfolio, "BTCUSDT", "buy", 100, 60000) // $6M order
	if !result.Allowed {
		t.Errorf("expected empty limits to allow any order, got: %s", result.Reason)
	}
}
