package mcp

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestMarshalSchema(t *testing.T) {
	schema := mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"symbol": map[string]interface{}{
				"type":        "string",
				"description": "The trading symbol",
			},
		},
	}

	result := marshalSchema(schema)

	// Should be valid JSON.
	if !json.Valid(result) {
		t.Fatalf("marshalSchema returned invalid JSON: %s", string(result))
	}

	// Should contain the expected fields.
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	if parsed["type"] != "object" {
		t.Errorf("type = %v, want object", parsed["type"])
	}

	props, ok := parsed["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties not found or not an object")
	}
	if _, ok := props["symbol"]; !ok {
		t.Error("symbol property not found")
	}
}

func TestMarshalSchemaEmpty(t *testing.T) {
	schema := mcp.ToolInputSchema{
		Type: "object",
	}

	result := marshalSchema(schema)
	if !json.Valid(result) {
		t.Fatalf("marshalSchema(empty) returned invalid JSON: %s", string(result))
	}
}

func TestHasTradeCapability(t *testing.T) {
	tests := []struct {
		name string
		caps []Capability
		want bool
	}{
		{"nil caps", nil, false},
		{"empty caps", []Capability{}, false},
		{"read only", []Capability{CapReadData}, false},
		{"analytics only", []Capability{CapAnalytics}, false},
		{"has trade", []Capability{CapReadData, CapTrade}, true},
		{"has on-chain", []Capability{CapOnChain}, true},
		{"trade and on-chain", []Capability{CapTrade, CapOnChain}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasTradeCapability(tc.caps); got != tc.want {
				t.Errorf("hasTradeCapability(%v) = %v, want %v", tc.caps, got, tc.want)
			}
		})
	}
}

func TestNewClientManager(t *testing.T) {
	cm := NewClientManager("1.0.0")
	if cm == nil {
		t.Fatal("NewClientManager returned nil")
	}
	if cm.ConnectionCount() != 0 {
		t.Errorf("ConnectionCount = %d, want 0", cm.ConnectionCount())
	}
	if len(cm.Connections()) != 0 {
		t.Errorf("Connections length = %d, want 0", len(cm.Connections()))
	}
	if len(cm.Failed()) != 0 {
		t.Errorf("Failed length = %d, want 0", len(cm.Failed()))
	}
}

func TestClientManagerReconnectFailedEmpty(t *testing.T) {
	cm := NewClientManager("1.0.0")
	recovered := cm.ReconnectFailed()
	if recovered != 0 {
		t.Errorf("ReconnectFailed (no failures) = %d, want 0", recovered)
	}
}
