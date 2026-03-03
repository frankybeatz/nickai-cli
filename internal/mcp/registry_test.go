package mcp

import (
	"testing"
)

func TestSearchRegistryAll(t *testing.T) {
	results := SearchRegistry("")
	if len(results) != len(CuratedRegistry) {
		t.Errorf("SearchRegistry('') returned %d entries, want %d", len(results), len(CuratedRegistry))
	}
}

func TestSearchRegistryByName(t *testing.T) {
	results := SearchRegistry("ccxt")
	if len(results) == 0 {
		t.Fatal("SearchRegistry('ccxt') returned 0 results")
	}
	found := false
	for _, r := range results {
		if r.Name == "ccxt" {
			found = true
			break
		}
	}
	if !found {
		t.Error("SearchRegistry('ccxt') did not return the ccxt entry")
	}
}

func TestSearchRegistryByTag(t *testing.T) {
	results := SearchRegistry("defi")
	if len(results) == 0 {
		t.Fatal("SearchRegistry('defi') returned 0 results")
	}
	// At least defillama and solana should match.
	names := map[string]bool{}
	for _, r := range results {
		names[r.Name] = true
	}
	if !names["defillama"] {
		t.Error("defi search should include defillama")
	}
}

func TestSearchRegistryByDescription(t *testing.T) {
	results := SearchRegistry("prediction")
	found := false
	for _, r := range results {
		if r.Name == "polymarket" {
			found = true
			break
		}
	}
	if !found {
		t.Error("SearchRegistry('prediction') should find polymarket")
	}
}

func TestSearchRegistryCaseInsensitive(t *testing.T) {
	lower := SearchRegistry("binance")
	upper := SearchRegistry("BINANCE")
	mixed := SearchRegistry("Binance")

	if len(lower) == 0 {
		t.Fatal("SearchRegistry('binance') returned 0")
	}
	if len(lower) != len(upper) || len(lower) != len(mixed) {
		t.Errorf("case-insensitive search inconsistent: lower=%d upper=%d mixed=%d",
			len(lower), len(upper), len(mixed))
	}
}

func TestSearchRegistryNoResults(t *testing.T) {
	results := SearchRegistry("zzz_nonexistent_xyz")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestGetEntry(t *testing.T) {
	tests := []struct {
		name    string
		wantNil bool
	}{
		{"ccxt", false},
		{"alpaca", false},
		{"polymarket", false},
		{"brave-search", false},
		{"nonexistent", true},
	}

	for _, tc := range tests {
		entry := GetEntry(tc.name)
		if tc.wantNil && entry != nil {
			t.Errorf("GetEntry(%q) should be nil", tc.name)
		}
		if !tc.wantNil && entry == nil {
			t.Errorf("GetEntry(%q) should not be nil", tc.name)
		}
		if !tc.wantNil && entry != nil && entry.Name != tc.name {
			t.Errorf("GetEntry(%q).Name = %q", tc.name, entry.Name)
		}
	}
}

func TestRegistryEntryFields(t *testing.T) {
	entry := GetEntry("ccxt")
	if entry == nil {
		t.Fatal("ccxt entry not found")
	}

	if entry.DisplayName == "" {
		t.Error("DisplayName should not be empty")
	}
	if entry.Description == "" {
		t.Error("Description should not be empty")
	}
	if entry.Repo == "" {
		t.Error("Repo should not be empty")
	}
	if entry.Command == "" {
		t.Error("Command should not be empty")
	}
	if entry.Tier != TierVerified && entry.Tier != TierCommunity {
		t.Errorf("Tier = %q, expected verified or community", entry.Tier)
	}
	if len(entry.Capabilities) == 0 {
		t.Error("Capabilities should not be empty")
	}
	if len(entry.Tags) == 0 {
		t.Error("Tags should not be empty")
	}
}

func TestMatchesQueryDirectly(t *testing.T) {
	entry := RegistryEntry{
		Name:        "test-server",
		DisplayName: "Test Server",
		Description: "A test MCP server for unit testing",
		Tags:        []string{"test", "mock"},
	}

	tests := []struct {
		query string
		want  bool
	}{
		{"test", true},
		{"test server", true},  // matchesQuery expects pre-lowercased query
		{"unit testing", true},
		{"mock", true},
		{"production", false},
		{"", true}, // empty string matches everything via strings.Contains
	}

	for _, tc := range tests {
		got := matchesQuery(entry, tc.query)
		if got != tc.want {
			t.Errorf("matchesQuery(entry, %q) = %v, want %v", tc.query, got, tc.want)
		}
	}
}
