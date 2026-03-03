package cli

import (
	"strings"
	"testing"
)

func TestCLICommands(t *testing.T) {
	out := CLICommands()

	// Must contain the expected subcommands.
	for _, want := range []string{"price", "portfolio", "orders", "ask", "analyze", "consensus"} {
		if !strings.Contains(out, want) {
			t.Errorf("CLICommands() missing %q", want)
		}
	}

	// Must mention the binary name.
	if !strings.Contains(out, "nickai") {
		t.Error("CLICommands() should mention 'nickai'")
	}
}

func TestFmtPrice(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{68000.00, "$68000.00"},
		{1.50, "$1.50"},
		{1.00, "$1.00"},
		{0.999999, "$0.999999"},
		{0.00042, "$0.000420"},
		{0.0, "$0.000000"},
		{100000.99, "$100000.99"},
	}

	for _, tc := range tests {
		got := fmtPrice(tc.input)
		if got != tc.want {
			t.Errorf("fmtPrice(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFmtPriceBoundary(t *testing.T) {
	// Exactly 1.0 should use 2-decimal format.
	got := fmtPrice(1.0)
	if !strings.HasPrefix(got, "$1.") {
		t.Errorf("fmtPrice(1.0) = %q, expected $1.xx format", got)
	}

	// Just under 1.0 should use 6-decimal format.
	got = fmtPrice(0.99)
	if count := strings.Count(got, "."); count != 1 {
		t.Errorf("fmtPrice(0.99) = %q, expected exactly one decimal point", got)
	}
	// The sub-1 format should have 6 decimal places.
	parts := strings.Split(got, ".")
	if len(parts) == 2 && len(parts[1]) != 6 {
		t.Errorf("fmtPrice(0.99) = %q, expected 6 decimal places for sub-$1 prices", got)
	}
}
