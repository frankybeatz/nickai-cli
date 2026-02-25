package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const DefaultBaseURL = "https://paper.getnick.ai/api/v1"

// Config holds persistent NickAI CLI configuration.
type Config struct {
	APIKey       string `json:"api_key,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
	AnthropicKey string `json:"anthropic_key,omitempty"`
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

	return os.WriteFile(path, data, 0600)
}

// HasAPIKey returns true if an API key is configured.
func (c *Config) HasAPIKey() bool {
	return c.APIKey != ""
}

// MaskedKey returns the API key with most characters hidden.
func (c *Config) MaskedKey() string {
	return maskKey(c.APIKey)
}

// MaskedAnthropicKey returns the Anthropic key with most characters hidden.
func (c *Config) MaskedAnthropicKey() string {
	return maskKey(c.AnthropicKey)
}

// AnthropicKeyOrEnv returns the Anthropic key from config, falling back to
// the ANTHROPIC_API_KEY environment variable.
func (c *Config) AnthropicKeyOrEnv() string {
	if c.AnthropicKey != "" {
		return c.AnthropicKey
	}
	return os.Getenv("ANTHROPIC_API_KEY")
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
