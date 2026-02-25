package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nickai/cli/internal/api"
)

const (
	anthropicURL     = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
	model            = "claude-sonnet-4-20250514"
	maxTokens        = 4096
	maxToolRounds    = 10
	maxRetries       = 3
	retryDelay       = 1 * time.Second
)

const systemPrompt = `You are Nick, an AI trading analyst on the NickAI platform. You have tools for live market data, portfolio management, and trade execution. When asked about markets or whether to buy/sell, ALWAYS use get_prices first to check current data. Be concise, data-driven, and actionable. Give specific price levels and reasoning. The user is paper trading on PaperNick with $100K starting capital.`

// --- Anthropic API types ---

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []contentBlock or []toolResultBlock
}

type contentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type toolResultBlock struct {
	Type      string `json:"type"` // always "tool_result"
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

type toolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type apiRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	System    string        `json:"system"`
	Tools     []toolDef     `json:"tools"`
	Messages  []chatMessage `json:"messages"`
}

type apiResponse struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
}

type apiErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// --- Tool input types ---

type getPricesInput struct {
	Symbols []string `json:"symbols"`
}

type placeOrderInput struct {
	Symbol   string  `json:"symbol"`
	Side     string  `json:"side"`
	Quantity float64 `json:"quantity"`
	Type     string  `json:"type"`
	Price    float64 `json:"price,omitempty"`
}

// --- Tool definitions ---

var tools = []toolDef{
	{
		Name:        "get_prices",
		Description: "Get current prices for cryptocurrency symbols. Returns symbol and current price.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"symbols": {
					"type": "array",
					"items": {"type": "string"},
					"description": "List of symbol tickers, e.g. [\"BTC\", \"ETH\", \"SOL\"]"
				}
			},
			"required": ["symbols"]
		}`),
	},
	{
		Name:        "get_portfolio",
		Description: "Get the user's current portfolio including cash balance, total value, and all open positions with quantities and values.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {},
			"required": []
		}`),
	},
	{
		Name:        "get_orders",
		Description: "Get the user's recent order history including filled, pending, and cancelled orders.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {},
			"required": []
		}`),
	},
	{
		Name:        "place_order",
		Description: "Place a trade order. Symbol should include quote currency (e.g. BTCUSDT). For market orders, omit price. For limit orders, include price.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"symbol": {
					"type": "string",
					"description": "Trading pair symbol, e.g. BTCUSDT, ETHUSDT"
				},
				"side": {
					"type": "string",
					"enum": ["buy", "sell"],
					"description": "Order side"
				},
				"quantity": {
					"type": "number",
					"description": "Quantity to trade"
				},
				"type": {
					"type": "string",
					"enum": ["market", "limit"],
					"description": "Order type"
				},
				"price": {
					"type": "number",
					"description": "Limit price (required for limit orders, omit for market)"
				}
			},
			"required": ["symbol", "side", "quantity", "type"]
		}`),
	},
}

// --- Agent ---

// Agent manages conversation with Anthropic and executes tools against PaperNick.
type Agent struct {
	client  *api.PapernickClient
	apiKey  string
	history []chatMessage
	http    *http.Client
}

// NewAgent creates an agent with the given PaperNick client and Anthropic API key.
func NewAgent(client *api.PapernickClient, anthropicKey string) *Agent {
	return &Agent{
		client: client,
		apiKey: anthropicKey,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
	}
}

// UpdateClient refreshes the PaperNick client reference (after config change).
func (a *Agent) UpdateClient(client *api.PapernickClient) {
	a.client = client
}

// Chat sends a user message and runs the ReAct tool-use loop until the model
// produces a final text response. Returns the concatenated text output.
func (a *Agent) Chat(userMessage string) (string, error) {
	// Append user message to history.
	a.history = append(a.history, chatMessage{
		Role:    "user",
		Content: userMessage,
	})

	for range maxToolRounds {
		resp, err := a.callAnthropic()
		if err != nil {
			return "", err
		}

		// Append assistant response to history (preserve full content for multi-turn).
		a.history = append(a.history, chatMessage{
			Role:    "assistant",
			Content: resp.Content,
		})

		// Collect tool_use blocks.
		var toolUses []contentBlock
		for _, block := range resp.Content {
			if block.Type == "tool_use" {
				toolUses = append(toolUses, block)
			}
		}

		// No tool calls — extract text and return.
		if len(toolUses) == 0 {
			return extractText(resp.Content), nil
		}

		// Execute each tool and build results.
		var results []toolResultBlock
		for _, tu := range toolUses {
			result := a.executeTool(tu.Name, tu.Input)
			results = append(results, toolResultBlock{
				Type:      "tool_result",
				ToolUseID: tu.ID,
				Content:   result,
			})
		}

		// Append tool results as a user message and loop.
		a.history = append(a.history, chatMessage{
			Role:    "user",
			Content: results,
		})
	}

	return "", fmt.Errorf("agent exceeded maximum tool rounds (%d)", maxToolRounds)
}

// callAnthropic sends the current conversation to the Anthropic API.
// Retries up to maxRetries times on transient errors (connection resets, 5xx).
func (a *Agent) callAnthropic() (*apiResponse, error) {
	reqBody := apiRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Tools:     tools,
		Messages:  a.history,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			time.Sleep(retryDelay)
		}

		req, err := http.NewRequest("POST", anthropicURL, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("x-api-key", a.apiKey)
		req.Header.Set("anthropic-version", anthropicVersion)
		req.Header.Set("content-type", "application/json")

		resp, err := a.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("API request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		// Retry on server errors (5xx).
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("Anthropic API %d: %s", resp.StatusCode, string(respBody))
			continue
		}

		// Client errors (4xx) are not retryable.
		if resp.StatusCode >= 400 {
			var errResp apiErrorResponse
			if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error.Message != "" {
				return nil, fmt.Errorf("Anthropic API %d: %s", resp.StatusCode, errResp.Error.Message)
			}
			return nil, fmt.Errorf("Anthropic API %d: %s", resp.StatusCode, string(respBody))
		}

		var apiResp apiResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return &apiResp, nil
	}

	return nil, lastErr
}

// executeTool dispatches a tool call to the PaperNick client and returns JSON.
func (a *Agent) executeTool(name string, rawInput json.RawMessage) string {
	switch name {
	case "get_prices":
		var input getPricesInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return errorJSON("invalid input: " + err.Error())
		}
		prices, err := a.client.GetPrices(input.Symbols)
		if err != nil {
			return errorJSON(err.Error())
		}
		return toJSON(prices)

	case "get_portfolio":
		portfolio, err := a.client.GetPortfolio()
		if err != nil {
			return errorJSON(err.Error())
		}
		return toJSON(portfolio)

	case "get_orders":
		orders, err := a.client.GetOrders()
		if err != nil {
			return errorJSON(err.Error())
		}
		return toJSON(orders)

	case "place_order":
		var input placeOrderInput
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return errorJSON("invalid input: " + err.Error())
		}
		// Normalize symbol — append USDT if no quote currency.
		symbol := strings.ToUpper(input.Symbol)
		if !strings.HasSuffix(symbol, "USDT") &&
			!strings.HasSuffix(symbol, "USDC") &&
			!strings.HasSuffix(symbol, "USD") {
			symbol += "USDT"
		}
		order, err := a.client.PlaceOrder(api.PlaceOrderRequest{
			Symbol:   symbol,
			Side:     input.Side,
			Quantity: input.Quantity,
			Type:     input.Type,
			Price:    input.Price,
		})
		if err != nil {
			return errorJSON(err.Error())
		}
		return toJSON(order)

	default:
		return errorJSON("unknown tool: " + name)
	}
}

// extractText concatenates all text blocks from a response.
func extractText(blocks []contentBlock) string {
	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func toJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return errorJSON(err.Error())
	}
	return string(data)
}

func errorJSON(msg string) string {
	return fmt.Sprintf(`{"error": %q}`, msg)
}
