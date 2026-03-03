package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Unicode block characters for sparklines, ordered by height.
var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Braille dot patterns for higher-resolution mini charts.
// Each column represents 4 vertical pixels using braille dots.
// Row bits (from bottom): dot-7 (0x40), dot-3 (0x04), dot-2 (0x02), dot-1 (0x01)
// for left column. Right column uses dot-8, dot-6, dot-5, dot-4.
// We use single-column braille: left column only.
var brailleDots = [4]rune{
	0x2840, // ⡀ — bottom dot
	0x2804, // ⠄ — second from bottom
	0x2802, // ⠂ — third from bottom
	0x2801, // ⠁ — top dot
}

// brailleBase is the braille blank character.
const brailleBase = 0x2800

// Sparkline renders a single-line sparkline using Unicode block characters.
// data is the series of values, width is the maximum number of characters.
// Returns an empty string for empty or invalid data.
func Sparkline(data []float64, width int) string {
	cleaned := cleanData(data)
	if len(cleaned) == 0 || width <= 0 {
		return ""
	}

	minVal, maxVal := bounds(cleaned)
	span := maxVal - minVal
	if span == 0 {
		// All values identical — render mid-height bar.
		return strings.Repeat(string(sparkBlocks[3]), min(width, len(cleaned)))
	}

	w := min(width, len(cleaned))
	result := make([]rune, w)
	for i := 0; i < w; i++ {
		// Map data index to sparkline position.
		idx := i * len(cleaned) / w
		normalized := (cleaned[idx] - minVal) / span
		blockIdx := int(normalized * float64(len(sparkBlocks)-1))
		blockIdx = clampInt(blockIdx, 0, len(sparkBlocks)-1)
		result[i] = sparkBlocks[blockIdx]
	}

	return string(result)
}

// SparklineWithColor renders a colored sparkline. If the last value is greater
// than or equal to the first, the up style is used; otherwise the down style.
func SparklineWithColor(data []float64, width int, up, down lipgloss.Style) string {
	line := Sparkline(data, width)
	if line == "" {
		return ""
	}

	cleaned := cleanData(data)
	if len(cleaned) < 2 {
		return up.Render(line)
	}

	if cleaned[len(cleaned)-1] >= cleaned[0] {
		return up.Render(line)
	}
	return down.Render(line)
}

// MiniChart renders a multi-line ASCII chart using braille dot characters
// for higher vertical resolution (4 vertical dots per character cell).
// width is the number of columns, height is the number of text rows.
// Each row encodes 4 vertical pixels, so total vertical resolution is height*4.
func MiniChart(data []float64, width, height int) string {
	cleaned := cleanData(data)
	if len(cleaned) == 0 || width <= 0 || height <= 0 {
		return ""
	}

	minVal, maxVal := bounds(cleaned)
	span := maxVal - minVal
	if span == 0 {
		// All values identical — draw a flat line in the middle.
		return miniChartFlat(width, height)
	}

	totalRows := height * 4 // vertical pixel resolution

	// Sample data to fit width.
	w := min(width, len(cleaned))
	pixels := make([]int, w) // pixel height for each column (0-based from bottom)
	for i := 0; i < w; i++ {
		idx := i * len(cleaned) / w
		normalized := (cleaned[idx] - minVal) / span
		pxHeight := int(normalized * float64(totalRows-1))
		pixels[i] = clampInt(pxHeight, 0, totalRows-1)
	}

	// Build the grid row by row, top to bottom.
	lines := make([]string, height)
	for row := 0; row < height; row++ {
		// This text row covers vertical pixels from topPx to bottomPx.
		topPx := (height - 1 - row) * 4   // top pixel of this row (highest)
		bottomPx := topPx + 3              // bottom pixel of this row (lowest)
		_ = bottomPx

		var rowRunes []rune
		for col := 0; col < w; col++ {
			ch := rune(brailleBase)
			for dot := 0; dot < 4; dot++ {
				// dot 0 = bottom of cell, dot 3 = top of cell
				pixelY := topPx + (3 - dot)
				if pixels[col] >= pixelY {
					ch |= brailleDots[dot]
				}
			}
			rowRunes = append(rowRunes, ch)
		}
		lines[row] = string(rowRunes)
	}

	return strings.Join(lines, "\n")
}

// MiniChartWithColor renders a colored multi-line braille chart.
func MiniChartWithColor(data []float64, width, height int, up, down lipgloss.Style) string {
	chart := MiniChart(data, width, height)
	if chart == "" {
		return ""
	}

	cleaned := cleanData(data)
	if len(cleaned) < 2 {
		return up.Render(chart)
	}

	if cleaned[len(cleaned)-1] >= cleaned[0] {
		return up.Render(chart)
	}
	return down.Render(chart)
}

// miniChartFlat renders a flat line at mid-height for constant data.
func miniChartFlat(width, height int) string {
	lines := make([]string, height)
	midRow := height / 2
	for row := 0; row < height; row++ {
		if row == midRow {
			// Place a dot in the second-from-bottom position of this row.
			ch := rune(brailleBase) | brailleDots[0]
			lines[row] = strings.Repeat(string(ch), width)
		} else {
			lines[row] = strings.Repeat(string(rune(brailleBase)), width)
		}
	}
	return strings.Join(lines, "\n")
}

// cleanData filters out NaN and Inf values from a data slice.
func cleanData(data []float64) []float64 {
	result := make([]float64, 0, len(data))
	for _, v := range data {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			result = append(result, v)
		}
	}
	return result
}

// bounds returns the min and max of a non-empty slice.
func bounds(data []float64) (float64, float64) {
	minVal, maxVal := data[0], data[0]
	for _, v := range data[1:] {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	return minVal, maxVal
}

// clampInt clamps v to the range [lo, hi].
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
