package ui

import (
	"math"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestSparklineEmpty(t *testing.T) {
	if got := Sparkline(nil, 10); got != "" {
		t.Errorf("nil data: got %q, want empty", got)
	}
	if got := Sparkline([]float64{}, 10); got != "" {
		t.Errorf("empty data: got %q, want empty", got)
	}
}

func TestSparklineZeroWidth(t *testing.T) {
	if got := Sparkline([]float64{1, 2, 3}, 0); got != "" {
		t.Errorf("zero width: got %q, want empty", got)
	}
}

func TestSparklineAllSameValues(t *testing.T) {
	got := Sparkline([]float64{5, 5, 5, 5}, 4)
	// All same values should render mid-height blocks.
	if len([]rune(got)) != 4 {
		t.Errorf("all same: got %d runes, want 4", len([]rune(got)))
	}
	// Each rune should be the same mid-height block (▄ at index 3).
	for _, r := range got {
		if r != '▄' {
			t.Errorf("all same: unexpected rune %c, want ▄", r)
		}
	}
}

func TestSparklineAscending(t *testing.T) {
	data := []float64{0, 1, 2, 3, 4, 5, 6, 7}
	got := Sparkline(data, 8)
	runes := []rune(got)
	if len(runes) != 8 {
		t.Fatalf("ascending: got %d runes, want 8", len(runes))
	}
	// First rune should be lowest block, last should be highest.
	if runes[0] != '▁' {
		t.Errorf("ascending first: got %c, want ▁", runes[0])
	}
	if runes[7] != '█' {
		t.Errorf("ascending last: got %c, want █", runes[7])
	}
}

func TestSparklineDescending(t *testing.T) {
	data := []float64{7, 6, 5, 4, 3, 2, 1, 0}
	got := Sparkline(data, 8)
	runes := []rune(got)
	if len(runes) != 8 {
		t.Fatalf("descending: got %d runes, want 8", len(runes))
	}
	if runes[0] != '█' {
		t.Errorf("descending first: got %c, want █", runes[0])
	}
	if runes[7] != '▁' {
		t.Errorf("descending last: got %c, want ▁", runes[7])
	}
}

func TestSparklineNegativeValues(t *testing.T) {
	data := []float64{-10, -5, 0, 5, 10}
	got := Sparkline(data, 5)
	runes := []rune(got)
	if len(runes) != 5 {
		t.Fatalf("negative: got %d runes, want 5", len(runes))
	}
	if runes[0] != '▁' {
		t.Errorf("negative first: got %c, want ▁", runes[0])
	}
	if runes[4] != '█' {
		t.Errorf("negative last: got %c, want █", runes[4])
	}
}

func TestSparklineNaNFiltering(t *testing.T) {
	data := []float64{1, math.NaN(), 3, math.Inf(1), 5}
	got := Sparkline(data, 3)
	// After filtering NaN/Inf, we have [1, 3, 5] — 3 valid points.
	if got == "" {
		t.Error("NaN filtering: got empty, want non-empty")
	}
	runes := []rune(got)
	if len(runes) != 3 {
		t.Errorf("NaN filtering: got %d runes, want 3", len(runes))
	}
}

func TestSparklineWidthLargerThanData(t *testing.T) {
	data := []float64{1, 5}
	got := Sparkline(data, 100)
	// Width should be clamped to data length.
	runes := []rune(got)
	if len(runes) != 2 {
		t.Errorf("wide width: got %d runes, want 2", len(runes))
	}
}

func TestSparklineWithColorUp(t *testing.T) {
	up := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	down := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	data := []float64{1, 2, 3, 4, 5} // uptrend
	got := SparklineWithColor(data, 5, up, down)
	if got == "" {
		t.Error("colored up: got empty")
	}
}

func TestSparklineWithColorDown(t *testing.T) {
	up := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	down := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	data := []float64{5, 4, 3, 2, 1} // downtrend
	got := SparklineWithColor(data, 5, up, down)
	if got == "" {
		t.Error("colored down: got empty")
	}
}

func TestSparklineWithColorEmpty(t *testing.T) {
	up := lipgloss.NewStyle()
	down := lipgloss.NewStyle()
	got := SparklineWithColor(nil, 5, up, down)
	if got != "" {
		t.Errorf("colored empty: got %q, want empty", got)
	}
}

func TestMiniChartEmpty(t *testing.T) {
	if got := MiniChart(nil, 10, 3); got != "" {
		t.Errorf("nil data: got %q, want empty", got)
	}
	if got := MiniChart([]float64{}, 10, 3); got != "" {
		t.Errorf("empty data: got %q, want empty", got)
	}
}

func TestMiniChartZeroDimensions(t *testing.T) {
	if got := MiniChart([]float64{1, 2, 3}, 0, 3); got != "" {
		t.Errorf("zero width: got %q, want empty", got)
	}
	if got := MiniChart([]float64{1, 2, 3}, 10, 0); got != "" {
		t.Errorf("zero height: got %q, want empty", got)
	}
}

func TestMiniChartAllSame(t *testing.T) {
	got := MiniChart([]float64{5, 5, 5}, 3, 2)
	if got == "" {
		t.Error("all same: got empty, want non-empty")
	}
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Errorf("all same: got %d lines, want 2", len(lines))
	}
}

func TestMiniChartAscending(t *testing.T) {
	data := []float64{0, 1, 2, 3, 4, 5, 6, 7}
	got := MiniChart(data, 8, 3)
	if got == "" {
		t.Error("ascending: got empty")
	}
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Errorf("ascending: got %d lines, want 3", len(lines))
	}
	// Each line should have 8 runes.
	for i, line := range lines {
		runes := []rune(line)
		if len(runes) != 8 {
			t.Errorf("ascending line %d: got %d runes, want 8", i, len(runes))
		}
	}
}

func TestMiniChartNaNFiltering(t *testing.T) {
	data := []float64{math.NaN(), 1, math.NaN(), 3, math.Inf(-1)}
	got := MiniChart(data, 2, 2)
	// After filtering, we have [1, 3].
	if got == "" {
		t.Error("NaN filtering: got empty, want non-empty")
	}
}

func TestMiniChartWithColor(t *testing.T) {
	up := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	down := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

	dataUp := []float64{1, 2, 3, 4}
	got := MiniChartWithColor(dataUp, 4, 2, up, down)
	if got == "" {
		t.Error("colored up: got empty")
	}

	dataDown := []float64{4, 3, 2, 1}
	got = MiniChartWithColor(dataDown, 4, 2, up, down)
	if got == "" {
		t.Error("colored down: got empty")
	}
}

func TestCleanData(t *testing.T) {
	data := []float64{1, math.NaN(), 3, math.Inf(1), math.Inf(-1), 5}
	got := cleanData(data)
	if len(got) != 3 {
		t.Errorf("cleanData: got %d elements, want 3", len(got))
	}
	expected := []float64{1, 3, 5}
	for i, v := range got {
		if v != expected[i] {
			t.Errorf("cleanData[%d]: got %f, want %f", i, v, expected[i])
		}
	}
}

func TestBounds(t *testing.T) {
	lo, hi := bounds([]float64{3, 1, 4, 1, 5, 9, 2, 6})
	if lo != 1 {
		t.Errorf("bounds min: got %f, want 1", lo)
	}
	if hi != 9 {
		t.Errorf("bounds max: got %f, want 9", hi)
	}
}

func TestClampInt(t *testing.T) {
	tests := []struct {
		v, lo, hi, want int
	}{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 0, 0},
	}
	for _, tt := range tests {
		got := clampInt(tt.v, tt.lo, tt.hi)
		if got != tt.want {
			t.Errorf("clampInt(%d, %d, %d) = %d, want %d", tt.v, tt.lo, tt.hi, got, tt.want)
		}
	}
}
