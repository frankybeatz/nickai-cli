package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nickai/cli/internal/credential"
)

func TestKeyStorage_KeyringAvailable(t *testing.T) {
	if !credential.KeyringAvailable() {
		t.Skip("OS keyring not available in this environment")
	}

	cfg := &Config{BaseURL: DefaultBaseURL}
	defer credential.KeyringDelete("anthropic_key")

	cfg.SetSecureKey("anthropic_key", "sk-ant-keyring-test")

	// When keyring is available, the config field should be cleared.
	if cfg.AnthropicKey != "" {
		t.Error("expected AnthropicKey config field to be empty when keyring stores the value")
	}

	// KeyStorage should report "keyring".
	if s := cfg.KeyStorage("anthropic_key"); s != "keyring" {
		t.Errorf("KeyStorage = %q, want %q", s, "keyring")
	}
}

func TestKeyStorage_KeyringFallback(t *testing.T) {
	// Simulate keyring unavailable by setting the config field directly,
	// which is the path SetSecureKey takes when KeyringSet returns false.
	cfg := &Config{
		BaseURL:      DefaultBaseURL,
		AnthropicKey: "sk-fallback-test",
	}

	if s := cfg.KeyStorage("anthropic_key"); s != "config" {
		t.Errorf("KeyStorage = %q, want %q", s, "config")
	}

	if got := cfg.AnthropicKeyOrEnv(); got != "sk-fallback-test" {
		t.Errorf("AnthropicKeyOrEnv = %q, want %q", got, "sk-fallback-test")
	}
}

func TestMaskedKey_Short(t *testing.T) {
	// Keys <= 8 chars produce first 2 chars + "***".
	got := maskKey("abcd1234")
	want := "ab***"
	if got != want {
		t.Errorf("maskKey(%q) = %q, want %q", "abcd1234", got, want)
	}

	got = maskKey("ab")
	want = "ab***"
	if got != want {
		t.Errorf("maskKey(%q) = %q, want %q", "ab", got, want)
	}
}

func TestMaskedKey_Long(t *testing.T) {
	// Keys > 8 chars show first 4 + "..." + last 4.
	key := "sk-ant-abcdefghij"
	got := maskKey(key)
	want := "sk-a...ghij"
	if got != want {
		t.Errorf("maskKey(%q) = %q, want %q", key, got, want)
	}
}

func TestMaskedKey_Empty(t *testing.T) {
	got := maskKey("")
	want := "(not set)"
	if got != want {
		t.Errorf("maskKey(%q) = %q, want %q", "", got, want)
	}
}

func TestHasAPIKey(t *testing.T) {
	cfg := &Config{
		BaseURL: DefaultBaseURL,
		APIKey:  "paper-key-123",
	}
	if !cfg.HasAPIKey() {
		t.Error("HasAPIKey should return true when APIKey is set in config")
	}
}

func TestHasAPIKey_Empty(t *testing.T) {
	cfg := &Config{BaseURL: DefaultBaseURL}
	// Unless keyring has it, this should be false.
	if credential.KeyringAvailable() {
		// Make sure keyring doesn't have a stale value.
		credential.KeyringDelete("api_key")
	}
	if cfg.HasAPIKey() {
		t.Error("HasAPIKey should return false when no key is set anywhere")
	}
}

func TestSetSecureKey_Empty(t *testing.T) {
	cfg := &Config{
		BaseURL:      DefaultBaseURL,
		AnthropicKey: "old-key",
	}

	// Setting empty value should clear the config field.
	cfg.SetSecureKey("anthropic_key", "")
	// The config field should be empty regardless of keyring path.
	// If keyring stored "", config clears. If keyring unavailable, config gets "".
	if cfg.AnthropicKey != "" {
		t.Errorf("AnthropicKey = %q after setting empty, want empty", cfg.AnthropicKey)
	}

	if credential.KeyringAvailable() {
		defer credential.KeyringDelete("anthropic_key")
	}
}

func TestMaskedAnthropicKey(t *testing.T) {
	cfg := &Config{
		BaseURL:      DefaultBaseURL,
		AnthropicKey: "sk-ant-longenoughkey123",
	}

	// Clear keyring so it falls through to config field.
	if credential.KeyringAvailable() {
		credential.KeyringDelete("anthropic_key")
	}

	got := cfg.MaskedAnthropicKey()
	want := "sk-a...y123"
	if got != want {
		t.Errorf("MaskedAnthropicKey = %q, want %q", got, want)
	}
}

func TestMaskedAnthropicKey_FromKeyring(t *testing.T) {
	if !credential.KeyringAvailable() {
		t.Skip("OS keyring not available in this environment")
	}

	cfg := &Config{BaseURL: DefaultBaseURL}
	defer credential.KeyringDelete("anthropic_key")

	credential.KeyringSet("anthropic_key", "sk-ant-keyringvalue1")
	got := cfg.MaskedAnthropicKey()
	want := "sk-a...lue1"
	if got != want {
		t.Errorf("MaskedAnthropicKey = %q, want %q", got, want)
	}
}

func TestConfigLoad_MissingFile(t *testing.T) {
	// Override HOME to a temp dir with no config file.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Create the .nickai dir but no config.json.
	if err := os.MkdirAll(filepath.Join(tmp, ".nickai"), 0700); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, DefaultBaseURL)
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", cfg.APIKey)
	}
}

func TestConfigLoad_ValidFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	dir := filepath.Join(tmp, ".nickai")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}

	data, _ := json.Marshal(&Config{
		APIKey:  "test-key",
		BaseURL: "https://custom.api.com",
		Theme:   "dark",
	})
	if err := os.WriteFile(filepath.Join(dir, "config.json"), data, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "test-key")
	}
	if cfg.BaseURL != "https://custom.api.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://custom.api.com")
	}
	if cfg.Theme != "dark" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "dark")
	}
}
