package ai

import (
	"testing"

	"github.com/nickai/cli/internal/tools"
)

func TestAgent_MissingAnthropicKey(t *testing.T) {
	reg := tools.NewRegistry()
	agent := NewAgent(nil, "", reg, "")

	// Anthropic model requires an API key.
	err := agent.SetModel("claude-sonnet")
	if err == nil {
		t.Error("expected error for missing Anthropic API key")
	}
	if err != nil && !containsStr(err.Error(), "Anthropic API key required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestAgent_MissingOpenRouterKey(t *testing.T) {
	reg := tools.NewRegistry()
	agent := NewAgent(nil, "sk-ant-test", reg, "")

	// OpenRouter model requires its own key.
	err := agent.SetModel("gpt-4o")
	if err == nil {
		t.Error("expected error for missing OpenRouter API key")
	}
	if err != nil && !containsStr(err.Error(), "OpenRouter API key required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestAgent_UnknownModel(t *testing.T) {
	reg := tools.NewRegistry()
	agent := NewAgent(nil, "sk-ant-test", reg, "")

	err := agent.SetModel("totally-fake-model")
	if err == nil {
		t.Error("expected error for unknown model")
	}
	if err != nil && !containsStr(err.Error(), "unknown model") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestAgent_ValidModelSwitch(t *testing.T) {
	reg := tools.NewRegistry()
	agent := NewAgent(nil, "sk-ant-test", reg, "")
	agent.SetOpenRouterKey("or-test-key")
	agent.SetMinimaxKey("mm-test-key")

	// Switch to various valid models.
	tests := []struct {
		modelID  string
		wantProv Provider
	}{
		{"claude-sonnet", ProviderAnthropic},
		{"claude-haiku", ProviderAnthropic},
		{"claude-opus", ProviderAnthropic},
		{"gpt-4o", ProviderOpenRouter},
		{"deepseek-v3", ProviderOpenRouter},
		{"deepseek-r1", ProviderOpenRouter},
		{"llama-3.3", ProviderOpenRouter},
		{"minimax", ProviderMiniMax},
	}

	for _, tc := range tests {
		err := agent.SetModel(tc.modelID)
		if err != nil {
			t.Errorf("SetModel(%q) unexpected error: %v", tc.modelID, err)
			continue
		}
		if agent.ModelID() != tc.modelID {
			t.Errorf("ModelID() = %q, want %q", agent.ModelID(), tc.modelID)
		}
		if agent.Provider() != tc.wantProv {
			t.Errorf("Provider() = %q, want %q for model %q", agent.Provider(), tc.wantProv, tc.modelID)
		}
	}
}

func TestAgent_ModelAPIName(t *testing.T) {
	// Verify each known model ID maps to the correct API name.
	tests := []struct {
		id   string
		want string
	}{
		{"claude-opus", "claude-opus-4-6"},
		{"claude-sonnet", "claude-sonnet-4-6"},
		{"claude-haiku", "claude-haiku-4-5-20251001"},
		{"gpt-4o", "openai/gpt-4o"},
		{"gemini-flash", "google/gemini-2.0-flash-001"},
		{"deepseek-v3", "deepseek/deepseek-chat-v3-0324"},
		{"deepseek-r1", "deepseek/deepseek-r1:free"},
		{"llama-3.3", "meta-llama/llama-3.3-70b-instruct"},
		{"minimax", "abab6.5s-chat"},
	}

	for _, tc := range tests {
		got := modelAPIName(tc.id)
		if got != tc.want {
			t.Errorf("modelAPIName(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}

	// Unknown model without "/" falls back to claude-sonnet-4-6.
	got := modelAPIName("nonexistent")
	if got != "claude-sonnet-4-6" {
		t.Errorf("modelAPIName(nonexistent) = %q, want %q", got, "claude-sonnet-4-6")
	}

	// Custom OpenRouter slugs with "/" pass through.
	custom := "openai/gpt-4o-mini"
	got = modelAPIName(custom)
	if got != custom {
		t.Errorf("modelAPIName(%q) = %q, want %q (passthrough)", custom, got, custom)
	}
}

func TestAgent_ProviderDetection(t *testing.T) {
	// Verify AvailableModels has the correct provider for each model.
	providerMap := make(map[string]Provider)
	for _, m := range AvailableModels {
		providerMap[m.ID] = m.Provider
	}

	expected := map[string]Provider{
		"claude-opus":   ProviderAnthropic,
		"claude-sonnet": ProviderAnthropic,
		"claude-haiku":  ProviderAnthropic,
		"gpt-4o":        ProviderOpenRouter,
		"gemini-flash":  ProviderOpenRouter,
		"deepseek-v3":   ProviderOpenRouter,
		"deepseek-r1":   ProviderOpenRouter,
		"llama-3.3":     ProviderOpenRouter,
		"minimax":       ProviderMiniMax,
	}

	for id, wantProv := range expected {
		gotProv, ok := providerMap[id]
		if !ok {
			t.Errorf("model %q not found in AvailableModels", id)
			continue
		}
		if gotProv != wantProv {
			t.Errorf("model %q provider = %q, want %q", id, gotProv, wantProv)
		}
	}

	// Also verify custom OpenRouter slug detection via SetModel.
	reg := tools.NewRegistry()
	agent := NewAgent(nil, "sk-ant-test", reg, "")
	agent.SetOpenRouterKey("or-key")

	err := agent.SetModel("openai/gpt-4o-mini")
	if err != nil {
		t.Fatalf("SetModel custom slug error: %v", err)
	}
	if agent.Provider() != ProviderOpenRouter {
		t.Errorf("custom slug provider = %q, want %q", agent.Provider(), ProviderOpenRouter)
	}
}

// containsStr is a small helper to check substring presence.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
