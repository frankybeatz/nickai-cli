package ai

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	Stream      bool        `json:"stream,omitempty"`
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

// orStreamDelta represents a delta in a streaming chunk.
type orStreamDelta struct {
	Content string `json:"content"`
}

// orStreamChoice represents a single choice in a streaming chunk.
type orStreamChoice struct {
	Delta        orStreamDelta `json:"delta"`
	FinishReason *string       `json:"finish_reason"`
}

// orStreamChunk represents one SSE data payload from the OpenRouter streaming API.
type orStreamChunk struct {
	Choices []orStreamChoice `json:"choices"`
}

// ChatCompletionStream sends a streaming chat completion request to OpenRouter.
// It calls onChunk for each content delta as it arrives, and returns the full
// accumulated response text when the stream completes.
func (c *OpenRouterClient) ChatCompletionStream(ctx context.Context, model, systemPrompt, userPrompt string, onChunk func(chunk string)) (string, error) {
	reqBody := orRequest{
		Model: model,
		Messages: []orMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		Stream:      true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use a longer timeout for streaming responses.
	streamClient := &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig:  &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openRouterURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://nickai.dev")

	resp, err := streamClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("OpenRouter stream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenRouter API %d: %s", resp.StatusCode, string(respBody))
	}

	// Read SSE events from the response body.
	var full strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// SSE lines that carry data start with "data: ".
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		// The stream ends with a sentinel value.
		if data == "[DONE]" {
			break
		}

		var chunk orStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // skip malformed chunks
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				full.WriteString(choice.Delta.Content)
				if onChunk != nil {
					onChunk(choice.Delta.Content)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return full.String(), fmt.Errorf("stream read error: %w", err)
	}

	return full.String(), nil
}
