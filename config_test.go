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
  rounding:
    - step: floor
      to: 1m
    - step: ceil
      to: 5m
jira:
  base_url: https://mycompany.atlassian.net
  username: user@example.com
  token: jira-secret-token
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
	if len(cfg.Tempo.Rounding) != 2 {
		t.Errorf("rounding: got %d steps, want 2", len(cfg.Tempo.Rounding))
	} else {
		if cfg.Tempo.Rounding[0].Step != "floor" || cfg.Tempo.Rounding[0].To != "1m" {
			t.Errorf("rounding[0]: got %s:%s, want floor:1m", cfg.Tempo.Rounding[0].Step, cfg.Tempo.Rounding[0].To)
		}
		if cfg.Tempo.Rounding[1].Step != "ceil" || cfg.Tempo.Rounding[1].To != "5m" {
			t.Errorf("rounding[1]: got %s:%s, want ceil:5m", cfg.Tempo.Rounding[1].Step, cfg.Tempo.Rounding[1].To)
		}
	}
	if cfg.Jira.BaseURL != "https://mycompany.atlassian.net" {
		t.Errorf("jira.base_url: got %q, want %q", cfg.Jira.BaseURL, "https://mycompany.atlassian.net")
	}
	if cfg.Jira.Username != "user@example.com" {
		t.Errorf("jira.username: got %q, want %q", cfg.Jira.Username, "user@example.com")
	}
	if cfg.Jira.Token != "jira-secret-token" {
		t.Errorf("jira.token: got %q, want %q", cfg.Jira.Token, "jira-secret-token")
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

func TestResolveJiraToken(t *testing.T) {
	cfg := &Config{Jira: JiraConfig{Token: ""}}
	_, err := cfg.ResolveJiraToken()
	if err == nil {
		t.Error("expected error for empty jira token")
	}

	cfg.Jira.Token = "plain-jira-token"
	tok, err := cfg.ResolveJiraToken()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if tok != "plain-jira-token" {
		t.Errorf("plain token: got %q, want %q", tok, "plain-jira-token")
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

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", origXDG)

	cfg := &Config{
		WorklogFile: "/tmp/tasks.ics",
		Tempo: TempoConfig{
			BaseURL:   "https://api.tempo.io/4",
			Token:     "secret",
			AccountID: "abc-123",
		},
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig after save: %v", err)
	}
	if loaded.WorklogFile != cfg.WorklogFile {
		t.Errorf("worklog_file: got %q, want %q", loaded.WorklogFile, cfg.WorklogFile)
	}
	if loaded.Tempo.Token != cfg.Tempo.Token {
		t.Errorf("token: got %q, want %q", loaded.Tempo.Token, cfg.Tempo.Token)
	}
}

func TestConfigGetSet(t *testing.T) {
	cfg := &Config{}

	// Set and get worklog_file
	if err := cfg.Set("worklog_file", "/path/to/tasks.ics"); err != nil {
		t.Fatalf("Set worklog_file: %v", err)
	}
	val, err := cfg.Get("worklog_file")
	if err != nil {
		t.Fatalf("Get worklog_file: %v", err)
	}
	if val != "/path/to/tasks.ics" {
		t.Errorf("worklog_file: got %q", val)
	}

	// Token is hidden by default
	cfg.Tempo.Token = "secret"
	val, err = cfg.Get("tempo.token")
	if err != nil {
		t.Fatalf("Get tempo.token: %v", err)
	}
	if val != "<hidden>" {
		t.Errorf("expected <hidden>, got %q", val)
	}

	// Unmasked token
	val, err = cfg.GetUnmasked("tempo.token")
	if err != nil {
		t.Fatalf("GetUnmasked tempo.token: %v", err)
	}
	if val != "secret" {
		t.Errorf("expected secret, got %q", val)
	}

	// Empty token returns empty, not <hidden>
	cfg.Tempo.Token = ""
	val, err = cfg.Get("tempo.token")
	if err != nil {
		t.Fatalf("Get empty token: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty, got %q", val)
	}

	// Jira username visible by default
	cfg.Jira.Username = "user@example.com"
	val, err = cfg.Get("jira.username")
	if err != nil {
		t.Fatalf("Get jira.username: %v", err)
	}
	if val != "user@example.com" {
		t.Errorf("expected user@example.com, got %q", val)
	}

	// Unmasked jira username
	val, err = cfg.GetUnmasked("jira.username")
	if err != nil {
		t.Fatalf("GetUnmasked jira.username: %v", err)
	}
	if val != "user@example.com" {
		t.Errorf("expected user@example.com, got %q", val)
	}

	// Jira token hidden by default
	cfg.Jira.Token = "jira-secret"
	val, err = cfg.Get("jira.token")
	if err != nil {
		t.Fatalf("Get jira.token: %v", err)
	}
	if val != "<hidden>" {
		t.Errorf("expected <hidden>, got %q", val)
	}

	// Unmasked jira token
	val, err = cfg.GetUnmasked("jira.token")
	if err != nil {
		t.Fatalf("GetUnmasked jira.token: %v", err)
	}
	if val != "jira-secret" {
		t.Errorf("expected jira-secret, got %q", val)
	}

	// Unknown key
	_, err = cfg.Get("unknown")
	if err == nil {
		t.Error("expected error for unknown key")
	}
	if err := cfg.Set("unknown", "x"); err == nil {
		t.Error("expected error for unknown key set")
	}

	// Rounding set/get
	if err := cfg.Set("tempo.rounding", "floor:1m,ceil:5m"); err != nil {
		t.Fatalf("Set tempo.rounding: %v", err)
	}
	val, err = cfg.Get("tempo.rounding")
	if err != nil {
		t.Fatalf("Get tempo.rounding: %v", err)
	}
	if val != "floor:1m,ceil:5m" {
		t.Errorf("tempo.rounding: got %q, want %q", val, "floor:1m,ceil:5m")
	}

	// Empty rounding
	if err := cfg.Set("tempo.rounding", ""); err != nil {
		t.Fatalf("Set tempo.rounding empty: %v", err)
	}
	val, err = cfg.Get("tempo.rounding")
	if err != nil {
		t.Fatalf("Get tempo.rounding empty: %v", err)
	}
	if val != "" {
		t.Errorf("tempo.rounding empty: got %q, want empty", val)
	}

	// Invalid rounding step
	if err := cfg.Set("tempo.rounding", "invalid"); err == nil {
		t.Error("expected error for invalid rounding step")
	}
}

func TestIsValidConfigKey(t *testing.T) {
	for _, key := range []string{"worklog_file", "tempo.base_url", "tempo.token", "tempo.account_id", "tempo.rounding", "jira.base_url", "jira.username", "jira.token"} {
		if !isValidConfigKey(key) {
			t.Errorf("expected %q to be valid", key)
		}
	}
	if isValidConfigKey("bogus") {
		t.Error("expected bogus to be invalid")
	}
}

func TestExpandTilde(t *testing.T) {
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", "/home/testuser")
	defer os.Setenv("HOME", origHome)

	got := expandTilde("~/tasks.ics")
	want := filepath.Join("/home", "testuser", "tasks.ics")
	if got != want {
		t.Errorf("expandTilde: got %q, want %q", got, want)
	}

	got = expandTilde("/absolute/path.ics")
	if got != "/absolute/path.ics" {
		t.Errorf("expandTilde absolute: got %q", got)
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
