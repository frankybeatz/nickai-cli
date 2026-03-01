package risk

import (
	"testing"

	"github.com/nickai/cli/internal/api"
)

func TestCheckOrderMaxOrderValue(t *testing.T) {
	limits := &RiskLimits{MaxOrderValue: 5000}
	portfolio := &api.Portfolio{TotalValue: 100000, Cash: 50000}

	// Order within limit.
	result := CheckOrder(limits, portfolio, "BTCUSDT", "buy", 0.05, 60000) // $3000
	if !result.Allowed {
		t.Errorf("expected order to be allowed, got: %s", result.Reason)
	}

	// Order exceeding limit.
	result = CheckOrder(limits, portfolio, "BTCUSDT", "buy", 0.1, 60000) // $6000
	if result.Allowed {
		t.Error("expected order to be blocked by max order value")
	}
}

func TestCheckOrderMaxPositionPct(t *testing.T) {
	limits := &RiskLimits{MaxPositionPct: 25}
	portfolio := &api.Portfolio{
		TotalValue: 100000,
		Cash:       50000,
		Assets: []api.Position{
			{Symbol: "BTCUSDT", Value: 20000, Quantity: 0.3},
		},
	}

	// Buy that would push BTC to 30% of portfolio.
	result := CheckOrder(limits, portfolio, "BTCUSDT", "buy", 0.1, 60000) // +$6000 → 26000/100000 = 26%
	if result.Allowed {
		t.Error("expected order to be blocked by max position pct")
	}

	// Small buy that keeps within limit.
	result = CheckOrder(limits, portfolio, "BTCUSDT", "buy", 0.05, 60000) // +$3000 → 23000/100000 = 23%
	if !result.Allowed {
		t.Errorf("expected order to be allowed, got: %s", result.Reason)
	}

	// Sells should not be blocked by position pct.
	result = CheckOrder(limits, portfolio, "BTCUSDT", "sell", 0.1, 60000)
	if !result.Allowed {
		t.Errorf("expected sell to be allowed, got: %s", result.Reason)
	}
}

func TestCheckOrderNilLimits(t *testing.T) {
	result := CheckOrder(nil, nil, "BTCUSDT", "buy", 1, 60000)
	if !result.Allowed {
		t.Error("expected nil limits to allow all orders")
	}
}

func TestIsEmpty(t *testing.T) {
	empty := &RiskLimits{}
	if !empty.IsEmpty() {
		t.Error("expected empty limits to return IsEmpty=true")
	}

	notEmpty := &RiskLimits{MaxOrderValue: 1000}
	if notEmpty.IsEmpty() {
		t.Error("expected non-empty limits to return IsEmpty=false")
	}
}

func TestCheckOrderEmptyLimits(t *testing.T) {
	limits := &RiskLimits{}
	result := CheckOrder(limits, nil, "BTCUSDT", "buy", 1, 60000)
	if !result.Allowed {
		t.Error("expected empty limits to allow all orders")
	}
}

func TestCheckOrderDailyLossPct(t *testing.T) {
	limits := &RiskLimits{DailyLossPct: 5}
	portfolio := &api.Portfolio{TotalValue: 93000, Cash: 93000} // down 7% from 100K

	result := CheckOrder(limits, portfolio, "BTCUSDT", "buy", 0.01, 60000)
	if result.Allowed {
		t.Error("expected order to be blocked by daily loss pct")
	}

	// Portfolio still above threshold.
	portfolio2 := &api.Portfolio{TotalValue: 97000, Cash: 97000} // down 3%
	result = CheckOrder(limits, portfolio2, "BTCUSDT", "buy", 0.01, 60000)
	if !result.Allowed {
		t.Errorf("expected order to be allowed, got: %s", result.Reason)
	}
}
