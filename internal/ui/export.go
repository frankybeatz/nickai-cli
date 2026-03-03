package ui

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// handleExport dispatches /export subcommands (trades, portfolio, backtest).
func (m *Model) handleExport(args []string) string {
	if len(args) == 0 {
		return renderExportHelp()
	}

	sub := args[0]
	switch sub {
	case "trades":
		return m.exportTrades()
	case "portfolio":
		return m.exportPortfolio()
	case "backtest":
		return m.exportBacktest()
	default:
		return ErrorStyle.Render("  Unknown export type: ") + sub + "\n" +
			DimStyle.Render("  Available: trades, portfolio, backtest")
	}
}

// renderExportHelp shows usage for the /export command.
func renderExportHelp() string {
	title := SecondaryStyle.Render("  Export Data to CSV")
	body := "\n" +
		"  " + CommandStyle.Render("/export trades") + DimStyle.Render("     — Export trade history") + "\n" +
		"  " + CommandStyle.Render("/export portfolio") + DimStyle.Render("  — Export portfolio positions") + "\n" +
		"  " + CommandStyle.Render("/export backtest") + DimStyle.Render("   — Export last backtest results") + "\n"
	return title + body
}

// exportDir returns ~/.nickai/exports/, creating it if needed.
func exportDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".nickai", "exports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create exports directory: %w", err)
	}
	return dir, nil
}

// writeCSV creates a timestamped CSV file and returns the absolute path.
func writeCSV(prefix string, header []string, rows [][]string) (string, error) {
	dir, err := exportDir()
	if err != nil {
		return "", err
	}

	stamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.csv", prefix, stamp)
	path := filepath.Join(dir, filename)

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("cannot create file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write(header); err != nil {
		return "", fmt.Errorf("cannot write header: %w", err)
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return "", fmt.Errorf("cannot write row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", fmt.Errorf("csv flush error: %w", err)
	}

	return path, nil
}

// exportTrades fetches orders from the API and writes them to CSV.
func (m *Model) exportTrades() string {
	if !m.client.IsConfigured() {
		return connectPrompt()
	}

	orders, err := m.client.GetOrders()
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch orders: ") + err.Error()
	}
	if len(orders) == 0 {
		return WarningStyle.Render("  No trades to export.")
	}

	header := []string{"date", "symbol", "side", "quantity", "price", "filled_price", "status"}
	rows := make([][]string, 0, len(orders))
	for _, o := range orders {
		rows = append(rows, []string{
			o.FilledAt,
			o.Symbol,
			o.Side,
			fmt.Sprintf("%.6f", o.Quantity),
			fmt.Sprintf("%.2f", o.Price),
			fmt.Sprintf("%.2f", o.FilledPrice),
			o.Status,
		})
	}

	path, err := writeCSV("trades", header, rows)
	if err != nil {
		return ErrorStyle.Render("  Export failed: ") + err.Error()
	}

	return BotMsgStyle.Render("nick: ") + "Exported " +
		BrandStyle.Render(fmt.Sprintf("%d trades", len(orders))) +
		" to\n  " + CommandStyle.Render(path)
}

// exportPortfolio fetches portfolio positions and writes them to CSV.
func (m *Model) exportPortfolio() string {
	if !m.client.IsConfigured() {
		return connectPrompt()
	}

	portfolio, err := m.client.GetPortfolio()
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch portfolio: ") + err.Error()
	}

	header := []string{"symbol", "quantity", "value", "avg_cost"}
	rows := make([][]string, 0, len(portfolio.Assets))
	for _, pos := range portfolio.Assets {
		avgCost := 0.0
		if pos.Quantity > 0 {
			avgCost = pos.Value / pos.Quantity
		}
		rows = append(rows, []string{
			pos.Symbol,
			fmt.Sprintf("%.6f", pos.Quantity),
			fmt.Sprintf("%.2f", pos.Value),
			fmt.Sprintf("%.2f", avgCost),
		})
	}

	if len(rows) == 0 {
		return WarningStyle.Render("  No positions to export.")
	}

	path, err := writeCSV("portfolio", header, rows)
	if err != nil {
		return ErrorStyle.Render("  Export failed: ") + err.Error()
	}

	return BotMsgStyle.Render("nick: ") + "Exported " +
		BrandStyle.Render(fmt.Sprintf("%d positions", len(rows))) +
		" to\n  " + CommandStyle.Render(path)
}

// exportBacktest exports the last backtest result to CSV.
func (m *Model) exportBacktest() string {
	if m.lastBacktestResult == nil {
		return WarningStyle.Render("  No backtest results to export.") + "\n" +
			DimStyle.Render("  Run a backtest first: ") + CommandStyle.Render("/backtest run <preset> <symbol>")
	}

	result := m.lastBacktestResult
	header := []string{"entry_time", "exit_time", "entry_price", "exit_price", "pnl_pct", "reason"}
	rows := make([][]string, 0, len(result.Trades))
	for _, t := range result.Trades {
		rows = append(rows, []string{
			t.EntryTime.Format(time.RFC3339),
			t.ExitTime.Format(time.RFC3339),
			fmt.Sprintf("%.2f", t.EntryPrice),
			fmt.Sprintf("%.2f", t.ExitPrice),
			fmt.Sprintf("%.2f", t.PnLPct),
			t.Reason,
		})
	}

	prefix := fmt.Sprintf("backtest-%s-%s", result.Symbol, result.Strategy)
	path, err := writeCSV(prefix, header, rows)
	if err != nil {
		return ErrorStyle.Render("  Export failed: ") + err.Error()
	}

	return BotMsgStyle.Render("nick: ") + "Exported " +
		BrandStyle.Render(fmt.Sprintf("%d backtest trades", len(result.Trades))) +
		" (" + result.Strategy + " on " + result.Symbol + ") to\n  " + CommandStyle.Render(path)
}
