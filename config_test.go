package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	ics "github.com/arran4/golang-ical"
)

func TestConfigPath(t *testing.T) {
	// Save and restore original env vars
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_CONFIG_HOME", origXDG)
		os.Setenv("HOME", origHome)
	}()

	// Test XDG_CONFIG_HOME takes precedence
	os.Setenv("XDG_CONFIG_HOME", "/xdg/config")
	os.Unsetenv("HOME")
	got := configPath()
	want := filepath.Join("/xdg", "config", "worklog", "config.yaml")
	if got != want {
		t.Errorf("XDG_CONFIG_HOME path: got %q, want %q", got, want)
	}

	// Test default home path on Linux
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Setenv("HOME", "/home/testuser")
	got = configPath()
	if runtime.GOOS == "darwin" {
		want = filepath.Join("/home", "testuser", "Library", "Application Support", "worklog", "config.yaml")
	} else {
		want = filepath.Join("/home", "testuser", ".config", "worklog", "config.yaml")
	}
	if got != want {
		t.Errorf("default path: got %q, want %q", got, want)
	}
}

func TestLoadConfigNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	// Point to a directory that exists but contains no worklog/config.yaml
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error for missing config, got %v", err)
	}
	if cfg.WorklogFile != "" {
		t.Errorf("expected empty config, got worklog_file=%q", cfg.WorklogFile)
	}
}

func TestLoadConfigValid(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	configContent := `
worklog_file: /path/to/tasks.ics
tempo:
  base_url: https://custom.tempo.io
  token: my-secret-token
  account_id: abc-123
`
	configDir := filepath.Join(tmpDir, "worklog")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.WorklogFile != "/path/to/tasks.ics" {
		t.Errorf("worklog_file: got %q, want %q", cfg.WorklogFile, "/path/to/tasks.ics")
	}
	if cfg.Tempo.BaseURL != "https://custom.tempo.io" {
		t.Errorf("base_url: got %q, want %q", cfg.Tempo.BaseURL, "https://custom.tempo.io")
	}
	if cfg.Tempo.Token != "my-secret-token" {
		t.Errorf("token: got %q, want %q", cfg.Tempo.Token, "my-secret-token")
	}
	if cfg.Tempo.AccountID != "abc-123" {
		t.Errorf("account_id: got %q, want %q", cfg.Tempo.AccountID, "abc-123")
	}
}

func TestResolveTempoToken(t *testing.T) {
	cfg := &Config{Tempo: TempoConfig{Token: ""}}
	_, err := cfg.ResolveTempoToken()
	if err == nil {
		t.Error("expected error for empty token")
	}

	cfg.Tempo.Token = "plain-token"
	tok, err := cfg.ResolveTempoToken()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tok != "plain-token" {
		t.Errorf("plain token: got %q, want %q", tok, "plain-token")
	}
}

func TestResolvePassSecret(t *testing.T) {
	// Create a fake pass script in a temp directory and prepend to PATH
	tmpDir := t.TempDir()
	script := `#!/bin/sh
echo "secret-from-pass"
`
	scriptPath := filepath.Join(tmpDir, "pass")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake pass: %v", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(filepath.ListSeparator)+origPath)
	defer os.Setenv("PATH", origPath)

	got, err := resolvePassSecret("worklog/tempo-token")
	if err != nil {
		t.Fatalf("resolvePassSecret: %v", err)
	}
	if strings.TrimSpace(got) != "secret-from-pass" {
		t.Errorf("got %q, want %q", got, "secret-from-pass")
	}
}

func TestLoadConfigPassPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	// Create a fake pass script
	scriptDir := t.TempDir()
	script := `#!/bin/sh
if [ "$1" = "worklog/token" ]; then
  echo "pass-secret"
else
  echo "wrong" >&2
  exit 1
fi
`
	scriptPath := filepath.Join(scriptDir, "pass")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake pass: %v", err)
	}
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", scriptDir+string(filepath.ListSeparator)+origPath)
	defer os.Setenv("PATH", origPath)

	configContent := `tempo:
  token: pass:worklog/token
`
	configDir := filepath.Join(tmpDir, "worklog")
	os.MkdirAll(configDir, 0755)
	configFile := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configFile, []byte(configContent), 0644)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	tok, err := cfg.ResolveTempoToken()
	if err != nil {
		t.Fatalf("resolve token: %v", err)
	}
	if strings.TrimSpace(tok) != "pass-secret" {
		t.Errorf("resolved token: got %q, want %q", tok, "pass-secret")
	}
}

func TestLoadConfigMalformed(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	configDir := filepath.Join(tmpDir, "worklog")
	os.MkdirAll(configDir, 0755)
	configFile := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configFile, []byte("not: valid: yaml: {["), 0644)

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for malformed yaml")
	}
}

func TestLoadWorklogConfigFallback(t *testing.T) {
	// Create a temp ics file and temp config pointing to it
	tmpDir := t.TempDir()
	icsPath := filepath.Join(tmpDir, "tasks.ics")
	cal := ics.NewCalendar()
	todo := ics.VTodo{}
	todo.SetProperty(ics.ComponentPropertyUniqueId, "test-uuid-1")
	todo.SetProperty(ics.ComponentPropertySummary, "Test Task")
	cal.Components = append(cal.Components, &todo)
	os.WriteFile(icsPath, []byte(cal.Serialize()), 0644)

	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	configDir := filepath.Join(tmpDir, "worklog")
	os.MkdirAll(configDir, 0755)
	configFile := filepath.Join(configDir, "config.yaml")
	configContent := fmt.Sprintf("worklog_file: %s\n", icsPath)
	os.WriteFile(configFile, []byte(configContent), 0644)

	// Unset WORKLOG_FILE and don't pass -file flag
	origWorklogFile := os.Getenv("WORKLOG_FILE")
	os.Unsetenv("WORKLOG_FILE")
	defer func() {
		if origWorklogFile != "" {
			os.Setenv("WORKLOG_FILE", origWorklogFile)
		}
	}()

	wl, err := loadWorklog("")
	if err != nil {
		t.Fatalf("loadWorklog fallback: %v", err)
	}
	if wl.path != icsPath {
		t.Errorf("path: got %q, want %q", wl.path, icsPath)
	}
}
