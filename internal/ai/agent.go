package ai

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/tools"
)

const (
	anthropicURL     = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
	maxTokens        = 4096
	maxToolRounds    = 15
	maxRetries       = 3
	retryDelay       = 1 * time.Second

	// MiniMax (OpenAI-compatible, free tier).
	minimaxURL = "https://api.minimax.chat/v1/text/chatcompletion_v2"
)

// Provider identifies which LLM backend to use.
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderMiniMax   Provider = "minimax"
)

// ModelOption represents a selectable model.
type ModelOption struct {
	ID       string
	Name     string
	Provider Provider
	Free     bool
}

// AvailableModels lists all supported models.
var AvailableModels = []ModelOption{
	{ID: "claude-sonnet", Name: "Claude Sonnet 4", Provider: ProviderAnthropic, Free: false},
	{ID: "claude-haiku", Name: "Claude Haiku 4.5", Provider: ProviderAnthropic, Free: false},
	{ID: "minimax", Name: "MiniMax abab6.5s", Provider: ProviderMiniMax, Free: true},
}

// modelAPIName returns the actual API model string for a given model ID.
func modelAPIName(id string) string {
	switch id {
	case "claude-sonnet":
		return "claude-sonnet-4-20250514"
	case "claude-haiku":
		return "claude-haiku-4-5-20251001"
	case "minimax":
		return "abab6.5s-chat"
	default:
		return "claude-sonnet-4-20250514"
	}
}

const baseSystemPrompt = `You are Nick, an AI trading analyst on the NickAI platform. You have tools for live market data, portfolio management, trade execution, and strategy backtesting. When asked about markets or whether to buy/sell, ALWAYS use get_prices first to check current data. Be concise, data-driven, and actionable. Give specific price levels and reasoning. The user is paper trading on PaperNick with $100K starting capital.

When the user asks to backtest a strategy, use the backtest_strategy tool. Translate their natural language into structured conditions:
- Available indicators: rsi, macd, macd_histogram, macd_signal, bollinger_upper, bollinger_lower, sma20, sma50, ema12, ema26, price, fear_greed
- Available operators: < (less than), > (greater than), crosses_above, crosses_below
- Always include either exit conditions or stop_loss_pct/take_profit_pct
- Default period is 180d if not specified
- You can use preset strategies by name: rsi-reversal, macd-crossover, bollinger-bounce, golden-cross, momentum, fear-and-greed, dip-buyer
- After showing results, offer to tweak parameters and run again

When the user asks about Polymarket or prediction markets, use available MCP tools to:
1. Fetch current events and market odds
2. Search for relevant news and context
3. Compare implied probability (market odds) vs your assessed probability
4. Flag contracts where the gap is largest (potential mispricing)
Always note that prediction markets carry risk and past event analysis doesn't guarantee outcomes.

When the user shares preferences (risk tolerance, favorite symbols, position sizes) or when you observe patterns in their trading, use save_memory to remember for future sessions. Use recall_memory to search past context. After showing backtest results, offer to activate the strategy as a live monitoring rule using the activate_strategy tool.

Available commands the user can run:
Trading: /buy, /sell, /price, /orders, /pnl, /history
Analysis: /analyze, /chart, /backtest, /consensus, /risk, /analytics
Automation: /auto, /trigger, /alert, /watch
Multi-Vertical: /connect, /balances, /positions, /markets, /wallet, /swap, /gas, /stock, /screen, /bet, /odds, /lines, /funding
Memory: /memory, /strategy, /notify
Setup: /config, /mcp, /credential, /model, /theme

When answering questions, you can suggest relevant commands the user can run to take action.`

// mcpRegistryHint lists MCP servers the user can install for extra capabilities.
const mcpRegistryHint = `

You can suggest the user install MCP servers for capabilities you don't currently have. Available servers (install via /mcp add <name>):
- ccxt: Trade on 100+ crypto exchanges (needs API keys)
- alpaca: Stocks, ETFs, options, crypto on Alpaca (needs API keys)
- defillama: DeFi protocol data — TVL, yields, volumes (free, no keys)
- tradingview: Technical analysis, indicators, screeners (free, no keys)
- onchain: On-chain data — ERC20 tokens, transactions, contracts (free, no keys)
- web3: Multi-chain — Ethereum, Solana, Bitcoin (free, no keys)
- solana: 40+ Solana actions — tokens, DeFi, NFTs (needs RPC URL)
- jupiter: Solana DEX trades via Jupiter (needs private key)
- lifi: Cross-chain bridge and swap (free, no keys)
Quick install all free servers: /mcp quick`

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

// toolDef is the Anthropic API tool wire format, aliased from the tools package.
type toolDef = tools.AnthropicToolDef

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

// --- Streaming SSE types ---

type streamAPIRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	System    string        `json:"system"`
	Tools     []toolDef     `json:"tools"`
	Messages  []chatMessage `json:"messages"`
	Stream    bool          `json:"stream"`
}

type sseContentBlock struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Text string `json:"text,omitempty"`
}

type sseDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	StopReason  string `json:"stop_reason,omitempty"`
}

type sseEvent struct {
	Type         string           `json:"type"`
	Index        int              `json:"index"`
	ContentBlock *sseContentBlock `json:"content_block,omitempty"`
	Delta        *sseDelta        `json:"delta,omitempty"`
}

// --- Agent ---

// Agent manages conversation with an LLM and executes tools against PaperNick.
type Agent struct {
	client   *api.PapernickClient
	apiKey   string
	modelID  string
	provider Provider
	history  []chatMessage
	http     *http.Client
	registry *tools.Registry

	// MiniMax key (separate from Anthropic).
	minimaxKey string

	// Dynamic system prompt (includes MCP hints).
	systemPrompt string

	// Risk prompt suffix injected when risk limits are set.
	riskPromptSuffix string

	// Automation prompt suffix injected when automations exist.
	autoPromptSuffix string

	// Memory prompt suffix injected when memories exist.
	memoryPromptSuffix string

	// Portfolio context suffix (injected on boot and after trades).
	portfolioSuffix string

	// Recent activity suffix (last 3 commands the user ran).
	recentActivitySuffix string
}

// NewAgent creates an agent with the given PaperNick client, Anthropic API key,
// and tool registry.
func NewAgent(client *api.PapernickClient, anthropicKey string, registry *tools.Registry) *Agent {
	// Build system prompt: base + MCP tool sources + registry hints.
	prompt := baseSystemPrompt

	// Tell the LLM which MCP tools are currently connected.
	var mcpTools []string
	for _, entry := range registry.All() {
		if entry.Source != "builtin" {
			mcpTools = append(mcpTools, entry.Name+" (from "+entry.Source+")")
		}
	}
	if len(mcpTools) > 0 {
		prompt += "\n\nYou also have MCP tools connected: " + strings.Join(mcpTools, ", ") + "."
	}

	// Always include the registry hint so the LLM can suggest installs.
	prompt += mcpRegistryHint

	return &Agent{
		client:       client,
		apiKey:       anthropicKey,
		modelID:      "claude-sonnet",
		provider:     ProviderAnthropic,
		registry:     registry,
		systemPrompt: prompt,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
				TLSClientConfig:  &tls.Config{MaxVersion: tls.VersionTLS12},
			},
		},
	}
}

// SetModel switches the active model. Returns an error if the model needs an API key that isn't set.
func (a *Agent) SetModel(modelID string) error {
	for _, m := range AvailableModels {
		if m.ID == modelID {
			if m.Provider == ProviderAnthropic && a.apiKey == "" {
				return fmt.Errorf("Anthropic API key required. Set with /config set anthropic_key <key>")
			}
			if m.Provider == ProviderMiniMax && a.minimaxKey == "" {
				return fmt.Errorf("MiniMax API key required. Set with /config set minimax_key <key>")
			}
			a.modelID = modelID
			a.provider = m.Provider
			// Clear history on model switch to avoid format mismatches.
			a.history = nil
			return nil
		}
	}
	return fmt.Errorf("unknown model: %s", modelID)
}

// SetMinimaxKey sets the MiniMax API key.
func (a *Agent) SetMinimaxKey(key string) {
	a.minimaxKey = key
}

// ModelID returns the current model ID.
func (a *Agent) ModelID() string {
	return a.modelID
}

// UpdateClient refreshes the PaperNick client reference (after config change).
func (a *Agent) UpdateClient(client *api.PapernickClient) {
	a.client = client
}

// SetRiskInfo sets a risk limits suffix that gets appended to the system prompt.
// Call with empty string to clear.
func (a *Agent) SetRiskInfo(info string) {
	a.riskPromptSuffix = info
}

// SetAutoInfo sets an automation hint suffix for the system prompt.
func (a *Agent) SetAutoInfo(info string) {
	a.autoPromptSuffix = info
}

// SetMemoryInfo sets a memory context suffix for the system prompt.
func (a *Agent) SetMemoryInfo(info string) {
	a.memoryPromptSuffix = info
}

// SetPortfolioContext sets a portfolio summary suffix for the system prompt.
func (a *Agent) SetPortfolioContext(info string) {
	a.portfolioSuffix = info
}

// SetRecentActivity sets a recent-commands suffix for the system prompt.
func (a *Agent) SetRecentActivity(info string) {
	a.recentActivitySuffix = info
}

// effectivePrompt returns the system prompt with any dynamic suffixes.
func (a *Agent) effectivePrompt() string {
	p := a.systemPrompt
	if a.riskPromptSuffix != "" {
		p += "\n\n" + a.riskPromptSuffix
	}
	if a.autoPromptSuffix != "" {
		p += "\n\n" + a.autoPromptSuffix
	}
	if a.memoryPromptSuffix != "" {
		p += "\n\n" + a.memoryPromptSuffix
	}
	if a.portfolioSuffix != "" {
		p += "\n\n" + a.portfolioSuffix
	}
	if a.recentActivitySuffix != "" {
		p += "\n\n" + a.recentActivitySuffix
	}
	return p
}

// Chat sends a user message and runs the tool-use loop until the model
// produces a final text response. Routes to the correct provider.
func (a *Agent) Chat(userMessage string) (string, error) {
	switch a.provider {
	case ProviderMiniMax:
		return a.chatMiniMax(userMessage)
	default:
		return a.chatAnthropic(userMessage)
	}
}

// chatAnthropic runs the Anthropic ReAct tool-use loop.
func (a *Agent) chatAnthropic(userMessage string) (string, error) {
	a.history = append(a.history, chatMessage{
		Role:    "user",
		Content: userMessage,
	})

	for range maxToolRounds {
		resp, err := a.callAnthropic()
		if err != nil {
			return "", err
		}

		a.history = append(a.history, chatMessage{
			Role:    "assistant",
			Content: resp.Content,
		})

		var toolUses []contentBlock
		for _, block := range resp.Content {
			if block.Type == "tool_use" {
				toolUses = append(toolUses, block)
			}
		}

		if len(toolUses) == 0 {
			return extractText(resp.Content), nil
		}

		var results []toolResultBlock
		for _, tu := range toolUses {
			result := a.registry.ExecuteTool(tu.Name, tu.Input)
			results = append(results, toolResultBlock{
				Type:      "tool_result",
				ToolUseID: tu.ID,
				Content:   result,
			})
		}

		a.history = append(a.history, chatMessage{
			Role:    "user",
			Content: results,
		})
	}

	return "", fmt.Errorf("agent exceeded maximum tool rounds (%d)", maxToolRounds)
}

// chatMiniMax sends a simple chat completion to MiniMax (no tool use).
func (a *Agent) chatMiniMax(userMessage string) (string, error) {
	type mmMessage struct {
		Role    string `json:"sender_type"`
		Text    string `json:"text"`
	}
	type mmRequest struct {
		Model            string      `json:"model"`
		Messages         []mmMessage `json:"messages"`
		TokensToGenerate int         `json:"tokens_to_generate"`
		Prompt           string      `json:"prompt,omitempty"`
	}

	var messages []mmMessage
	messages = append(messages, mmMessage{Role: "USER", Text: a.effectivePrompt() + "\n\n" + userMessage})

	reqBody := mmRequest{
		Model:            modelAPIName(a.modelID),
		Messages:         messages,
		TokensToGenerate: maxTokens,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", minimaxURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.minimaxKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("MiniMax API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("MiniMax API %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Reply   string `json:"reply"`
		Choices []struct {
			Messages []struct {
				Text string `json:"text"`
			} `json:"messages"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to decode MiniMax response: %w", err)
	}

	if result.Reply != "" {
		return result.Reply, nil
	}
	if len(result.Choices) > 0 && len(result.Choices[0].Messages) > 0 {
		return result.Choices[0].Messages[0].Text, nil
	}

	return "", fmt.Errorf("empty response from MiniMax")
}

// callAnthropic sends the current conversation to the Anthropic API.
// Retries up to maxRetries times on transient errors (connection resets, 5xx).
func (a *Agent) callAnthropic() (*apiResponse, error) {
	a.sanitizeHistory()

	reqBody := apiRequest{
		Model:     modelAPIName(a.modelID),
		MaxTokens: maxTokens,
		System:    a.effectivePrompt(),
		Tools:     a.registry.ToAnthropicTools(),
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


// --- Streaming ---

// ChatStream sends a user message and streams text tokens through tokenCh.
// Returns the final complete response text. Tool-use rounds are handled
// internally. MiniMax falls back to non-streaming.
func (a *Agent) ChatStream(userMessage string, tokenCh chan<- string) (string, error) {
	if a.provider == ProviderMiniMax {
		resp, err := a.chatMiniMax(userMessage)
		if err != nil {
			return "", err
		}
		tokenCh <- resp
		return resp, nil
	}

	a.history = append(a.history, chatMessage{
		Role:    "user",
		Content: userMessage,
	})

	for range maxToolRounds {
		resp, err := a.callAnthropicStream(tokenCh)
		if err != nil {
			return "", err
		}

		a.history = append(a.history, chatMessage{
			Role:    "assistant",
			Content: resp.Content,
		})

		var toolUses []contentBlock
		for _, block := range resp.Content {
			if block.Type == "tool_use" {
				toolUses = append(toolUses, block)
			}
		}

		if len(toolUses) == 0 {
			return extractText(resp.Content), nil
		}

		var results []toolResultBlock
		for _, tu := range toolUses {
			result := a.registry.ExecuteTool(tu.Name, tu.Input)
			results = append(results, toolResultBlock{
				Type:      "tool_result",
				ToolUseID: tu.ID,
				Content:   result,
			})
		}

		a.history = append(a.history, chatMessage{
			Role:    "user",
			Content: results,
		})
	}

	return "", fmt.Errorf("agent exceeded maximum tool rounds (%d)", maxToolRounds)
}

// sanitizeHistory ensures every tool_use block in the conversation history
// has a non-nil Input field. The Anthropic API rejects requests where
// tool_use blocks lack the "input" field.
func (a *Agent) sanitizeHistory() {
	for i := range a.history {
		blocks, ok := a.history[i].Content.([]contentBlock)
		if !ok {
			continue
		}
		for j := range blocks {
			if blocks[j].Type == "tool_use" && len(blocks[j].Input) == 0 {
				blocks[j].Input = json.RawMessage("{}")
			}
		}
	}
}

// callAnthropicStream sends a streaming request to the Anthropic API.
// Text tokens are sent via tokenCh as they arrive. Returns the
// accumulated response (including any tool_use blocks).
func (a *Agent) callAnthropicStream(tokenCh chan<- string) (*apiResponse, error) {
	a.sanitizeHistory()

	reqBody := streamAPIRequest{
		Model:     modelAPIName(a.modelID),
		MaxTokens: maxTokens,
		System:    a.effectivePrompt(),
		Tools:     a.registry.ToAnthropicTools(),
		Messages:  a.history,
		Stream:    true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", anthropicURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")

	// Longer timeout for streaming (responses can be lengthy).
	streamClient := &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig:  &tls.Config{MaxVersion: tls.VersionTLS12},
		},
	}

	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		var errResp apiErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("Anthropic API %d: %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("Anthropic API %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse SSE stream.
	var blocks []contentBlock
	toolInputBufs := make(map[int]*strings.Builder)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event sseEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_start":
			if event.ContentBlock != nil {
				block := contentBlock{
					Type: event.ContentBlock.Type,
					ID:   event.ContentBlock.ID,
					Name: event.ContentBlock.Name,
				}
				if event.ContentBlock.Type == "tool_use" {
					// Default to empty object — tools with no params
					// may not send any input_json_delta events.
					block.Input = json.RawMessage("{}")
					toolInputBufs[event.Index] = &strings.Builder{}
				}
				blocks = append(blocks, block)
			}

		case "content_block_delta":
			if event.Delta != nil {
				switch event.Delta.Type {
				case "text_delta":
					if event.Delta.Text != "" && event.Index < len(blocks) {
						blocks[event.Index].Text += event.Delta.Text
						tokenCh <- event.Delta.Text
					}
				case "input_json_delta":
					if buf, ok := toolInputBufs[event.Index]; ok {
						buf.WriteString(event.Delta.PartialJSON)
					}
				}
			}

		case "content_block_stop":
			if buf, ok := toolInputBufs[event.Index]; ok {
				if event.Index < len(blocks) && buf.Len() > 0 {
					blocks[event.Index].Input = json.RawMessage(buf.String())
				}
				delete(toolInputBufs, event.Index)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("stream read error: %w", err)
	}

	return &apiResponse{
		Role:    "assistant",
		Content: blocks,
	}, nil
}
