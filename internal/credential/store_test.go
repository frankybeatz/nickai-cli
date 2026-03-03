package credential

import (
	"testing"
)

func TestAdd_NewCredential(t *testing.T) {
	s := &Store{}

	s.Add(Credential{
		Name:      "my-binance",
		Exchange:  "Binance",
		APIKey:    "key123",
		APISecret: "secret456",
	})

	if len(s.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(s.Credentials))
	}
	if s.Credentials[0].Name != "my-binance" {
		t.Errorf("Name = %q, want %q", s.Credentials[0].Name, "my-binance")
	}
	// Exchange should be lowercased by Add.
	if s.Credentials[0].Exchange != "binance" {
		t.Errorf("Exchange = %q, want %q", s.Credentials[0].Exchange, "binance")
	}
	if s.Credentials[0].APIKey != "key123" {
		t.Errorf("APIKey = %q, want %q", s.Credentials[0].APIKey, "key123")
	}
}

func TestAdd_DuplicateExchange(t *testing.T) {
	s := &Store{}

	s.Add(Credential{
		Name:      "my-binance",
		Exchange:  "binance",
		APIKey:    "old-key",
		APISecret: "old-secret",
	})
	s.Add(Credential{
		Name:      "my-binance",
		Exchange:  "binance",
		APIKey:    "new-key",
		APISecret: "new-secret",
	})

	if len(s.Credentials) != 1 {
		t.Fatalf("expected 1 credential after replacing, got %d", len(s.Credentials))
	}
	if s.Credentials[0].APIKey != "new-key" {
		t.Errorf("APIKey = %q, want %q (should be replaced)", s.Credentials[0].APIKey, "new-key")
	}
}

func TestRemove_Existing(t *testing.T) {
	s := &Store{}
	s.Add(Credential{Name: "test-cred", Exchange: "coinbase", APIKey: "k", APISecret: "s"})

	removed := s.Remove("test-cred")
	if !removed {
		t.Error("Remove returned false for existing credential")
	}
	if len(s.Credentials) != 0 {
		t.Errorf("expected 0 credentials after remove, got %d", len(s.Credentials))
	}
}

func TestRemove_NonExistent(t *testing.T) {
	s := &Store{}
	s.Add(Credential{Name: "real-cred", Exchange: "alpaca", APIKey: "k", APISecret: "s"})

	removed := s.Remove("nonexistent")
	if removed {
		t.Error("Remove returned true for non-existent credential")
	}
	if len(s.Credentials) != 1 {
		t.Errorf("expected 1 credential unchanged, got %d", len(s.Credentials))
	}
}

func TestGet_Existing(t *testing.T) {
	s := &Store{}
	s.Add(Credential{Name: "my-hl", Exchange: "hyperliquid", APIKey: "hk", APISecret: "hs"})

	cred := s.Get("my-hl")
	if cred == nil {
		t.Fatal("Get returned nil for existing credential")
	}
	if cred.Exchange != "hyperliquid" {
		t.Errorf("Exchange = %q, want %q", cred.Exchange, "hyperliquid")
	}
	if cred.APIKey != "hk" {
		t.Errorf("APIKey = %q, want %q", cred.APIKey, "hk")
	}
}

func TestGet_NonExistent(t *testing.T) {
	s := &Store{}
	s.Add(Credential{Name: "some-cred", Exchange: "binance", APIKey: "k", APISecret: "s"})

	cred := s.Get("missing")
	if cred != nil {
		t.Errorf("Get returned non-nil for missing credential: %+v", cred)
	}
}

func TestSupportedExchanges(t *testing.T) {
	exchanges := SupportedExchanges()
	if len(exchanges) != 5 {
		t.Fatalf("expected 5 supported exchanges, got %d: %v", len(exchanges), exchanges)
	}

	expected := map[string]bool{
		"binance":     true,
		"coinbase":    true,
		"hyperliquid": true,
		"alpaca":      true,
		"polymarket":  true,
	}

	for _, ex := range exchanges {
		if !expected[ex] {
			t.Errorf("unexpected exchange in list: %q", ex)
		}
	}

	// Verify IsSupportedExchange matches.
	for _, ex := range exchanges {
		if !IsSupportedExchange(ex) {
			t.Errorf("IsSupportedExchange(%q) = false, want true", ex)
		}
	}

	// Verify unsupported exchange.
	if IsSupportedExchange("kraken") {
		t.Error("IsSupportedExchange(kraken) = true, want false")
	}
}
