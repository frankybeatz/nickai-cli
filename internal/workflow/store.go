package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// NodeType represents the type of a workflow node.
type NodeType string

const (
	NodeTrigger      NodeType = "trigger"
	NodeSchedule     NodeType = "schedule"
	NodePriceFeed    NodeType = "price_feed"
	NodeData         NodeType = "data"
	NodeAnalysis     NodeType = "analysis"
	NodeLLM          NodeType = "llm"
	NodeCondition    NodeType = "condition"
	NodeFilter       NodeType = "filter"
	NodeTrade        NodeType = "trade"
	NodeExecution    NodeType = "execution"
	NodeNotification NodeType = "notification"
	NodeWebhook      NodeType = "webhook"
)

// Node is a single step in a workflow.
type Node struct {
	ID         string         `json:"id"`
	Type       NodeType       `json:"type"`
	Name       string         `json:"name"`
	Config     map[string]any `json:"config,omitempty"`
	ConnectsTo []string       `json:"connects_to,omitempty"`
}

// LogEntry is a single execution log line.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	NodeID    string `json:"node_id"`
	NodeName  string `json:"node_name"`
	Status    string `json:"status"` // started, completed, skipped, error
	Message   string `json:"message,omitempty"`
}

// Workflow is a multi-node automation pipeline.
type Workflow struct {
	Name      string     `json:"name"`
	Nodes     []Node     `json:"nodes"`
	Status    string     `json:"status"` // stopped, running
	RunCount  int        `json:"run_count"`
	LastRun   string     `json:"last_run,omitempty"`
	CreatedAt string     `json:"created_at"`
	Logs      []LogEntry `json:"logs,omitempty"`
}

// Store manages workflows on disk.
type Store struct {
	Workflows []Workflow `json:"workflows"`
}

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nickai", "workflows.json"), nil
}

// Load reads workflows from disk.
func Load() (*Store, error) {
	s := &Store{}
	path, err := storePath()
	if err != nil {
		return s, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return &Store{}, err
	}
	return s, nil
}

// Save writes workflows to disk.
func (s *Store) Save() error {
	path, err := storePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// Get returns a workflow by name, or nil if not found.
func (s *Store) Get(name string) *Workflow {
	for i, w := range s.Workflows {
		if w.Name == name {
			return &s.Workflows[i]
		}
	}
	return nil
}

// Add adds a new workflow. Returns error if one with same name exists.
func (s *Store) Add(w Workflow) error {
	if s.Get(w.Name) != nil {
		return fmt.Errorf("workflow %q already exists", w.Name)
	}
	w.Status = "stopped"
	w.CreatedAt = time.Now().Format(time.RFC3339)
	s.Workflows = append(s.Workflows, w)
	return nil
}

// Remove deletes a workflow by name. Returns true if found.
func (s *Store) Remove(name string) bool {
	for i, w := range s.Workflows {
		if w.Name == name {
			s.Workflows = append(s.Workflows[:i], s.Workflows[i+1:]...)
			return true
		}
	}
	return false
}

// CreateFromFile reads a workflow JSON file and adds it to the store.
func (s *Store) CreateFromFile(path string) (*Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	var w Workflow
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if w.Name == "" {
		return nil, fmt.Errorf("workflow must have a name")
	}
	if len(w.Nodes) == 0 {
		return nil, fmt.Errorf("workflow must have at least one node")
	}
	if err := s.Add(w); err != nil {
		return nil, err
	}
	return s.Get(w.Name), nil
}

// Run simulates starting a workflow, generating mock execution logs.
func (s *Store) Run(name string) ([]LogEntry, error) {
	w := s.Get(name)
	if w == nil {
		return nil, fmt.Errorf("workflow %q not found", name)
	}
	if w.Status == "running" {
		return nil, fmt.Errorf("workflow %q is already running", name)
	}

	w.Status = "running"
	w.RunCount++
	now := time.Now()
	w.LastRun = now.Format(time.RFC3339)

	// Generate mock execution logs.
	var logs []LogEntry
	t := now
	for _, node := range w.Nodes {
		logs = append(logs, LogEntry{
			Timestamp: t.Format("15:04:05.000"),
			NodeID:    node.ID,
			NodeName:  node.Name,
			Status:    "started",
			Message:   fmt.Sprintf("Executing %s node", node.Type),
		})
		t = t.Add(120 * time.Millisecond)
		logs = append(logs, LogEntry{
			Timestamp: t.Format("15:04:05.000"),
			NodeID:    node.ID,
			NodeName:  node.Name,
			Status:    "completed",
			Message:   "OK",
		})
		t = t.Add(80 * time.Millisecond)
	}

	w.Logs = logs
	return logs, nil
}

// Stop sets a workflow's status to stopped.
func (s *Store) Stop(name string) error {
	w := s.Get(name)
	if w == nil {
		return fmt.Errorf("workflow %q not found", name)
	}
	w.Status = "stopped"
	return nil
}
