package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if len(r.All()) != 0 {
		t.Errorf("new registry should be empty, got %d entries", len(r.All()))
	}
}

func TestRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(ToolEntry{
		Name:        "test_tool",
		Description: "A test tool",
		Source:      "builtin",
		Execute: func(ctx context.Context, raw json.RawMessage) (string, error) {
			return `{"ok": true}`, nil
		},
	})

	entry, ok := r.Get("test_tool")
	if !ok {
		t.Fatal("expected to find test_tool")
	}
	if entry.Name != "test_tool" {
		t.Errorf("Name = %q, want test_tool", entry.Name)
	}
	if entry.Description != "A test tool" {
		t.Errorf("Description = %q, want 'A test tool'", entry.Description)
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent tool to not be found")
	}
}

func TestRegisterCollision(t *testing.T) {
	r := NewRegistry()
	r.Register(ToolEntry{Name: "prices", Source: "builtin", Execute: func(ctx context.Context, raw json.RawMessage) (string, error) { return "", nil }})
	r.Register(ToolEntry{Name: "prices", Source: "mcp-exchange", Execute: func(ctx context.Context, raw json.RawMessage) (string, error) { return "", nil }})

	// First should be found as-is.
	_, ok := r.Get("prices")
	if !ok {
		t.Error("expected to find 'prices'")
	}

	// Collision should be prefixed with source.
	_, ok = r.Get("mcp-exchange/prices")
	if !ok {
		t.Error("expected to find 'mcp-exchange/prices'")
	}
}

func TestAll(t *testing.T) {
	r := NewRegistry()
	r.Register(ToolEntry{Name: "a", Source: "builtin", Execute: func(ctx context.Context, raw json.RawMessage) (string, error) { return "", nil }})
	r.Register(ToolEntry{Name: "b", Source: "builtin", Execute: func(ctx context.Context, raw json.RawMessage) (string, error) { return "", nil }})
	r.Register(ToolEntry{Name: "c", Source: "builtin", Execute: func(ctx context.Context, raw json.RawMessage) (string, error) { return "", nil }})

	all := r.All()
	if len(all) != 3 {
		t.Fatalf("All() returned %d entries, want 3", len(all))
	}
	// Insertion order preserved.
	if all[0].Name != "a" || all[1].Name != "b" || all[2].Name != "c" {
		t.Errorf("order not preserved: got %s, %s, %s", all[0].Name, all[1].Name, all[2].Name)
	}
}

func TestToAnthropicTools(t *testing.T) {
	r := NewRegistry()
	schema := json.RawMessage(`{"type": "object"}`)
	r.Register(ToolEntry{
		Name:        "get_prices",
		Description: "Get crypto prices",
		InputSchema: schema,
		Source:      "builtin",
		Execute:     func(ctx context.Context, raw json.RawMessage) (string, error) { return "", nil },
	})

	defs := r.ToAnthropicTools()
	if len(defs) != 1 {
		t.Fatalf("ToAnthropicTools returned %d defs, want 1", len(defs))
	}
	if defs[0].Name != "get_prices" {
		t.Errorf("Name = %q, want get_prices", defs[0].Name)
	}
}

func TestExecuteTool(t *testing.T) {
	r := NewRegistry()
	r.Register(ToolEntry{
		Name:   "echo",
		Source: "builtin",
		Execute: func(ctx context.Context, raw json.RawMessage) (string, error) {
			return `{"result": "hello"}`, nil
		},
	})

	result := r.ExecuteTool("echo", nil)
	if !strings.Contains(result, "hello") {
		t.Errorf("ExecuteTool result = %q, want to contain 'hello'", result)
	}
}

func TestExecuteToolUnknown(t *testing.T) {
	r := NewRegistry()
	result := r.ExecuteTool("nonexistent", nil)
	if !strings.Contains(result, "unknown tool") {
		t.Errorf("expected 'unknown tool' error, got %q", result)
	}
}

func TestExecuteToolTruncation(t *testing.T) {
	r := NewRegistry()
	// Return a result larger than 16KB.
	bigResult := strings.Repeat("x", maxToolResultBytes+1000)
	r.Register(ToolEntry{
		Name:   "big",
		Source: "builtin",
		Execute: func(ctx context.Context, raw json.RawMessage) (string, error) {
			return bigResult, nil
		},
	})

	result := r.ExecuteTool("big", nil)
	if len(result) > maxToolResultBytes+100 {
		t.Errorf("result should be truncated, got length %d", len(result))
	}
	if !strings.HasSuffix(result, "...(truncated)") {
		t.Error("truncated result should end with ...(truncated)")
	}
}

func TestToJSON(t *testing.T) {
	result := ToJSON(map[string]string{"key": "value"})
	if result != `{"key":"value"}` {
		t.Errorf("ToJSON = %q, want {\"key\":\"value\"}", result)
	}
}

func TestErrorJSON(t *testing.T) {
	result := ErrorJSON("something went wrong")
	if !strings.Contains(result, "something went wrong") {
		t.Errorf("ErrorJSON = %q, want to contain error message", result)
	}
	// Should be valid JSON.
	var parsed map[string]string
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("ErrorJSON result is not valid JSON: %v", err)
	}
}
