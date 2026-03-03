package sanitize

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// roleMarkerRe matches common LLM role markers at the start of a line.
// Catches "system:", "user:", "assistant:", "human:", "AI:" with optional whitespace.
var roleMarkerRe = regexp.MustCompile(`(?im)^(system|user|assistant|human|AI)\s*:`)

// toolTagRe matches XML-like tool tags that could be interpreted as tool markers.
// Catches <tool_use>, </tool_use>, <tool_result>, </tool_result>, <function_call>, etc.
var toolTagRe = regexp.MustCompile(`(?i)</?(?:tool_use|tool_result|function_call|function_result|tool_code|tool_output|system|thinking)[^>]*>`)

// SanitizeForPrompt strips LLM role markers and tool markers from text
// that will be included in system prompts. Prevents prompt injection.
func SanitizeForPrompt(s string) string {
	s = roleMarkerRe.ReplaceAllStringFunc(s, func(match string) string {
		// Replace the colon-separated marker but keep the word as content.
		// "system: do X" -> "system - do X"
		idx := strings.Index(match, ":")
		if idx >= 0 {
			return match[:idx] + " -"
		}
		return match
	})
	s = toolTagRe.ReplaceAllString(s, "")
	return s
}

// SanitizeMCPResult strips control characters and truncates MCP tool results
// before feeding them to the LLM.
func SanitizeMCPResult(s string, maxBytes int) string {
	// Strip control characters except \n (0x0A) and \t (0x09).
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			continue
		}
		b.WriteRune(r)
	}
	cleaned := b.String()

	if maxBytes > 0 && len(cleaned) > maxBytes {
		cleaned = cleaned[:maxBytes] + "... [truncated]"
	}
	return cleaned
}

// ValidateSymbol validates a trading symbol (e.g. "BTC", "ETH-USD").
// Returns cleaned uppercase symbol or error if invalid.
func ValidateSymbol(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("symbol cannot be empty")
	}
	if len(s) > 20 {
		return "", fmt.Errorf("symbol too long (max 20 chars): %q", s)
	}
	for _, r := range s {
		if !isSymbolChar(r) {
			return "", fmt.Errorf("invalid character %q in symbol %q", string(r), s)
		}
	}
	return strings.ToUpper(s), nil
}

// isSymbolChar returns true for characters allowed in trading symbols:
// alphanumeric, dash, forward slash.
func isSymbolChar(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '/'
}
