package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// ToolFunc executes a tool given raw JSON input and returns a JSON string result.
type ToolFunc func(ctx context.Context, rawInput json.RawMessage) (string, error)

// ToolEntry bundles a tool's metadata with its executor.
type ToolEntry struct {
	Name        string
	Description string
	InputSchema json.RawMessage // JSON Schema object
	Execute     ToolFunc
	Source      string // "builtin" or MCP server name
}

// Registry holds all available tools, both built-in and MCP-discovered.
type Registry struct {
	entries map[string]*ToolEntry
	order   []string // preserves insertion order for listing
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]*ToolEntry),
	}
}

// Register adds a tool to the registry. If a tool with the same name
// already exists, the new one is prefixed with its source to avoid collision.
func (r *Registry) Register(entry ToolEntry) {
	name := entry.Name
	if _, exists := r.entries[name]; exists {
		// Avoid collision by prefixing with source.
		name = entry.Source + "/" + entry.Name
		entry.Name = name
	}
	r.entries[name] = &entry
	r.order = append(r.order, name)
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (*ToolEntry, bool) {
	e, ok := r.entries[name]
	return e, ok
}

// All returns all registered tools in insertion order.
func (r *Registry) All() []*ToolEntry {
	result := make([]*ToolEntry, 0, len(r.order))
	for _, name := range r.order {
		if e, ok := r.entries[name]; ok {
			result = append(result, e)
		}
	}
	return result
}

// AnthropicToolDef is the Anthropic API tool wire format.
type AnthropicToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ToAnthropicTools converts all entries into the format the Anthropic API expects.
func (r *Registry) ToAnthropicTools() []AnthropicToolDef {
	defs := make([]AnthropicToolDef, 0, len(r.order))
	for _, name := range r.order {
		e, ok := r.entries[name]
		if !ok {
			continue
		}
		defs = append(defs, AnthropicToolDef{
			Name:        e.Name,
			Description: e.Description,
			InputSchema: e.InputSchema,
		})
	}
	return defs
}

// ExecuteTool looks up a tool by name and runs it. Returns a JSON result string.
func (r *Registry) ExecuteTool(name string, rawInput json.RawMessage) string {
	entry, ok := r.Get(name)
	if !ok {
		return ErrorJSON("unknown tool: " + name)
	}
	result, err := entry.Execute(context.Background(), rawInput)
	if err != nil {
		return ErrorJSON(err.Error())
	}
	return result
}

// ToJSON marshals a value to a JSON string.
func ToJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ErrorJSON(err.Error())
	}
	return string(data)
}

// ErrorJSON returns a JSON error object.
func ErrorJSON(msg string) string {
	return fmt.Sprintf(`{"error": %q}`, msg)
}
