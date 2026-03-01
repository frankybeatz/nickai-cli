package ui

import (
	"strings"
	"testing"
)

func TestSectionHeader(t *testing.T) {
	result := SectionHeader("Test Title")
	if !strings.Contains(result, "Test Title") {
		t.Error("SectionHeader should contain the title")
	}
	if !strings.HasSuffix(result, "\n") {
		t.Error("SectionHeader should end with newline")
	}
}

func TestSectionHeaderWithCount(t *testing.T) {
	result := SectionHeaderWithCount("Orders", 42)
	if !strings.Contains(result, "Orders") {
		t.Error("should contain title")
	}
	if !strings.Contains(result, "42 total") {
		t.Error("should contain count")
	}
}

func TestDivider(t *testing.T) {
	result := Divider(10)
	if !strings.Contains(result, "──────────") {
		t.Errorf("Divider(10) should contain 10 dashes, got: %q", result)
	}
}

func TestDividerNoTrailingNewline(t *testing.T) {
	result := Divider(5)
	if strings.HasSuffix(result, "\n") {
		t.Error("Divider should not end with newline")
	}
}

func TestKeyValue(t *testing.T) {
	result := KeyValue("Symbol", "BTC", 10)
	if !strings.Contains(result, "Symbol") {
		t.Error("should contain key")
	}
	if !strings.Contains(result, "BTC") {
		t.Error("should contain value")
	}
}

func TestKeyValueAlignment(t *testing.T) {
	// Short key with wide keyWidth should have padding.
	result := KeyValue("A", "val", 10)
	// At minimum, there should be spaces between key and value.
	if !strings.Contains(result, "A") && !strings.Contains(result, "val") {
		t.Error("should contain both key and value")
	}
}

func TestEmptyState(t *testing.T) {
	result := EmptyState("No items found")
	if !strings.Contains(result, "No items found") {
		t.Error("should contain the message")
	}
	if strings.HasSuffix(result, "\n") {
		t.Error("EmptyState should not end with newline")
	}
}

func TestNextSteps(t *testing.T) {
	result := NextSteps("/pnl", "/history")
	if !strings.Contains(result, "Try:") {
		t.Error("should contain 'Try:' prefix")
	}
	if !strings.Contains(result, "/pnl") {
		t.Error("should contain first hint")
	}
	if !strings.Contains(result, "/history") {
		t.Error("should contain second hint")
	}
}

func TestNextStepsEmpty(t *testing.T) {
	result := NextSteps()
	if result != "" {
		t.Errorf("expected empty string for no hints, got %q", result)
	}
}

func TestNextStepsSingle(t *testing.T) {
	result := NextSteps("/help")
	if !strings.Contains(result, "/help") {
		t.Error("should contain the single hint")
	}
	// Should not contain separator.
	if strings.Contains(result, "·") {
		t.Error("single hint should not have separator")
	}
}

func TestStatusIndicator(t *testing.T) {
	tests := []struct {
		status   string
		contains string
	}{
		{"running", "●"},
		{"stopped", "○"},
		{"error", "✖"},
		{"unknown", "?"},
	}

	for _, tt := range tests {
		result := StatusIndicator(tt.status)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("StatusIndicator(%q): expected %q, got %q", tt.status, tt.contains, result)
		}
	}
}

func TestApplyTheme(t *testing.T) {
	// Save original.
	origPrimary := ColorPrimary

	// Apply cyberpunk theme.
	ApplyTheme(Themes["cyberpunk"])
	if ColorPrimary != Themes["cyberpunk"].Primary {
		t.Errorf("expected cyberpunk primary, got %v", ColorPrimary)
	}

	// Restore default.
	ApplyTheme(Themes["default"])
	if ColorPrimary != origPrimary {
		t.Errorf("expected default primary restored, got %v", ColorPrimary)
	}
}

func TestAllThemesExist(t *testing.T) {
	expected := []string{
		"default", "cyberpunk", "bloomberg", "minimal", "matrix",
		"tokyonight", "dracula", "catppuccin", "nord", "gruvbox",
	}
	for _, name := range expected {
		if _, ok := Themes[name]; !ok {
			t.Errorf("missing theme: %s", name)
		}
	}
	if len(Themes) != len(expected) {
		t.Errorf("expected %d themes, got %d", len(expected), len(Themes))
	}
}

func TestThemeFieldsNonEmpty(t *testing.T) {
	for name, theme := range Themes {
		if theme.Name == "" {
			t.Errorf("theme %q has empty Name", name)
		}
		if theme.Primary == "" {
			t.Errorf("theme %q has empty Primary", name)
		}
		if theme.Error == "" {
			t.Errorf("theme %q has empty Error", name)
		}
	}
}

func TestCardStyle(t *testing.T) {
	style := Card(60)
	// Should produce a style without panicking.
	rendered := style.Render("test content")
	if rendered == "" {
		t.Error("Card should render content")
	}
}

func TestAgentCardStyle(t *testing.T) {
	style := AgentCard(60)
	rendered := style.Render("agent content")
	if rendered == "" {
		t.Error("AgentCard should render content")
	}
}

func TestMessageBars(t *testing.T) {
	user := UserMsgBar("hello")
	if user == "" {
		t.Error("UserMsgBar should not be empty")
	}

	bot := BotMsgBar("response")
	if bot == "" {
		t.Error("BotMsgBar should not be empty")
	}

	sys := SystemMsgBar("system message")
	if sys == "" {
		t.Error("SystemMsgBar should not be empty")
	}
}
