package notify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	os.MkdirAll(filepath.Join(dir, ".nickai"), 0o700)
	return dir, func() {
		os.Setenv("HOME", origHome)
	}
}

func TestConfigIsEmpty(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"all false", Config{}, true},
		{"desktop only", Config{Desktop: true}, false},
		{"webhook only", Config{WebhookURL: "https://example.com"}, false},
		{"sound only", Config{Sound: true}, false},
		{"all enabled", Config{Desktop: true, WebhookURL: "https://example.com", Sound: true}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.IsEmpty(); got != tc.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestConfigSerialization(t *testing.T) {
	cfg := Config{
		Desktop:    true,
		WebhookURL: "https://hooks.example.com/notify",
		Sound:      false,
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Desktop != cfg.Desktop {
		t.Errorf("Desktop = %v, want %v", decoded.Desktop, cfg.Desktop)
	}
	if decoded.WebhookURL != cfg.WebhookURL {
		t.Errorf("WebhookURL = %q, want %q", decoded.WebhookURL, cfg.WebhookURL)
	}
	if decoded.Sound != cfg.Sound {
		t.Errorf("Sound = %v, want %v", decoded.Sound, cfg.Sound)
	}
}

func TestConfigOmitEmptyWebhook(t *testing.T) {
	cfg := Config{Desktop: true}
	data, _ := json.Marshal(cfg)

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	if _, exists := raw["webhook_url"]; exists {
		t.Error("empty WebhookURL should be omitted (omitempty)")
	}
}

func TestEscapeAppleScript(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello world", "hello world"},
		{`say "hi"`, `say \"hi\"`},
		{`path\to\file`, `path\\to\\file`},
		{`mixed "quotes" and \slashes\`, `mixed \"quotes\" and \\slashes\\`},
		{"", ""},
	}

	for _, tc := range tests {
		got := escapeAppleScript(tc.input)
		if got != tc.want {
			t.Errorf("escapeAppleScript(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSendNilConfig(t *testing.T) {
	// Should not panic.
	Send(nil, "title", "body")
}

func TestSendEmptyConfig(t *testing.T) {
	// Should not panic or send anything.
	Send(&Config{}, "title", "body")
}

func TestSendWebhook(t *testing.T) {
	var receivedBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := sendWebhook(server.URL, "Price Alert", "BTC > $70k")
	if err != nil {
		t.Fatalf("sendWebhook: %v", err)
	}

	if receivedBody["title"] != "Price Alert" {
		t.Errorf("title = %q, want 'Price Alert'", receivedBody["title"])
	}
	if receivedBody["body"] != "BTC > $70k" {
		t.Errorf("body = %q, want 'BTC > $70k'", receivedBody["body"])
	}
	if receivedBody["timestamp"] == "" {
		t.Error("timestamp should not be empty")
	}
}

func TestSendWebhookError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := sendWebhook(server.URL, "test", "test")
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestLoadSaveConfig(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// Load from nonexistent file returns empty config.
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load (empty): %v", err)
	}
	if !cfg.IsEmpty() {
		t.Error("expected empty config from nonexistent file")
	}

	// Save and reload.
	cfg.Desktop = true
	cfg.WebhookURL = "https://example.com/hook"
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !loaded.Desktop {
		t.Error("loaded Desktop should be true")
	}
	if loaded.WebhookURL != "https://example.com/hook" {
		t.Errorf("loaded WebhookURL = %q", loaded.WebhookURL)
	}
}
