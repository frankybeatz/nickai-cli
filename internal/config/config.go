package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/nickai/cli/internal/credential"
	"github.com/nickai/cli/internal/safefile"
)

const DefaultBaseURL = "https://paper.getnick.ai/api/v1"

// Config holds persistent NickAI CLI configuration.
type Config struct {
	APIKey       string            `json:"api_key,omitempty"`
	BaseURL      string            `json:"base_url,omitempty"`
	AnthropicKey string            `json:"anthropic_key,omitempty"`
	MinimaxKey   string            `json:"minimax_key,omitempty"`
	Theme        string            `json:"theme,omitempty"`
	Model        string            `json:"model,omitempty"`
	DataKeys     map[string]string `json:"data_keys,omitempty"` // premium data source API keys
	Vibe         string            `json:"vibe,omitempty"`
}

// configPath returns ~/.nickai/config.json.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nickai", "config.json"), nil
}

// Load reads config from disk, returning defaults if the file doesn't exist.
func Load() (*Config, error) {
	cfg := &Config{BaseURL: DefaultBaseURL}

	path, err := configPath()
	if err != nil {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return &Config{BaseURL: DefaultBaseURL}, err
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	return cfg, nil
}

// Save writes config to disk, creating ~/.nickai/ if needed.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return safefile.AtomicWrite(path, data, 0600)
}

// HasAPIKey returns true if an API key is configured (keyring or config file).
func (c *Config) HasAPIKey() bool {
	return c.HasAPIKeyAnywhere()
}

// MaskedKey returns the API key with most characters hidden.
func (c *Config) MaskedKey() string {
	return maskKey(c.APIKeyResolved())
}

// MaskedAnthropicKey returns the Anthropic key with most characters hidden.
// It checks the OS keyring first, then falls back to the config file value.
func (c *Config) MaskedAnthropicKey() string {
	if v, ok := credential.KeyringGet("anthropic_key"); ok {
		return maskKey(v)
	}
	return maskKey(c.AnthropicKey)
}

// AnthropicKeyOrEnv returns the Anthropic key, checking OS keyring first,
// then config file, then the ANTHROPIC_API_KEY environment variable.
func (c *Config) AnthropicKeyOrEnv() string {
	if v, ok := credential.KeyringGet("anthropic_key"); ok {
		return v
	}
	if c.AnthropicKey != "" {
		return c.AnthropicKey
	}
	return os.Getenv("ANTHROPIC_API_KEY")
}

// MinimaxKeyOrEnv returns the MiniMax key, checking OS keyring first,
// then config file, then the MINIMAX_API_KEY environment variable.
func (c *Config) MinimaxKeyOrEnv() string {
	if v, ok := credential.KeyringGet("minimax_key"); ok {
		return v
	}
	if c.MinimaxKey != "" {
		return c.MinimaxKey
	}
	return os.Getenv("MINIMAX_API_KEY")
}

// DataKeyOrEnv returns a premium data API key, checking OS keyring first,
// then config file, then an environment variable. The env var name is
// constructed as <PROVIDER>_API_KEY (uppercased, hyphens replaced with
// underscores).
func (c *Config) DataKeyOrEnv(provider string) string {
	keyringName := strings.ToLower(strings.ReplaceAll(provider, "-", "_")) + "_key"
	if v, ok := credential.KeyringGet(keyringName); ok {
		return v
	}
	if c.DataKeys != nil {
		if key, ok := c.DataKeys[provider]; ok && key != "" {
			return key
		}
	}
	envName := strings.ToUpper(strings.ReplaceAll(provider, "-", "_")) + "_API_KEY"
	return os.Getenv(envName)
}

// SetSecureKey stores a credential, preferring the OS keyring.
// If the keyring is available the value is stored there and the
// corresponding plaintext field in config.json is cleared.
// If the keyring is not available the value is written to the
// config struct (caller must still call Save).
//
// Supported key names: api_key, anthropic_key, minimax_key, openrouter_key.
func (c *Config) SetSecureKey(key, value string) {
	stored := credential.KeyringSet(key, value)

	switch key {
	case "api_key":
		if stored {
			c.APIKey = "" // clear plaintext
		} else {
			c.APIKey = value
		}
	case "anthropic_key":
		if stored {
			c.AnthropicKey = ""
		} else {
			c.AnthropicKey = value
		}
	case "minimax_key":
		if stored {
			c.MinimaxKey = ""
		} else {
			c.MinimaxKey = value
		}
	case "openrouter_key":
		if c.DataKeys == nil {
			c.DataKeys = make(map[string]string)
		}
		if stored {
			delete(c.DataKeys, "openrouter") // clear plaintext
		} else {
			c.DataKeys["openrouter"] = value
		}
	}
}

// DeleteSecureKey removes a credential from both the OS keyring and the
// config struct (caller must still call Save).
func (c *Config) DeleteSecureKey(key string) {
	credential.KeyringDelete(key)

	switch key {
	case "api_key":
		c.APIKey = ""
	case "anthropic_key":
		c.AnthropicKey = ""
	case "minimax_key":
		c.MinimaxKey = ""
	case "openrouter_key":
		if c.DataKeys != nil {
			delete(c.DataKeys, "openrouter")
		}
	}
}

// KeyStorage returns "keyring" if the given key is stored in the OS keyring,
// "config" if it exists in config.json, or "env"/"(not set)" otherwise.
// Useful for /config show to indicate where each secret lives.
func (c *Config) KeyStorage(key string) string {
	if _, ok := credential.KeyringGet(key); ok {
		return "keyring"
	}
	switch key {
	case "api_key":
		if c.APIKey != "" {
			return "config"
		}
	case "anthropic_key":
		if c.AnthropicKey != "" {
			return "config"
		}
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			return "env"
		}
	case "minimax_key":
		if c.MinimaxKey != "" {
			return "config"
		}
		if os.Getenv("MINIMAX_API_KEY") != "" {
			return "env"
		}
	case "openrouter_key":
		if c.DataKeys != nil {
			if v, ok := c.DataKeys["openrouter"]; ok && v != "" {
				return "config"
			}
		}
		if os.Getenv("OPENROUTER_API_KEY") != "" {
			return "env"
		}
	}
	return "(not set)"
}

// HasAPIKeyAnywhere returns true if an API key is available from any source
// (keyring, config file, or environment variable) for the paper trading service.
func (c *Config) HasAPIKeyAnywhere() bool {
	if _, ok := credential.KeyringGet("api_key"); ok {
		return true
	}
	return c.APIKey != ""
}

// APIKeyResolved returns the paper trading API key from the best available
// source: OS keyring first, then config file.
func (c *Config) APIKeyResolved() string {
	if v, ok := credential.KeyringGet("api_key"); ok {
		return v
	}
	return c.APIKey
}

// MaskedKeyResolved returns the masked API key from whichever source holds it.
func (c *Config) MaskedKeyResolved() string {
	return maskKey(c.APIKeyResolved())
}

// UseKeyring reports whether the OS keyring is available for secure storage.
func UseKeyring() bool {
	return credential.KeyringAvailable()
}

func maskKey(k string) string {
	if k == "" {
		return "(not set)"
	}
	if len(k) <= 8 {
		return k[:2] + "***"
	}
	return k[:4] + "..." + k[len(k)-4:]
}
