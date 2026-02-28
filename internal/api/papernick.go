package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nickai/cli/internal/config"
)

// --- Response types ---

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Position struct {
	Symbol            string  `json:"symbol"`
	Quantity          float64 `json:"quantity"`
	ReservedQuantity  float64 `json:"reservedQuantity"`
	AvailableQuantity float64 `json:"availableQuantity"`
	Value             float64 `json:"value"`
	AvailableValue    float64 `json:"availableValue"`
}

type Portfolio struct {
	UserID        string     `json:"userId"`
	Cash          float64    `json:"cash"`
	ReservedCash  float64    `json:"reservedCash"`
	AvailableCash float64    `json:"availableCash"`
	Assets        []Position `json:"assets"`
	TotalValue    float64    `json:"totalValue"`
}

type Cash struct {
	Cash          float64 `json:"cash"`
	ReservedCash  float64 `json:"reservedCash"`
	AvailableCash float64 `json:"availableCash"`
}

type Order struct {
	ID          string  `json:"id"`
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"`
	Type        string  `json:"type"`
	Quantity    float64 `json:"quantity"`
	Price       float64 `json:"price"`
	FilledPrice float64 `json:"filledPrice"`
	Status      string  `json:"status"`
	FilledAt    string  `json:"filledAt"`
	OrderClass  string  `json:"orderClass"`
}

type Price struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
}

type Symbol struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

type PlaceOrderRequest struct {
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	Side     string  `json:"side"`
	Type     string  `json:"type"`
	Price    float64 `json:"price,omitempty"`
}

// --- API Error ---

type APIError struct {
	StatusCode int
	Message    string `json:"message"`
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API %d: %s", e.StatusCode, e.Body)
}

// --- Client ---

type PapernickClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewClient creates a client from the given config.
func NewClient(cfg *config.Config) *PapernickClient {
	return &PapernickClient{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// UpdateConfig refreshes the client's credentials from a new config.
func (c *PapernickClient) UpdateConfig(cfg *config.Config) {
	c.baseURL = cfg.BaseURL
	c.apiKey = cfg.APIKey
}

// IsConfigured returns true if an API key is set.
func (c *PapernickClient) IsConfigured() bool {
	return c.apiKey != ""
}

// --- HTTP helpers ---

func (c *PapernickClient) doGet(path string, out any) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *PapernickClient) doPost(path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *PapernickClient) doDelete(path string, out any) error {
	req, err := http.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *PapernickClient) do(req *http.Request, out any) error {
	body, err := c.doRaw(req)
	if err != nil {
		return err
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

// doRaw executes a request and returns the raw response body.
// Handles auth header and HTTP error status codes.
func (c *PapernickClient) doRaw(req *http.Request) ([]byte, error) {
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
		_ = json.Unmarshal(respBody, apiErr)
		return nil, apiErr
	}

	return respBody, nil
}

// --- API methods ---

func (c *PapernickClient) GetUser() (*User, error) {
	var u User
	return &u, c.doGet("/user", &u)
}

func (c *PapernickClient) GetPortfolio() (*Portfolio, error) {
	var p Portfolio
	return &p, c.doGet("/portfolio", &p)
}

func (c *PapernickClient) GetCash() (*Cash, error) {
	var cash Cash
	return &cash, c.doGet("/cash", &cash)
}

func (c *PapernickClient) GetOrders() ([]Order, error) {
	var orders []Order
	return orders, c.doGet("/orders", &orders)
}

// NormalizeSymbol uppercases and appends USDT if the symbol looks like a base
// asset (short ticker) rather than a full trading pair.
func NormalizeSymbol(s string) string {
	sym := strings.ToUpper(s)
	if len(sym) > 5 {
		return sym
	}
	return sym + "USDT"
}

func (c *PapernickClient) GetPrices(symbols []string) ([]Price, error) {
	normalized := make([]string, len(symbols))
	for i, s := range symbols {
		normalized[i] = NormalizeSymbol(s)
	}
	path := "/prices?symbol=" + strings.Join(normalized, ",")
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRaw(req)
	if err != nil {
		return nil, err
	}

	// Single symbol returns a flat object: {"symbol":"BTCUSDT","price":68649.5}
	// Multiple symbols returns a wrapper: {"prices":[{"symbol":...,"price":...}, ...]}
	var wrapper struct {
		Prices []Price `json:"prices"`
	}
	if err := json.Unmarshal(body, &wrapper); err == nil && len(wrapper.Prices) > 0 {
		return wrapper.Prices, nil
	}

	var single Price
	if err := json.Unmarshal(body, &single); err == nil && single.Symbol != "" {
		return []Price{single}, nil
	}

	preview := string(body)
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	return nil, fmt.Errorf("unexpected price response format: %s", preview)
}

func (c *PapernickClient) GetSymbols() ([]Symbol, error) {
	var syms []Symbol
	return syms, c.doGet("/symbols", &syms)
}

func (c *PapernickClient) PlaceOrder(req PlaceOrderRequest) (*Order, error) {
	var order Order
	return &order, c.doPost("/orders", req, &order)
}

func (c *PapernickClient) CancelOrder(id string) error {
	return c.doDelete("/orders/"+id, nil)
}

func (c *PapernickClient) TestConnection() (*User, error) {
	return c.GetUser()
}

// --- Registration (no auth required) ---

// CreateAccountUser is the user object returned by POST /users.
type CreateAccountUser struct {
	ID          string  `json:"id"`
	APIKey      string  `json:"apiKey"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Cash        float64 `json:"cash"`
}

// CreateAccountResponse holds the result of POST /users.
type CreateAccountResponse struct {
	User CreateAccountUser `json:"user"`
}

// CreateAccount creates a new paper trading account (no auth required).
// Returns the user with an API key and starting cash.
func CreateAccount(baseURL, name string) (*CreateAccountResponse, error) {
	body := map[string]interface{}{
		"name":        name,
		"description": "Auto-generated for NickAI CLI",
		"initialCash": 100000,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("POST", baseURL+"/users", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("account creation failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("account creation failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result CreateAccountResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
