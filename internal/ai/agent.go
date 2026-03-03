package ai

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nickai/cli/internal/api"
	"github.com/nickai/cli/internal/logging"
	"github.com/nickai/cli/internal/personality"
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

// backoff returns an exponential delay with jitter for retry attempt n (1-indexed).
// 1s base → ~1-2s, ~2-4s, ~4-8s.
func backoff(attempt int) time.Duration {
	base := float64(retryDelay) * math.Pow(2, float64(attempt-1))
	jitter := base * (0.5 + rand.Float64()) // 50-150% of base
	return time.Duration(jitter)
}

// Provider identifies which LLM backend to use.
type Provider string

const (
	ProviderAnthropic   Provider = "anthropic"
	ProviderOpenRouter  Provider = "openrouter"
	ProviderMiniMax     Provider = "minimax"
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
	{ID: "claude-opus", Name: "Claude Opus 4.6", Provider: ProviderAnthropic, Free: false},
	{ID: "claude-sonnet", Name: "Claude Sonnet 4.6", Provider: ProviderAnthropic, Free: false},
	{ID: "claude-haiku", Name: "Claude Haiku 4.5", Provider: ProviderAnthropic, Free: false},
	{ID: "gpt-4o", Name: "GPT-4o", Provider: ProviderOpenRouter, Free: false},
	{ID: "gemini-flash", Name: "Gemini 2.0 Flash", Provider: ProviderOpenRouter, Free: false},
	{ID: "deepseek-v3", Name: "DeepSeek V3", Provider: ProviderOpenRouter, Free: false},
	{ID: "deepseek-r1", Name: "DeepSeek R1", Provider: ProviderOpenRouter, Free: true},
	{ID: "llama-3.3", Name: "Llama 3.3 70B", Provider: ProviderOpenRouter, Free: true},
	{ID: "minimax", Name: "MiniMax abab6.5s", Provider: ProviderMiniMax, Free: true},
}

// modelAPIName returns the actual API model string for a given model ID.
func modelAPIName(id string) string {
	switch id {
	case "claude-opus":
		return "claude-opus-4-6"
	case "claude-sonnet":
		return "claude-sonnet-4-6"
	case "claude-haiku":
		return "claude-haiku-4-5-20251001"
	case "gpt-4o":
		return "openai/gpt-4o"
	case "gemini-flash":
		return "google/gemini-2.0-flash-001"
	case "deepseek-v3":
		return "deepseek/deepseek-chat-v3-0324"
	case "deepseek-r1":
		return "deepseek/deepseek-r1:free"
	case "llama-3.3":
		return "meta-llama/llama-3.3-70b-instruct"
	case "minimax":
		return "abab6.5s-chat"
	default:
		// Custom OpenRouter slugs (contain "/") pass through as-is.
		if strings.Contains(id, "/") {
			return id
		}
		return "claude-sonnet-4-6"
	}
}

const systemPromptIntro = `You are Nick — a sharp, opinionated crypto trading assistant. You live for the markets and always back your calls with data.`

const systemPromptBody = `
## YOUR TOOLS
ALWAYS call get_prices before discussing any market. ALWAYS call get_portfolio before sizing recommendations. ALWAYS call analyze_market before giving TA. Data first, vibes second.

Tools you have:
- get_prices: Live prices for any crypto symbols. Call this FIRST on market questions.
- get_portfolio: User's cash, positions, total value. Call before any sizing advice.
- get_orders: Recent order history (filled, pending, cancelled).
- place_order: Execute a trade (market or limit). Requires user confirmation. Risk limits auto-checked.
- rebalance_portfolio: Calculate trades to hit target allocations. Returns a plan — call place_order for each trade.
- analyze_market: Full technical analysis — RSI(14), MACD(12/26/9), Bollinger(20), SMA 20/50, trend, Fear & Greed. Uses real Binance klines.
- backtest_strategy: Test strategies on historical data. Returns win rate, Sharpe, drawdown, trade list, equity curve.
- activate_strategy: Convert a backtest into a live monitoring rule (checked every 60s).
- create_automation: Set up automated rules — scheduled (daily DCA), conditional (BTC > 100K), portfolio-based (drawdown > 5%), or indicator-based.
- create_twap: Scale into/out over time. Auto-slices with confirmations.
- get_analytics: Portfolio Sharpe, max drawdown, win rate, profit factor, allocation.
- get_trade_journal: Full trade log with rationale, win rate, P&L.
- save_memory: Remember user preferences, patterns, insights across sessions.
- recall_memory: Search past memories by keyword.

## PLATFORM CONTEXT
Paper trading on PaperNick with $100K starting capital. All trades are simulated — zero real risk.

## BACKTESTING
Use backtest_strategy. Indicators: rsi, macd, macd_histogram, macd_signal, bollinger_upper, bollinger_lower, sma20, sma50, ema12, ema26, price, fear_greed. Operators: <, >, crosses_above, crosses_below. Always include exit conditions or stop_loss/take_profit. Default period: 180d. Presets: rsi-reversal, macd-crossover, bollinger-bounce, golden-cross, momentum, fear-and-greed, dip-buyer. After results, offer to tweak params and rerun, or activate_strategy to go live.

## PREDICTION MARKETS
For Polymarket: fetch events/odds via MCP tools, compare implied probability vs your assessed probability, flag contracts where the gap is largest (potential alpha). Always note that prediction markets carry risk.

## MEMORY
Use save_memory when you learn something about the user: their risk tolerance, favorite assets, position sizing style, trade history patterns. Use recall_memory to check if you've seen them before. This makes you better every session.

## ALL COMMANDS THE USER CAN RUN
Guide them to the right command at the right time:

Trading: /buy <sym> <qty> [limit <price>], /sell <sym> <qty>, /price <sym...>, /orders, /pnl, /history
Analysis: /analyze <sym>, /chart <sym>, /backtest [presets|run <preset> <sym>], /consensus [all|budget] <sym>, /analytics
Automation: /auto list, /trigger add <sym> < > <price> buy|sell <qty>, /alert <sym> > < <price>, /strategy twap <sym> buy|sell $<val> <dur>, /watch <sym...>
Risk: /risk set max-order|max-position|daily-loss <val>, /notify set desktop|webhook
Markets: /market, /snapshot, /dashboard, /funding <sym>
Multi-vertical: /stock <ticker>, /screen <filters>, /polymarket scan|analyze, /bet, /odds, /lines, /wallet, /swap, /gas, /balances, /positions
Setup: /config show|set|test, /mcp list|search|add|quick, /connect <exchange>, /credential add|list, /model, /theme, /vibe
Memory: /memory, /guide, /man <cmd>

## WORKFLOW AWARENESS
Know what comes next:
- After /price: suggest /analyze or /buy
- After /analyze: suggest /backtest to validate or /buy to act
- After /backtest with good results: suggest /backtest activate to go live
- After /buy or /sell: suggest /orders to verify, /pnl to track
- After /pnl: suggest /analytics for deeper metrics
- If they have cash sitting: suggest scanning for entries
- If they have no risk limits: nudge /risk after their first trade
- If they have no MCP tools: suggest /mcp quick for free market data
- If they haven't tried /consensus: mention it when they ask for an opinion
- If they haven't backtested: suggest it when discussing strategies`

// mcpRegistryHint lists MCP servers the user can install for extra capabilities.
const mcpRegistryHint = `

## MCP SERVERS (suggest /mcp add <name> when relevant)
Free (no keys): defillama (DeFi TVL/yields), tradingview (TA/screeners), onchain (ERC20/txns), hyperliquid (perp data), polymarket (prediction markets)
Paid (needs keys): ccxt (100+ exchanges), alpaca (stocks/ETFs/options), binance (dedicated), solana (40+ actions), jupiter (Solana DEX), coinmarketcap (50+ tools), brave-search (web/news), evm (30+ chains)
Quick install all free: /mcp quick`

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
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	System      string        `json:"system"`
	Tools       []toolDef     `json:"tools"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
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
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	System      string        `json:"system"`
	Tools       []toolDef     `json:"tools"`
	Messages    []chatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature"`
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

	// OpenRouter key (separate from Anthropic).
	openrouterKey string

	// Active vibe ID.
	vibeID string

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

	// Guidance context suffix (user journey stage).
	guidanceSuffix string
}

// NewAgent creates an agent with the given PaperNick client, Anthropic API key,
// tool registry, and vibe ID.
func NewAgent(client *api.PapernickClient, anthropicKey string, registry *tools.Registry, vibeID string) *Agent {
	a := &Agent{
		client:   client,
		apiKey:   anthropicKey,
		modelID:  "claude-sonnet",
		provider: ProviderAnthropic,
		registry: registry,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
				TLSClientConfig:  &tls.Config{MinVersion: tls.VersionTLS12},
			},
		},
	}
	a.SetVibe(vibeID)
	return a
}

// SetVibe switches the agent's personality and rebuilds the system prompt.
func (a *Agent) SetVibe(vibeID string) {
	vibe := personality.GetVibe(vibeID)
	a.vibeID = vibe.ID

	// Build system prompt: intro + vibe voice + body + MCP tools + registry hints.
	prompt := systemPromptIntro + "\n\n" + vibe.Prompt + "\n" + systemPromptBody

	// Tell the LLM which MCP tools are currently connected.
	var mcpTools []string
	for _, entry := range a.registry.All() {
		if entry.Source != "builtin" {
			mcpTools = append(mcpTools, entry.Name+" (from "+entry.Source+")")
		}
	}
	if len(mcpTools) > 0 {
		prompt += "\n\nYou also have MCP tools connected: " + strings.Join(mcpTools, ", ") + "."
	}

	// Always include the registry hint so the LLM can suggest installs.
	prompt += mcpRegistryHint
	a.systemPrompt = prompt
}

// VibeID returns the current vibe ID.
func (a *Agent) VibeID() string {
	return a.vibeID
}

// SetModel switches the active model. Returns an error if the model needs an API key that isn't set.
func (a *Agent) SetModel(modelID string) error {
	for _, m := range AvailableModels {
		if m.ID == modelID {
			if m.Provider == ProviderAnthropic && a.apiKey == "" {
				return fmt.Errorf("Anthropic API key required. Set with /config set anthropic_key <key>")
			}
			if m.Provider == ProviderOpenRouter && a.openrouterKey == "" {
				return fmt.Errorf("OpenRouter API key required. Set with /config set openrouter_key <key>")
			}
			if m.Provider == ProviderMiniMax && a.minimaxKey == "" {
				return fmt.Errorf("MiniMax API key required. Set with /config set minimax_key <key>")
			}
			a.modelID = modelID
			a.provider = m.Provider
			// Clear history on model switch to avoid format mismatches.
			a.history = nil
			logging.Info("model switched", "model", modelID, "provider", string(m.Provider))
			return nil
		}
	}

	// Accept custom OpenRouter model slugs (contain "/").
	if strings.Contains(modelID, "/") {
		if a.openrouterKey == "" {
			return fmt.Errorf("OpenRouter API key required. Set with /config set openrouter_key <key>")
		}
		a.modelID = modelID
		a.provider = ProviderOpenRouter
		a.history = nil
		logging.Info("model switched", "model", modelID, "provider", "openrouter")
		return nil
	}

	return fmt.Errorf("unknown model: %s (tip: use a slug like openai/gpt-4o-mini for custom OpenRouter models)", modelID)
}

// SetMinimaxKey sets the MiniMax API key.
func (a *Agent) SetMinimaxKey(key string) {
	a.minimaxKey = key
}

// SetOpenRouterKey sets the OpenRouter API key.
func (a *Agent) SetOpenRouterKey(key string) {
	a.openrouterKey = key
}

// ModelID returns the current model ID.
func (a *Agent) ModelID() string {
	return a.modelID
}

// Provider returns the current provider.
func (a *Agent) Provider() Provider {
	return a.provider
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

// SetGuidanceContext sets the user journey stage context for the system prompt.
func (a *Agent) SetGuidanceContext(info string) {
	a.guidanceSuffix = info
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
	if a.guidanceSuffix != "" {
		p += "\n\n" + a.guidanceSuffix
	}
	return p
}

// pruneHistory drops oldest message pairs when history grows too large.
// Keeps the most recent turns to stay within context window limits.
// Estimates ~4 chars per token, and keeps history under 80% of a 200K window.
func (a *Agent) pruneHistory() {
	const maxHistoryChars = 200000 * 4 * 80 / 100 // ~80% of 200K context in chars
	const minKeep = 4                               // always keep last 4 messages

	if len(a.history) <= minKeep {
		return
	}

	total := 0
	for _, msg := range a.history {
		switch v := msg.Content.(type) {
		case string:
			total += len(v)
		case []contentBlock:
			for _, b := range v {
				total += len(b.Text) + len(b.Input)
			}
		case []toolResultBlock:
			for _, r := range v {
				total += len(r.Content)
			}
		}
	}

	for total > maxHistoryChars && len(a.history) > minKeep {
		msg := a.history[0]
		switch v := msg.Content.(type) {
		case string:
			total -= len(v)
		case []contentBlock:
			for _, b := range v {
				total -= len(b.Text) + len(b.Input)
			}
		case []toolResultBlock:
			for _, r := range v {
				total -= len(r.Content)
			}
		}
		a.history = a.history[1:]
	}
}

// Chat sends a user message and runs the tool-use loop until the model
// produces a final text response. Routes to the correct provider.
func (a *Agent) Chat(ctx context.Context, userMessage string) (string, error) {
	switch a.provider {
	case ProviderOpenRouter:
		return a.chatOpenRouter(ctx, userMessage)
	case ProviderMiniMax:
		return a.chatMiniMax(ctx, userMessage)
	default:
		return a.chatAnthropic(ctx, userMessage)
	}
}

// chatAnthropic runs the Anthropic ReAct tool-use loop.
func (a *Agent) chatAnthropic(ctx context.Context, userMessage string) (string, error) {
	a.history = append(a.history, chatMessage{
		Role:    "user",
		Content: userMessage,
	})
	a.pruneHistory()

	for range maxToolRounds {
		resp, err := a.callAnthropic(ctx)
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
func (a *Agent) chatMiniMax(ctx context.Context, userMessage string) (string, error) {
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

	req, err := http.NewRequestWithContext(ctx, "POST", minimaxURL, bytes.NewReader(body))
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

// chatOpenRouter sends a simple chat completion to OpenRouter (no tool use).
func (a *Agent) chatOpenRouter(ctx context.Context, userMessage string) (string, error) {
	reqBody := orRequest{
		Model: modelAPIName(a.modelID),
		Messages: []orMessage{
			{Role: "system", Content: a.effectivePrompt()},
			{Role: "user", Content: userMessage},
		},
		Temperature: 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openRouterURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.openrouterKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://nickai.dev")

	resp, err := a.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("OpenRouter API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenRouter API %d: %s", resp.StatusCode, string(respBody))
	}

	var result orResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to decode OpenRouter response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenRouter (no choices)")
	}

	return result.Choices[0].Message.Content, nil
}

// chatOpenRouterStream sends a streaming chat completion to OpenRouter.
// Content deltas are forwarded through tokenCh as they arrive.
func (a *Agent) chatOpenRouterStream(ctx context.Context, userMessage string, tokenCh chan<- string) (string, error) {
	client := NewOpenRouterClient(a.openrouterKey)
	return client.ChatCompletionStream(ctx, modelAPIName(a.modelID), a.effectivePrompt(), userMessage, func(chunk string) {
		tokenCh <- chunk
	})
}

// callAnthropic sends the current conversation to the Anthropic API.
// Retries up to maxRetries times on transient errors (connection resets, 5xx).
func (a *Agent) callAnthropic(ctx context.Context) (*apiResponse, error) {
	a.sanitizeHistory()

	reqBody := apiRequest{
		Model:       modelAPIName(a.modelID),
		MaxTokens:   maxTokens,
		System:      a.effectivePrompt(),
		Tools:       a.registry.ToAnthropicTools(),
		Messages:    a.history,
		Temperature: 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			wait := backoff(attempt)
			logging.Debug("anthropic retry", "attempt", attempt+1, "model", a.modelID, "backoff_ms", wait.Milliseconds())
			time.Sleep(wait)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", anthropicURL, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("x-api-key", a.apiKey)
		req.Header.Set("anthropic-version", anthropicVersion)
		req.Header.Set("content-type", "application/json")

		resp, err := a.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastErr = fmt.Errorf("API request failed: %w", err)
			logging.Warn("anthropic request error", "error", err, "attempt", attempt+1)
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
			logging.Warn("anthropic server error", "status", resp.StatusCode, "attempt", attempt+1)
			continue
		}

		// Retry on rate limit (429) with Retry-After header.
		if resp.StatusCode == 429 {
			wait := retryDelay
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, parseErr := strconv.Atoi(ra); parseErr == nil && secs > 0 {
					wait = time.Duration(secs) * time.Second
				}
			}
			lastErr = fmt.Errorf("Anthropic API 429: rate limited")
			time.Sleep(wait)
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
func (a *Agent) ChatStream(ctx context.Context, userMessage string, tokenCh chan<- string) (string, error) {
	if a.provider == ProviderOpenRouter {
		return a.chatOpenRouterStream(ctx, userMessage, tokenCh)
	}
	if a.provider == ProviderMiniMax {
		resp, err := a.chatMiniMax(ctx, userMessage)
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
	a.pruneHistory()

	for range maxToolRounds {
		resp, err := a.callAnthropicStream(ctx, tokenCh)
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

// callAnthropicStream sends a streaming request to the Anthropic API with
// retry logic matching callAnthropic. Text tokens are sent via tokenCh as
// they arrive. Returns the accumulated response (including any tool_use blocks).
func (a *Agent) callAnthropicStream(ctx context.Context, tokenCh chan<- string) (*apiResponse, error) {
	a.sanitizeHistory()

	reqBody := streamAPIRequest{
		Model:       modelAPIName(a.modelID),
		MaxTokens:   maxTokens,
		System:      a.effectivePrompt(),
		Tools:       a.registry.ToAnthropicTools(),
		Messages:    a.history,
		Stream:      true,
		Temperature: 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Longer timeout for streaming (responses can be lengthy).
	streamClient := &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig:  &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}

	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			wait := backoff(attempt)
			logging.Debug("anthropic stream retry", "attempt", attempt+1, "backoff_ms", wait.Milliseconds())
			time.Sleep(wait)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", anthropicURL, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("x-api-key", a.apiKey)
		req.Header.Set("anthropic-version", anthropicVersion)
		req.Header.Set("content-type", "application/json")

		resp, err := streamClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastErr = fmt.Errorf("stream request failed: %w", err)
			continue
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("Anthropic API %d (stream)", resp.StatusCode)
			continue
		}

		if resp.StatusCode == 429 {
			wait := retryDelay
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, parseErr := strconv.Atoi(ra); parseErr == nil && secs > 0 {
					wait = time.Duration(secs) * time.Second
				}
			}
			resp.Body.Close()
			lastErr = fmt.Errorf("Anthropic API 429: rate limited (stream)")
			time.Sleep(wait)
			continue
		}

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			var errResp apiErrorResponse
			if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error.Message != "" {
				return nil, fmt.Errorf("Anthropic API %d: %s", resp.StatusCode, errResp.Error.Message)
			}
			return nil, fmt.Errorf("Anthropic API %d: %s", resp.StatusCode, string(respBody))
		}

		// Successful connection — parse the SSE stream.
		apiResp, err := a.parseSSEStream(resp, tokenCh)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		return apiResp, nil
	}

	return nil, lastErr
}

// parseSSEStream reads an SSE response from the Anthropic streaming API.
func (a *Agent) parseSSEStream(resp *http.Response, tokenCh chan<- string) (*apiResponse, error) {
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
