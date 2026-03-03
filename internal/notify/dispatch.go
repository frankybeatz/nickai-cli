package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/nickai/cli/internal/safefile"
)

// Config holds notification preferences.
type Config struct {
	Desktop    bool   `json:"desktop"`
	WebhookURL string `json:"webhook_url,omitempty"`
	Sound      bool   `json:"sound"`
}

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nickai", "notify.json"), nil
}

// Load reads notification config from disk.
func Load() (*Config, error) {
	path, err := storePath()
	if err != nil {
		return &Config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return &Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &Config{}, err
	}
	return &cfg, nil
}

// Save writes notification config to disk.
func Save(cfg *Config) error {
	path, err := storePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return safefile.AtomicWrite(path, data, 0o600)
}

// IsEmpty returns true if no notification channels are configured.
func (c *Config) IsEmpty() bool {
	return !c.Desktop && c.WebhookURL == "" && !c.Sound
}

// Send dispatches a notification to all enabled channels.
// Non-blocking: desktop runs in a goroutine, webhook has a 5s timeout.
func Send(cfg *Config, title, body string) {
	if cfg == nil || cfg.IsEmpty() {
		return
	}
	if cfg.Desktop {
		go sendDesktop(title, body)
	}
	if cfg.WebhookURL != "" {
		go sendWebhook(cfg.WebhookURL, title, body)
	}
}

// sendDesktop sends a native desktop notification.
func sendDesktop(title, body string) {
	switch runtime.GOOS {
	case "darwin":
		script := `display notification "` + escapeAppleScript(body) + `" with title "` + escapeAppleScript(title) + `"`
		_ = exec.Command("osascript", "-e", script).Run()
	case "linux":
		_ = exec.Command("notify-send", title, body).Run()
	}
}

// escapeAppleScript escapes quotes for AppleScript strings.
func escapeAppleScript(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		if r == '"' {
			buf.WriteString(`\"`)
		} else if r == '\\' {
			buf.WriteString(`\\`)
		} else {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// sendWebhook POSTs a JSON payload to the configured URL.
func sendWebhook(url, title, body string) error {
	payload := map[string]string{
		"title":     title,
		"body":      body,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}
