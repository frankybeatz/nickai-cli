package workflow

import (
	"encoding/json"
	"testing"
)

func TestNodeTypeSerialization(t *testing.T) {
	tests := []struct {
		nodeType NodeType
		want     string
	}{
		{NodeTrigger, "trigger"},
		{NodeSchedule, "schedule"},
		{NodePriceFeed, "price_feed"},
		{NodeData, "data"},
		{NodeAnalysis, "analysis"},
		{NodeLLM, "llm"},
		{NodeCondition, "condition"},
		{NodeFilter, "filter"},
		{NodeTrade, "trade"},
		{NodeExecution, "execution"},
		{NodeNotification, "notification"},
		{NodeWebhook, "webhook"},
	}

	for _, tc := range tests {
		data, _ := json.Marshal(tc.nodeType)
		if string(data) != `"`+tc.want+`"` {
			t.Errorf("Marshal(%v) = %s, want %q", tc.nodeType, data, tc.want)
		}
	}
}

func TestStoreGetNotFound(t *testing.T) {
	s := &Store{}
	if got := s.Get("nonexistent"); got != nil {
		t.Errorf("Get(nonexistent) = %v, want nil", got)
	}
}

func TestStoreAddAndGet(t *testing.T) {
	s := &Store{}

	w := Workflow{
		Name: "test-workflow",
		Nodes: []Node{
			{ID: "n1", Type: NodeTrigger, Name: "Start"},
			{ID: "n2", Type: NodeTrade, Name: "Buy BTC", ConnectsTo: []string{"n3"}},
			{ID: "n3", Type: NodeNotification, Name: "Alert"},
		},
	}

	if err := s.Add(w); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got := s.Get("test-workflow")
	if got == nil {
		t.Fatal("Get returned nil after Add")
	}
	if got.Name != "test-workflow" {
		t.Errorf("Name = %q, want test-workflow", got.Name)
	}
	if got.Status != "stopped" {
		t.Errorf("Status = %q, want stopped", got.Status)
	}
	if got.CreatedAt == "" {
		t.Error("CreatedAt should be set by Add")
	}
	if len(got.Nodes) != 3 {
		t.Errorf("Nodes count = %d, want 3", len(got.Nodes))
	}
}

func TestStoreAddDuplicate(t *testing.T) {
	s := &Store{}

	w := Workflow{
		Name:  "dup",
		Nodes: []Node{{ID: "n1", Type: NodeTrigger, Name: "Start"}},
	}

	if err := s.Add(w); err != nil {
		t.Fatalf("first Add: %v", err)
	}

	err := s.Add(w)
	if err == nil {
		t.Fatal("second Add should return error for duplicate name")
	}
}

func TestStoreRemove(t *testing.T) {
	s := &Store{}
	s.Add(Workflow{Name: "a", Nodes: []Node{{ID: "1", Type: NodeTrigger, Name: "x"}}})
	s.Add(Workflow{Name: "b", Nodes: []Node{{ID: "2", Type: NodeTrade, Name: "y"}}})

	if !s.Remove("a") {
		t.Error("Remove(a) should return true")
	}
	if s.Get("a") != nil {
		t.Error("a should be gone after Remove")
	}
	if s.Get("b") == nil {
		t.Error("b should still exist")
	}

	if s.Remove("nonexistent") {
		t.Error("Remove(nonexistent) should return false")
	}
}

func TestStoreRun(t *testing.T) {
	s := &Store{}
	s.Add(Workflow{
		Name: "runner",
		Nodes: []Node{
			{ID: "n1", Type: NodePriceFeed, Name: "Get Prices"},
			{ID: "n2", Type: NodeCondition, Name: "Check RSI"},
			{ID: "n3", Type: NodeTrade, Name: "Execute Trade"},
		},
	})

	logs, err := s.Run("runner")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Each node produces 2 log entries (started + completed).
	if len(logs) != 6 {
		t.Errorf("expected 6 log entries, got %d", len(logs))
	}

	// Check first and last entries.
	if logs[0].Status != "started" {
		t.Errorf("first log status = %q, want started", logs[0].Status)
	}
	if logs[0].NodeID != "n1" {
		t.Errorf("first log nodeID = %q, want n1", logs[0].NodeID)
	}
	if logs[5].Status != "completed" {
		t.Errorf("last log status = %q, want completed", logs[5].Status)
	}
	if logs[5].NodeID != "n3" {
		t.Errorf("last log nodeID = %q, want n3", logs[5].NodeID)
	}

	// Workflow state should be updated.
	w := s.Get("runner")
	if w.Status != "running" {
		t.Errorf("workflow status = %q, want running", w.Status)
	}
	if w.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", w.RunCount)
	}
	if w.LastRun == "" {
		t.Error("LastRun should be set")
	}
}

func TestStoreRunNotFound(t *testing.T) {
	s := &Store{}
	_, err := s.Run("ghost")
	if err == nil {
		t.Error("Run(ghost) should error")
	}
}

func TestStoreRunAlreadyRunning(t *testing.T) {
	s := &Store{}
	s.Add(Workflow{
		Name:  "active",
		Nodes: []Node{{ID: "n1", Type: NodeTrigger, Name: "Start"}},
	})
	s.Run("active")

	_, err := s.Run("active")
	if err == nil {
		t.Error("Run on already-running workflow should error")
	}
}

func TestStoreStop(t *testing.T) {
	s := &Store{}
	s.Add(Workflow{
		Name:  "stoppable",
		Nodes: []Node{{ID: "n1", Type: NodeTrigger, Name: "Start"}},
	})
	s.Run("stoppable")

	if err := s.Stop("stoppable"); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	w := s.Get("stoppable")
	if w.Status != "stopped" {
		t.Errorf("status after Stop = %q, want stopped", w.Status)
	}
}

func TestStoreStopNotFound(t *testing.T) {
	s := &Store{}
	if err := s.Stop("nope"); err == nil {
		t.Error("Stop(nope) should error")
	}
}

func TestWorkflowSerialization(t *testing.T) {
	w := Workflow{
		Name:   "serialized",
		Status: "stopped",
		Nodes: []Node{
			{
				ID:         "n1",
				Type:       NodeTrigger,
				Name:       "Start",
				Config:     map[string]any{"interval": "5m"},
				ConnectsTo: []string{"n2"},
			},
			{
				ID:   "n2",
				Type: NodeTrade,
				Name: "Trade",
			},
		},
	}

	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Workflow
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Name != w.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, w.Name)
	}
	if len(decoded.Nodes) != 2 {
		t.Fatalf("Nodes count = %d, want 2", len(decoded.Nodes))
	}
	if decoded.Nodes[0].Config["interval"] != "5m" {
		t.Errorf("Config[interval] = %v, want 5m", decoded.Nodes[0].Config["interval"])
	}
	if len(decoded.Nodes[0].ConnectsTo) != 1 || decoded.Nodes[0].ConnectsTo[0] != "n2" {
		t.Errorf("ConnectsTo = %v, want [n2]", decoded.Nodes[0].ConnectsTo)
	}
}
