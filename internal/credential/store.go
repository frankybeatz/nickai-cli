package credential

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Credential holds an exchange API credential.
type Credential struct {
	Name      string `json:"name"`
	Exchange  string `json:"exchange"`
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
}

var supportedExchanges = map[string]bool{
	"binance":     true,
	"coinbase":    true,
	"hyperliquid": true,
	"alpaca":      true,
	"polymarket":  true,
}

// IsSupportedExchange checks if the exchange name is recognized.
func IsSupportedExchange(exchange string) bool {
	return supportedExchanges[strings.ToLower(exchange)]
}

// SupportedExchanges returns the list of supported exchange names.
func SupportedExchanges() []string {
	return []string{"binance", "coinbase", "hyperliquid", "alpaca", "polymarket"}
}

// Store manages credentials on disk.
type Store struct {
	Credentials []Credential `json:"credentials"`
}

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nickai", "credentials.json"), nil
}

// Load reads credentials from disk.
func Load() (*Store, error) {
	s := &Store{}
	path, err := storePath()
	if err != nil {
		return s, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return &Store{}, err
	}
	return s, nil
}

// Save writes credentials to disk with 0600 permissions.
func (s *Store) Save() error {
	path, err := storePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// Add inserts a credential, replacing any existing one with the same name.
func (s *Store) Add(cred Credential) {
	s.Remove(cred.Name)
	cred.Exchange = strings.ToLower(cred.Exchange)
	s.Credentials = append(s.Credentials, cred)
}

// Remove deletes a credential by name. Returns true if found.
func (s *Store) Remove(name string) bool {
	for i, c := range s.Credentials {
		if c.Name == name {
			s.Credentials = append(s.Credentials[:i], s.Credentials[i+1:]...)
			return true
		}
	}
	return false
}

// Get returns a credential by name, or nil if not found.
func (s *Store) Get(name string) *Credential {
	for _, c := range s.Credentials {
		if c.Name == name {
			return &c
		}
	}
	return nil
}
