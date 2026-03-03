package config

import (
	"testing"

	"github.com/nickai/cli/internal/credential"
)

func TestSetSecureKey_Keyring(t *testing.T) {
	if !credential.KeyringAvailable() {
		t.Skip("OS keyring not available in this environment")
	}

	cfg := &Config{BaseURL: DefaultBaseURL}
	defer func() {
		// Clean up keyring entries created during the test.
		credential.KeyringDelete("anthropic_key")
		credential.KeyringDelete("api_key")
		credential.KeyringDelete("minimax_key")
		credential.KeyringDelete("openrouter_key")
	}()

	// Test anthropic_key: should go to keyring, config field stays empty.
	cfg.SetSecureKey("anthropic_key", "sk-ant-test123")
	if cfg.AnthropicKey != "" {
		t.Error("expected AnthropicKey config field to be empty when keyring available")
	}
	got := cfg.AnthropicKeyOrEnv()
	if got != "sk-ant-test123" {
		t.Errorf("AnthropicKeyOrEnv = %q, want %q", got, "sk-ant-test123")
	}

	// KeyStorage should report keyring.
	if s := cfg.KeyStorage("anthropic_key"); s != "keyring" {
		t.Errorf("KeyStorage(anthropic_key) = %q, want %q", s, "keyring")
	}

	// Test api_key.
	cfg.SetSecureKey("api_key", "paper-key-456")
	if cfg.APIKey != "" {
		t.Error("expected APIKey config field to be empty when keyring available")
	}
	if got := cfg.APIKeyResolved(); got != "paper-key-456" {
		t.Errorf("APIKeyResolved = %q, want %q", got, "paper-key-456")
	}
	if !cfg.HasAPIKey() {
		t.Error("HasAPIKey should return true when key is in keyring")
	}

	// Test openrouter_key via DataKeyOrEnv.
	cfg.SetSecureKey("openrouter_key", "or-key-789")
	if cfg.DataKeys != nil {
		if v, ok := cfg.DataKeys["openrouter"]; ok && v != "" {
			t.Error("expected openrouter DataKeys entry to be cleared when keyring available")
		}
	}
	if got := cfg.DataKeyOrEnv("openrouter"); got != "or-key-789" {
		t.Errorf("DataKeyOrEnv(openrouter) = %q, want %q", got, "or-key-789")
	}

	// Test DeleteSecureKey.
	cfg.DeleteSecureKey("anthropic_key")
	if got := cfg.AnthropicKeyOrEnv(); got != "" {
		// Ignore env var — only fail if keyring still returns the value.
		if v, ok := credential.KeyringGet("anthropic_key"); ok {
			t.Errorf("keyring still has anthropic_key after delete: %q", v)
		}
	}
}

func TestSetSecureKey_Fallback(t *testing.T) {
	// Even if keyring is available, test the plaintext path by directly setting fields.
	cfg := &Config{BaseURL: DefaultBaseURL}

	// Simulate keyring unavailable by setting fields directly.
	cfg.AnthropicKey = "sk-fallback"
	if got := cfg.AnthropicKeyOrEnv(); got != "sk-fallback" {
		t.Errorf("AnthropicKeyOrEnv = %q, want %q", got, "sk-fallback")
	}

	cfg.MinimaxKey = "mm-fallback"
	if got := cfg.MinimaxKeyOrEnv(); got != "mm-fallback" {
		t.Errorf("MinimaxKeyOrEnv = %q, want %q", got, "mm-fallback")
	}
}

func TestMaskedAnthropicKey_Keyring(t *testing.T) {
	if !credential.KeyringAvailable() {
		t.Skip("OS keyring not available in this environment")
	}

	cfg := &Config{BaseURL: DefaultBaseURL}
	defer credential.KeyringDelete("anthropic_key")

	credential.KeyringSet("anthropic_key", "sk-ant-abcdefghij")
	masked := cfg.MaskedAnthropicKey()
	if masked == "(not set)" {
		t.Error("MaskedAnthropicKey should not return (not set) when key is in keyring")
	}
	// Should be masked (starts with first 4 chars, ends with last 4).
	if masked != "sk-a...ghij" {
		t.Errorf("MaskedAnthropicKey = %q, want %q", masked, "sk-a...ghij")
	}
}

func TestUseKeyring(t *testing.T) {
	// Just verify it doesn't panic and is consistent.
	a := UseKeyring()
	b := UseKeyring()
	if a != b {
		t.Error("UseKeyring returned different values on consecutive calls")
	}
}
