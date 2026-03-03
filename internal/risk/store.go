package risk

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/safefile"
)

// RiskLimits defines portfolio risk guardrails.
type RiskLimits struct {
	MaxPositionPct float64 `json:"max_position_pct,omitempty"` // e.g. 10 = 10%
	DailyLossPct   float64 `json:"daily_loss_pct,omitempty"`   // e.g. 5 = 5%
	MaxOrderValue  float64 `json:"max_order_value,omitempty"`  // e.g. 5000 = $5K cap
}

// CheckResult reports whether an order is allowed.
type CheckResult struct {
	Allowed bool
	Reason  string
}

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nickai", "risk.json"), nil
}

// Load reads risk limits from ~/.nickai/risk.json.
func Load() (*RiskLimits, error) {
	path, err := storePath()
	if err != nil {
		return &RiskLimits{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RiskLimits{}, nil
		}
		return &RiskLimits{}, err
	}
	var limits RiskLimits
	if err := json.Unmarshal(data, &limits); err != nil {
		return &RiskLimits{}, err
	}
	return &limits, nil
}

// Save writes risk limits to disk.
func Save(limits *RiskLimits) error {
	path, err := storePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(limits, "", "  ")
	if err != nil {
		return err
	}
	return safefile.AtomicWrite(path, data, 0o600)
}

// IsEmpty returns true if no limits are set.
func (l *RiskLimits) IsEmpty() bool {
	return l.MaxPositionPct == 0 && l.DailyLossPct == 0 && l.MaxOrderValue == 0
}

// CheckOrder validates a proposed order against risk limits.
func CheckOrder(limits *RiskLimits, portfolio *api.Portfolio, symbol string, side string, qty float64, price float64) CheckResult {
	if limits == nil || limits.IsEmpty() {
		return CheckResult{Allowed: true}
	}

	orderValue := qty * price

	// MaxOrderValue: single order cap.
	if limits.MaxOrderValue > 0 && orderValue > limits.MaxOrderValue {
		return CheckResult{
			Allowed: false,
			Reason:  fmt.Sprintf("Max order value $%.0f exceeded ($%.0f)", limits.MaxOrderValue, orderValue),
		}
	}

	// DailyLossPct: block all trades if portfolio is down too much from $100K.
	if limits.DailyLossPct > 0 && portfolio != nil {
		startingBalance := 100000.0
		lossPct := (startingBalance - portfolio.TotalValue) / startingBalance * 100
		if lossPct > limits.DailyLossPct {
			return CheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("Daily loss limit %.1f%% exceeded (current loss: %.1f%%)", limits.DailyLossPct, lossPct),
			}
		}
	}

	// MaxPositionPct: prevent single-asset concentration.
	if limits.MaxPositionPct > 0 && portfolio != nil && side == "buy" {
		totalValue := portfolio.TotalValue
		if totalValue <= 0 {
			totalValue = portfolio.Cash
		}
		if totalValue > 0 {
			// Find existing position value for this symbol.
			existingValue := 0.0
			for _, pos := range portfolio.Assets {
				if pos.Symbol == symbol {
					existingValue = pos.Value
					break
				}
			}
			newPct := (existingValue + orderValue) / totalValue * 100
			if newPct > limits.MaxPositionPct {
				return CheckResult{
					Allowed: false,
					Reason:  fmt.Sprintf("Max position %.0f%% exceeded (%s would be %.1f%% of portfolio)", limits.MaxPositionPct, symbol, newPct),
				}
			}
		}
	}

	return CheckResult{Allowed: true}
}
