package analytics

import (
	"math"
	"strings"

	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/journal"
)

// Metrics holds computed portfolio analytics.
type Metrics struct {
	SharpeRatio    float64
	MaxDrawdownPct float64
	WinRate        float64
	ProfitFactor   float64
	TotalTrades    int
	WinCount       int
	LossCount      int
	TotalPnL       float64
	BestTrade      float64
	WorstTrade     float64
	AvgWin         float64
	AvgLoss        float64
}

// Allocation represents a single asset's share of the portfolio.
type Allocation struct {
	Symbol  string
	Value   float64
	Percent float64
}

// Calculate computes portfolio metrics from journal entries and current prices.
func Calculate(entries []journal.JournalEntry, currentPrices map[string]float64) *Metrics {
	m := &Metrics{TotalTrades: len(entries)}
	if len(entries) == 0 {
		return m
	}

	var profits, losses float64
	var returns []float64
	cumPnL := 0.0
	peak := 0.0

	for _, e := range entries {
		base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(e.Symbol, "USDT"), "USDC"), "USD")
		curPrice := currentPrices[base]
		if curPrice <= 0 || e.Price <= 0 {
			continue
		}

		var pnl float64
		if e.Side == "buy" {
			pnl = (curPrice - e.Price) * e.Quantity
		} else {
			pnl = (e.Price - curPrice) * e.Quantity
		}

		m.TotalPnL += pnl
		returns = append(returns, pnl/(e.Price*e.Quantity)) // return fraction

		if pnl > 0 {
			m.WinCount++
			profits += pnl
			if pnl > m.BestTrade {
				m.BestTrade = pnl
			}
		} else {
			m.LossCount++
			losses += math.Abs(pnl)
			if pnl < m.WorstTrade {
				m.WorstTrade = pnl
			}
		}

		// Max drawdown tracking.
		cumPnL += pnl
		if cumPnL > peak {
			peak = cumPnL
		}
		if peak > 0 {
			dd := (peak - cumPnL) / peak * 100
			if dd > m.MaxDrawdownPct {
				m.MaxDrawdownPct = dd
			}
		}
	}

	evaluated := m.WinCount + m.LossCount
	if evaluated > 0 {
		m.WinRate = float64(m.WinCount) / float64(evaluated) * 100
	}
	if losses > 0 {
		m.ProfitFactor = profits / losses
	}
	if m.WinCount > 0 {
		m.AvgWin = profits / float64(m.WinCount)
	}
	if m.LossCount > 0 {
		m.AvgLoss = losses / float64(m.LossCount)
	}

	// Sharpe ratio: mean(returns) / stddev(returns) * sqrt(252).
	if len(returns) > 1 {
		mean := 0.0
		for _, r := range returns {
			mean += r
		}
		mean /= float64(len(returns))

		variance := 0.0
		for _, r := range returns {
			variance += (r - mean) * (r - mean)
		}
		variance /= float64(len(returns) - 1)
		stddev := math.Sqrt(variance)

		if stddev > 0 {
			m.SharpeRatio = (mean / stddev) * math.Sqrt(252)
		}
	}

	return m
}

// CalcAllocation computes portfolio allocation percentages.
func CalcAllocation(portfolio *api.Portfolio) []Allocation {
	if portfolio == nil || len(portfolio.Assets) == 0 {
		return nil
	}

	total := portfolio.TotalValue
	if total <= 0 {
		total = portfolio.Cash
		for _, a := range portfolio.Assets {
			total += a.Value
		}
	}
	if total <= 0 {
		return nil
	}

	var allocs []Allocation

	// Cash allocation.
	if portfolio.Cash > 0 {
		allocs = append(allocs, Allocation{
			Symbol:  "CASH",
			Value:   portfolio.Cash,
			Percent: portfolio.Cash / total * 100,
		})
	}

	for _, a := range portfolio.Assets {
		if a.Value <= 0 {
			continue
		}
		sym := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(a.Symbol, "USDT"), "USDC"), "USD")
		allocs = append(allocs, Allocation{
			Symbol:  sym,
			Value:   a.Value,
			Percent: a.Value / total * 100,
		})
	}

	return allocs
}
