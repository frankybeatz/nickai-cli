package ai

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ModelVerdict holds the result of a single model's analysis.
type ModelVerdict struct {
	Model      string        `json:"model"`
	Verdict    string        `json:"verdict"`              // BUY, SELL, HOLD
	Confidence string        `json:"confidence"`           // High, Medium, Low
	Reasoning  string        `json:"reasoning"`
	Error      string        `json:"error,omitempty"`
	Duration   time.Duration `json:"duration_ms"`
}

// ConsensusResult holds the aggregated result from all models.
type ConsensusResult struct {
	Verdicts  []ModelVerdict `json:"verdicts"`
	Consensus string         `json:"consensus"` // BUY, SELL, HOLD, NO_CONSENSUS
	Agreement string         `json:"agreement"` // e.g. "2/3", "3/3"
	Symbol    string         `json:"symbol"`
	Price     float64        `json:"price"`
}

// ConsensusConfig controls which models to query and the agreement threshold.
type ConsensusConfig struct {
	Models    []string `json:"models"`
	Threshold float64  `json:"threshold"` // e.g. 0.67 for 2/3
}

// Tier1Models are the frontier defaults — fast, high-quality consensus panel.
var Tier1Models = []string{
	"anthropic/claude-sonnet-4.6",
	"openai/gpt-5.2",
	"deepseek/deepseek-v3.2",
	"google/gemini-3-pro-preview",
}

// Tier2Models add diversity — different training data for contrarian signals.
var Tier2Models = []string{
	"x-ai/grok-4.1",
	"z-ai/glm-5",
	"alibaba/qwen3-235b",
	"google/gemini-3-flash-preview",
}

// Tier3Models are budget/free options for near-zero-cost consensus.
var Tier3Models = []string{
	"deepseek/deepseek-r1:free",
	"meta-llama/llama-3.3-70b-instruct",
}

// DefaultConsensusModels are queried when the user doesn't specify models.
// Uses Tier 1 (4 frontier models) for the best quality/speed tradeoff.
var DefaultConsensusModels = Tier1Models

// AllConsensusModels is the full set across all tiers.
var AllConsensusModels = append(append(append([]string{}, Tier1Models...), Tier2Models...), Tier3Models...)

// RunConsensus queries multiple LLMs in parallel for a BUY/SELL/HOLD verdict
// and returns the consensus result.
func RunConsensus(client *OpenRouterClient, config ConsensusConfig, symbol string, price float64, marketContext string) *ConsensusResult {
	systemPrompt := "You are a trading analyst. Given this market data for " + symbol +
		" at $" + fmt.Sprintf("%.2f", price) + ":\n\n" + marketContext + "\n\n" +
		`Respond with EXACTLY this JSON and nothing else:
{"verdict": "BUY or SELL or HOLD", "confidence": "High or Medium or Low", "reasoning": "1-2 sentences explaining why"}`

	userPrompt := fmt.Sprintf("Analyze %s at $%.2f and give your trading verdict.", symbol, price)

	models := config.Models
	if len(models) == 0 {
		models = DefaultConsensusModels
	}

	resultsCh := make(chan ModelVerdict, len(models))
	var wg sync.WaitGroup

	const perModelTimeout = 15 * time.Second

	for _, model := range models {
		wg.Add(1)
		go func(m string) {
			defer wg.Done()

			start := time.Now()

			// Per-model timeout: prevents one slow model from blocking results.
			type result struct {
				resp string
				err  error
			}
			done := make(chan result, 1)
			go func() {
				r, e := client.ChatCompletion(m, systemPrompt, userPrompt)
				done <- result{r, e}
			}()

			var resp string
			var err error
			select {
			case r := <-done:
				resp, err = r.resp, r.err
			case <-time.After(perModelTimeout):
				err = fmt.Errorf("timeout after %s", perModelTimeout)
			}

			elapsed := time.Since(start)

			if err != nil {
				resultsCh <- ModelVerdict{
					Model:    m,
					Error:    err.Error(),
					Duration: elapsed,
				}
				return
			}

			verdict, confidence, reasoning := parseVerdict(resp)
			resultsCh <- ModelVerdict{
				Model:      m,
				Verdict:    verdict,
				Confidence: confidence,
				Reasoning:  reasoning,
				Duration:   elapsed,
			}
		}(model)
	}

	// Close channel once all goroutines finish.
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	var verdicts []ModelVerdict
	for v := range resultsCh {
		verdicts = append(verdicts, v)
	}

	consensus, agreement := computeConsensus(verdicts, config.Threshold)

	return &ConsensusResult{
		Verdicts:  verdicts,
		Consensus: consensus,
		Agreement: agreement,
		Symbol:    symbol,
		Price:     price,
	}
}

// computeConsensus tallies verdicts and determines consensus.
// A verdict meets consensus if its count / total >= threshold.
func computeConsensus(verdicts []ModelVerdict, threshold float64) (consensus, agreement string) {
	counts := map[string]int{}
	total := 0

	for _, v := range verdicts {
		if v.Verdict == "" || v.Error != "" {
			continue
		}
		counts[v.Verdict]++
		total++
	}

	if total == 0 {
		return "NO_CONSENSUS", "0/0"
	}

	// Find the verdict with the highest count.
	bestVerdict := ""
	bestCount := 0
	for v, c := range counts {
		if c > bestCount {
			bestCount = c
			bestVerdict = v
		}
	}

	// Round to 2 decimal places to handle floating-point edge cases (e.g. 2/3 vs 0.67).
	ratio := math.Round(float64(bestCount)/float64(total)*100) / 100
	if ratio >= threshold {
		consensus = bestVerdict
	} else {
		consensus = "NO_CONSENSUS"
	}

	agreement = fmt.Sprintf("%d/%d", bestCount, total)
	return consensus, agreement
}

// parseVerdict extracts verdict, confidence, and reasoning from a model's
// response. It tries JSON parsing first, then falls back to regex extraction.
func parseVerdict(response string) (verdict, confidence, reasoning string) {
	// Try JSON parse first.
	var parsed struct {
		Verdict    string `json:"verdict"`
		Confidence string `json:"confidence"`
		Reasoning  string `json:"reasoning"`
	}

	// Try to find JSON within the response (models sometimes wrap it in markdown).
	jsonStr := response
	if idx := strings.Index(response, "{"); idx >= 0 {
		if end := strings.LastIndex(response, "}"); end > idx {
			jsonStr = response[idx : end+1]
		}
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
		verdict = normalizeVerdict(parsed.Verdict)
		confidence = normalizeConfidence(parsed.Confidence)
		reasoning = parsed.Reasoning
		if verdict != "" {
			return verdict, confidence, reasoning
		}
	}

	// Fallback: regex extraction.
	verdictRe := regexp.MustCompile(`(?i)\b(BUY|SELL|HOLD)\b`)
	if m := verdictRe.FindString(response); m != "" {
		verdict = strings.ToUpper(m)
	}

	confRe := regexp.MustCompile(`(?i)\b(High|Medium|Low)\b`)
	if m := confRe.FindString(response); m != "" {
		low := strings.ToLower(m)
		confidence = strings.ToUpper(low[:1]) + low[1:]
	}

	reasoning = "Extracted from unstructured response"

	return verdict, confidence, reasoning
}

// normalizeVerdict uppercases and validates a verdict string.
func normalizeVerdict(v string) string {
	upper := strings.ToUpper(strings.TrimSpace(v))
	switch upper {
	case "BUY", "SELL", "HOLD":
		return upper
	default:
		return ""
	}
}

// normalizeConfidence title-cases and validates a confidence string.
func normalizeConfidence(c string) string {
	lower := strings.ToLower(strings.TrimSpace(c))
	switch lower {
	case "high":
		return "High"
	case "medium":
		return "Medium"
	case "low":
		return "Low"
	default:
		return ""
	}
}
