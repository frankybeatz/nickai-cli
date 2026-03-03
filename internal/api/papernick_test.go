package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nickai/cli/internal/config"
)

func TestNormalizeSymbol(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"btc", "BTCUSDT"},
		{"BTC", "BTCUSDT"},
		{"eth", "ETHUSDT"},
		{"  sol  ", "SOLUSDT"},
		{"BTCUSDT", "BTCUSDT"},
		{"btcusdt", "BTCUSDT"},
		{"ETHUSDC", "ETHUSDC"},
		{"ethusdc", "ETHUSDC"},
		{"BTCUSD", "BTCUSD"},
		{"btcusd", "BTCUSD"},
		{"DOGEUSDT", "DOGEUSDT"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := NormalizeSymbol(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeSymbol(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestAPIErrorFormat(t *testing.T) {
	tests := []struct {
		name    string
		err     APIError
		wantStr string
	}{
		{
			name:    "with message",
			err:     APIError{StatusCode: 401, Message: "Unauthorized"},
			wantStr: "API 401: Unauthorized",
		},
		{
			name:    "without message, with body",
			err:     APIError{StatusCode: 500, Body: "Internal Server Error"},
			wantStr: "API 500: Internal Server Error",
		},
		{
			name:    "message takes priority",
			err:     APIError{StatusCode: 404, Message: "Not Found", Body: "raw body"},
			wantStr: "API 404: Not Found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.err.Error()
			if got != tc.wantStr {
				t.Errorf("Error() = %q, want %q", got, tc.wantStr)
			}
		})
	}
}

func TestIsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{"with key", "sk-test-123", true},
		{"empty key", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &PapernickClient{apiKey: tc.apiKey}
			if got := c.IsConfigured(); got != tc.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGetPortfolioMocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/portfolio" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("missing or wrong API key header: %s", r.Header.Get("X-API-Key"))
		}
		json.NewEncoder(w).Encode(Portfolio{
			UserID:        "u-1",
			Cash:          50000,
			AvailableCash: 48000,
			TotalValue:    75000,
			Assets: []Position{
				{Symbol: "BTCUSDT", Quantity: 0.5, Value: 25000},
			},
		})
	}))
	defer server.Close()

	c := NewClient(&config.Config{BaseURL: server.URL, APIKey: "test-key"})
	p, err := c.GetPortfolio()
	if err != nil {
		t.Fatalf("GetPortfolio: %v", err)
	}

	if p.TotalValue != 75000 {
		t.Errorf("TotalValue = %v, want 75000", p.TotalValue)
	}
	if p.AvailableCash != 48000 {
		t.Errorf("AvailableCash = %v, want 48000", p.AvailableCash)
	}
	if len(p.Assets) != 1 {
		t.Fatalf("Assets count = %d, want 1", len(p.Assets))
	}
	if p.Assets[0].Symbol != "BTCUSDT" {
		t.Errorf("Assets[0].Symbol = %q, want BTCUSDT", p.Assets[0].Symbol)
	}
}

func TestGetOrdersMocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Order{
			{ID: "o-1", Symbol: "BTCUSDT", Side: "buy", Quantity: 0.1, Price: 68000, Status: "filled"},
			{ID: "o-2", Symbol: "ETHUSDT", Side: "sell", Quantity: 5.0, Price: 3500, Status: "pending"},
		})
	}))
	defer server.Close()

	c := NewClient(&config.Config{BaseURL: server.URL, APIKey: "test-key"})
	orders, err := c.GetOrders()
	if err != nil {
		t.Fatalf("GetOrders: %v", err)
	}

	if len(orders) != 2 {
		t.Fatalf("orders count = %d, want 2", len(orders))
	}
	if orders[0].ID != "o-1" {
		t.Errorf("orders[0].ID = %q, want o-1", orders[0].ID)
	}
	if orders[1].Status != "pending" {
		t.Errorf("orders[1].Status = %q, want pending", orders[1].Status)
	}
}

func TestGetPricesMultiple(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query params contain normalized symbols.
		symbol := r.URL.Query().Get("symbol")
		if symbol == "" {
			t.Error("expected symbol query parameter")
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"prices": []map[string]interface{}{
				{"symbol": "BTCUSDT", "price": 68000.50},
				{"symbol": "ETHUSDT", "price": 3456.78},
			},
		})
	}))
	defer server.Close()

	c := NewClient(&config.Config{BaseURL: server.URL, APIKey: "test-key"})
	prices, err := c.GetPrices([]string{"BTC", "ETH"})
	if err != nil {
		t.Fatalf("GetPrices: %v", err)
	}

	if len(prices) != 2 {
		t.Fatalf("prices count = %d, want 2", len(prices))
	}
	if prices[0].Price != 68000.50 {
		t.Errorf("prices[0].Price = %v, want 68000.50", prices[0].Price)
	}
}

func TestGetPricesSingle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Single symbol returns a flat object.
		json.NewEncoder(w).Encode(Price{Symbol: "SOLUSDT", Price: 150.25})
	}))
	defer server.Close()

	c := NewClient(&config.Config{BaseURL: server.URL, APIKey: "test-key"})
	prices, err := c.GetPrices([]string{"SOL"})
	if err != nil {
		t.Fatalf("GetPrices (single): %v", err)
	}

	if len(prices) != 1 {
		t.Fatalf("prices count = %d, want 1", len(prices))
	}
	if prices[0].Symbol != "SOLUSDT" {
		t.Errorf("Symbol = %q, want SOLUSDT", prices[0].Symbol)
	}
}

func TestHTTPErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid API key"})
	}))
	defer server.Close()

	c := NewClient(&config.Config{BaseURL: server.URL, APIKey: "bad-key"})
	_, err := c.GetPortfolio()
	if err == nil {
		t.Fatal("expected error for 401 response")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if apiErr.Message != "Invalid API key" {
		t.Errorf("Message = %q, want 'Invalid API key'", apiErr.Message)
	}
}

func TestUpdateConfig(t *testing.T) {
	c := &PapernickClient{
		baseURL: "http://old.example.com",
		apiKey:  "old-key",
	}

	c.UpdateConfig(&config.Config{
		BaseURL: "http://new.example.com",
		APIKey:  "new-key",
	})

	if c.baseURL != "http://new.example.com" {
		t.Errorf("baseURL = %q, want http://new.example.com", c.baseURL)
	}
	if c.apiKey != "new-key" {
		t.Errorf("apiKey = %q, want new-key", c.apiKey)
	}
}

func TestGetUserMocked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(User{ID: "u-1", Email: "test@example.com", Name: "Test User"})
	}))
	defer server.Close()

	c := NewClient(&config.Config{BaseURL: server.URL, APIKey: "test-key"})
	u, err := c.GetUser()
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if u.Name != "Test User" {
		t.Errorf("Name = %q, want 'Test User'", u.Name)
	}
	if u.Email != "test@example.com" {
		t.Errorf("Email = %q, want 'test@example.com'", u.Email)
	}
}
