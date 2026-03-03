package ui

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/market"
)

// --- /chart: sparkline chart ---

// RenderChart renders a braille sparkline chart for a symbol.
func RenderChart(client *api.PapernickClient, symbol string, width int) string {
	cardWidth := min(width-4, 64)

	prices, err := client.GetPrices([]string{symbol})
	if err != nil {
		return ErrorStyle.Render("  Failed to fetch price: ") + err.Error()
	}
	if len(prices) == 0 {
		return DimStyle.Render("  No price data for: ") + symbol
	}

	currentPrice := prices[0].Price
	// Fetch real price history from Binance, fallback to synthetic.
	var data []float64
	if candles, err := market.FetchKlines(symbol, "1d", 50); err == nil && len(candles) > 0 {
		data = market.ClosePrices(candles)
	} else {
		data = generateSparklineData(currentPrice, 50)
	}

	return RenderSparkCard(prices[0].Symbol, data, cardWidth)
}

// RenderSparkCard renders a standalone sparkline chart card for a symbol.
// It shows a braille mini-chart, a single-line sparkline, and high/low stats.
func RenderSparkCard(symbol string, prices []float64, width int) string {
	if len(prices) == 0 {
		return DimStyle.Render("  No data for chart")
	}

	upStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
	downStyle := lipgloss.NewStyle().Foreground(ColorError)

	chartWidth := max(width-8, 10)

	// Braille mini-chart (3 rows of braille = 12 vertical pixels).
	miniChart := MiniChartWithColor(prices, chartWidth, 3, upStyle, downStyle)

	// Single-line sparkline.
	sparkline := SparklineWithColor(prices, chartWidth, upStyle, downStyle)

	// Compute high/low.
	cleaned := cleanData(prices)
	if len(cleaned) == 0 {
		return DimStyle.Render("  No valid data for chart")
	}
	high, low := bounds(cleaned)
	currentPrice := cleaned[len(cleaned)-1]

	// Trend indicator.
	var trendStr string
	if len(cleaned) >= 2 {
		pctChange := (cleaned[len(cleaned)-1] - cleaned[0]) / cleaned[0] * 100
		if pctChange >= 0 {
			trendStr = upStyle.Render(fmt.Sprintf("+%.2f%%", pctChange))
		} else {
			trendStr = downStyle.Render(fmt.Sprintf("%.2f%%", pctChange))
		}
	}

	var lines []string
	lines = append(lines, "")
	headerLine := BrandStyle.Render(symbol) + "  " +
		lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(formatPrice(currentPrice))
	if trendStr != "" {
		headerLine += "  " + trendStr
	}
	lines = append(lines, headerLine)
	lines = append(lines, "")
	lines = append(lines, miniChart)
	lines = append(lines, sparkline)
	lines = append(lines, "")
	lines = append(lines, DimStyle.Render("H ")+formatPrice(high)+
		DimStyle.Render("  L ")+formatPrice(low)+
		DimStyle.Render(fmt.Sprintf("  |  %d points", len(cleaned))))
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Width(max(width, 20)).
		Render(content)

	return SecondaryStyle.Render("  Sparkline Chart") + "\n" + box
}

// generateSparklineData produces a random walk of n points ending at basePrice.
func generateSparklineData(basePrice float64, n int) []float64 {
	data := make([]float64, n)
	data[n-1] = basePrice
	volatility := basePrice * 0.003
	for i := n - 2; i >= 0; i-- {
		delta := (rand.Float64()*2 - 1) * volatility
		data[i] = data[i+1] + delta
	}
	return data
}

// renderSparkline renders data as block characters with trend coloring.
// Delegates to the Sparkline/SparklineWithColor functions in sparkline.go.
func renderSparkline(data []float64, barWidth int) string {
	up := lipgloss.NewStyle().Foreground(ColorPrimary)
	down := lipgloss.NewStyle().Foreground(ColorError)
	return SparklineWithColor(data, barWidth, up, down)
}
