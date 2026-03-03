package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

// OpenRouterClient talks to the OpenRouter API for multi-LLM access.
type OpenRouterClient struct {
	apiKey string
	http   *http.Client
}

// orRequest is the request body for OpenRouter chat completions.
type orRequest struct {
	Model       string      `json:"model"`
	Messages    []orMessage `json:"messages"`
	Temperature float64     `json:"temperature"`
}

// orMessage is a single message in the OpenRouter chat format.
type orMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// orResponse is the response body from OpenRouter chat completions.
type orResponse struct {
	Choices []struct {
		Message orMessage `json:"message"`
	} `json:"choices"`
}

// NewOpenRouterClient creates a new OpenRouter API client.
func NewOpenRouterClient(apiKey string) *OpenRouterClient {
	return &OpenRouterClient{
		apiKey: apiKey,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ChatCompletion sends a chat completion request to the specified model via
// OpenRouter. It returns the assistant's response text or an error.
// For context-aware calls (e.g. with timeout), use ChatCompletionContext.
func (c *OpenRouterClient) ChatCompletion(model, systemPrompt, userPrompt string) (string, error) {
	reqBody := orRequest{
		Model: model,
		Messages: []orMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", openRouterURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://nickai.dev")

	resp, err := c.http.Do(req)
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
