package mcp

import (
	"testing"
)

func TestValidateCommand_Allowed(t *testing.T) {
	allowed := []string{"npx", "node", "python", "python3", "uvx", "docker", "deno", "bun"}
	for _, cmd := range allowed {
		if err := ValidateCommand(cmd); err != nil {
			t.Errorf("ValidateCommand(%q) unexpected error: %v", cmd, err)
		}
	}
}

func TestValidateCommand_Blocked(t *testing.T) {
	blocked := []string{"rm", "bash", "sh", "curl", "wget", "cat", "nc", "dd"}
	for _, cmd := range blocked {
		if err := ValidateCommand(cmd); err == nil {
			t.Errorf("ValidateCommand(%q) expected error, got nil", cmd)
		}
	}
}

func TestValidateCommand_WithPath(t *testing.T) {
	tests := []struct {
		command string
		wantErr bool
	}{
		{"/usr/bin/node", false},
		{"/usr/local/bin/npx", false},
		{"/home/user/.nvm/versions/node/v20/bin/node", false},
		{"/usr/bin/python3", false},
		{"/bin/bash", true},
		{"/usr/bin/rm", true},
		{"/usr/bin/curl", true},
	}
	for _, tt := range tests {
		err := ValidateCommand(tt.command)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateCommand(%q) expected error, got nil", tt.command)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateCommand(%q) unexpected error: %v", tt.command, err)
		}
	}
}

func TestSanitizeEnv_FiltersSecrets(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"HOME=/home/user",
		"ANTHROPIC_KEY=sk-secret",
		"AWS_SECRET_ACCESS_KEY=secret123",
		"OPENAI_API_KEY=sk-proj-xxx",
		"DATABASE_URL=postgres://...",
	}
	result := SanitizeEnv(env, nil)
	for _, entry := range result {
		if entry == "ANTHROPIC_KEY=sk-secret" ||
			entry == "AWS_SECRET_ACCESS_KEY=secret123" ||
			entry == "OPENAI_API_KEY=sk-proj-xxx" ||
			entry == "DATABASE_URL=postgres://..." {
			t.Errorf("SanitizeEnv should have filtered %q", entry)
		}
	}
	// PATH and HOME should be present.
	found := map[string]bool{}
	for _, entry := range result {
		if entry == "PATH=/usr/bin" {
			found["PATH"] = true
		}
		if entry == "HOME=/home/user" {
			found["HOME"] = true
		}
	}
	if !found["PATH"] {
		t.Error("SanitizeEnv should keep PATH")
	}
	if !found["HOME"] {
		t.Error("SanitizeEnv should keep HOME")
	}
}

func TestSanitizeEnv_KeepsDefaults(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"HOME=/home/user",
		"USER=testuser",
		"TERM=xterm-256color",
		"LANG=en_US.UTF-8",
		"LC_ALL=en_US.UTF-8",
		"NODE_PATH=/usr/lib/node_modules",
		"PYTHONPATH=/usr/lib/python3",
	}
	result := SanitizeEnv(env, nil)
	if len(result) != len(env) {
		t.Errorf("SanitizeEnv should keep all defaults: got %d, want %d", len(result), len(env))
	}
}

func TestSanitizeEnv_CustomAllowed(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"MY_MCP_TOKEN=abc123",
		"SECRET_KEY=xyz",
	}
	allowed := map[string]bool{"MY_MCP_TOKEN": true}
	result := SanitizeEnv(env, allowed)

	foundToken := false
	foundSecret := false
	for _, entry := range result {
		if entry == "MY_MCP_TOKEN=abc123" {
			foundToken = true
		}
		if entry == "SECRET_KEY=xyz" {
			foundSecret = true
		}
	}
	if !foundToken {
		t.Error("SanitizeEnv should allow custom allowed vars")
	}
	if foundSecret {
		t.Error("SanitizeEnv should filter non-allowed vars")
	}
}
