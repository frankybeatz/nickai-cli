package mcp

import (
	"os"
	"testing"
)

func TestPluginRegistryNotEmpty(t *testing.T) {
	if len(PluginRegistry) == 0 {
		t.Fatal("PluginRegistry should not be empty")
	}
}

func TestPluginRegistryContainsOfficialServers(t *testing.T) {
	names := []string{"filesystem", "github", "sqlite", "memory", "fetch"}
	for _, name := range names {
		p := GetPlugin(name)
		if p == nil {
			t.Errorf("PluginRegistry should contain %q", name)
		}
	}
}

func TestPluginRegistryContainsCuratedEntries(t *testing.T) {
	// Entries from CuratedRegistry should be imported.
	names := []string{"ccxt", "alpaca", "polymarket", "brave-search"}
	for _, name := range names {
		p := GetPlugin(name)
		if p == nil {
			t.Errorf("PluginRegistry should contain curated entry %q", name)
		}
	}
}

func TestGetPluginReturnsNilForUnknown(t *testing.T) {
	if p := GetPlugin("nonexistent_xyz"); p != nil {
		t.Error("GetPlugin should return nil for unknown plugin")
	}
}

func TestSearchPluginsAll(t *testing.T) {
	results := SearchPlugins("")
	if len(results) != len(PluginRegistry) {
		t.Errorf("SearchPlugins('') returned %d, want %d", len(results), len(PluginRegistry))
	}
}

func TestSearchPluginsByName(t *testing.T) {
	results := SearchPlugins("filesystem")
	if len(results) == 0 {
		t.Fatal("SearchPlugins('filesystem') returned 0 results")
	}
	found := false
	for _, r := range results {
		if r.Name == "filesystem" {
			found = true
			break
		}
	}
	if !found {
		t.Error("SearchPlugins('filesystem') should include filesystem")
	}
}

func TestSearchPluginsByTag(t *testing.T) {
	results := SearchPlugins("database")
	found := false
	for _, r := range results {
		if r.Name == "sqlite" {
			found = true
			break
		}
	}
	if !found {
		t.Error("SearchPlugins('database') should include sqlite")
	}
}

func TestSearchPluginsByDescription(t *testing.T) {
	results := SearchPlugins("knowledge graph")
	found := false
	for _, r := range results {
		if r.Name == "memory" {
			found = true
			break
		}
	}
	if !found {
		t.Error("SearchPlugins('knowledge graph') should include memory")
	}
}

func TestSearchPluginsNoResults(t *testing.T) {
	results := SearchPlugins("zzz_nonexistent_xyz")
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchPluginsCaseInsensitive(t *testing.T) {
	lower := SearchPlugins("github")
	upper := SearchPlugins("GITHUB")
	if len(lower) == 0 {
		t.Fatal("SearchPlugins('github') returned 0")
	}
	if len(lower) != len(upper) {
		t.Errorf("case-insensitive search inconsistent: lower=%d upper=%d", len(lower), len(upper))
	}
}

func TestPluginMatchesQuery(t *testing.T) {
	p := PluginEntry{
		Name:        "test-plugin",
		Description: "A test plugin for unit testing",
		Tags:        []string{"test", "mock"},
	}

	tests := []struct {
		query string
		want  bool
	}{
		{"test", true},
		{"plugin", true},
		{"unit testing", true},
		{"mock", true},
		{"production", false},
	}

	for _, tc := range tests {
		got := pluginMatchesQuery(p, tc.query)
		if got != tc.want {
			t.Errorf("pluginMatchesQuery(p, %q) = %v, want %v", tc.query, got, tc.want)
		}
	}
}

func TestPluginEntryFields(t *testing.T) {
	p := GetPlugin("filesystem")
	if p == nil {
		t.Fatal("filesystem plugin not found")
	}
	if p.Description == "" {
		t.Error("Description should not be empty")
	}
	if p.Command == "" {
		t.Error("Command should not be empty")
	}
	if len(p.Args) == 0 {
		t.Error("Args should not be empty")
	}
	if len(p.Tags) == 0 {
		t.Error("Tags should not be empty")
	}
	if !p.RequiresNpx {
		t.Error("filesystem should require npx")
	}
}

func TestMissingEnvKeysNoEnv(t *testing.T) {
	// filesystem has no env keys
	missing := MissingEnvKeys("filesystem", nil)
	if len(missing) != 0 {
		t.Errorf("filesystem should have no missing env keys, got %v", missing)
	}
}

func TestMissingEnvKeysWithRequiredKeys(t *testing.T) {
	// github requires GITHUB_TOKEN
	missing := MissingEnvKeys("github", nil)
	found := false
	for _, k := range missing {
		if k == "GITHUB_TOKEN" {
			found = true
		}
	}
	// Only expect missing if GITHUB_TOKEN is not set in the environment.
	if !found {
		// It might be set in the environment, so check.
		if val, _ := os.LookupEnv("GITHUB_TOKEN"); val == "" {
			t.Error("GITHUB_TOKEN should be in missing keys when not set")
		}
	}
}

func TestMissingEnvKeysProvided(t *testing.T) {
	extra := map[string]string{"GITHUB_TOKEN": "test-token"}
	missing := MissingEnvKeys("github", extra)
	for _, k := range missing {
		if k == "GITHUB_TOKEN" {
			t.Error("GITHUB_TOKEN should not be missing when provided in extraEnv")
		}
	}
}

func TestNoDuplicatePluginNames(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range PluginRegistry {
		if seen[p.Name] {
			t.Errorf("duplicate plugin name: %s", p.Name)
		}
		seen[p.Name] = true
	}
}
