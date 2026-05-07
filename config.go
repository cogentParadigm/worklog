package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WorklogFile string      `yaml:"worklog_file"`
	Tempo       TempoConfig `yaml:"tempo"`
	Jira        JiraConfig  `yaml:"jira"`
}

type TempoConfig struct {
	BaseURL   string `yaml:"base_url"`
	Token     string `yaml:"token"`
	AccountID string `yaml:"account_id"`
}

type JiraConfig struct {
	BaseURL string `yaml:"base_url"`
	Token   string `yaml:"token"`
}

func LoadConfig() (*Config, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}
	return &cfg, nil
}

func (cfg *Config) ResolveTempoToken() (string, error) {
	if cfg.Tempo.Token == "" {
		return "", fmt.Errorf("tempo token not configured")
	}
	if strings.HasPrefix(cfg.Tempo.Token, "pass:") {
		return resolvePassSecret(strings.TrimPrefix(cfg.Tempo.Token, "pass:"))
	}
	return cfg.Tempo.Token, nil
}

func (cfg *Config) ResolveJiraToken() (string, error) {
	if cfg.Jira.Token == "" {
		return "", fmt.Errorf("jira token not configured")
	}
	if strings.HasPrefix(cfg.Jira.Token, "pass:") {
		return resolvePassSecret(strings.TrimPrefix(cfg.Jira.Token, "pass:"))
	}
	return cfg.Jira.Token, nil
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "worklog")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "worklog")
	}
	return filepath.Join(home, ".config", "worklog")
}

func configPath() string {
	return filepath.Join(configDir(), "config.yaml")
}

func SaveConfig(cfg *Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	path := configPath()
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config file %s: %w", path, err)
	}
	return nil
}

func (cfg *Config) Get(key string) (string, error) {
	switch key {
	case "worklog_file":
		return cfg.WorklogFile, nil
	case "tempo.base_url":
		return cfg.Tempo.BaseURL, nil
	case "tempo.token":
		if cfg.Tempo.Token == "" {
			return "", nil
		}
		return "<hidden>", nil
	case "tempo.account_id":
		return cfg.Tempo.AccountID, nil
	case "jira.base_url":
		return cfg.Jira.BaseURL, nil
	case "jira.token":
		if cfg.Jira.Token == "" {
			return "", nil
		}
		return "<hidden>", nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

func (cfg *Config) GetUnmasked(key string) (string, error) {
	switch key {
	case "worklog_file":
		return cfg.WorklogFile, nil
	case "tempo.base_url":
		return cfg.Tempo.BaseURL, nil
	case "tempo.token":
		return cfg.Tempo.Token, nil
	case "tempo.account_id":
		return cfg.Tempo.AccountID, nil
	case "jira.base_url":
		return cfg.Jira.BaseURL, nil
	case "jira.token":
		return cfg.Jira.Token, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

func (cfg *Config) Set(key, value string) error {
	switch key {
	case "worklog_file":
		cfg.WorklogFile = value
	case "tempo.base_url":
		cfg.Tempo.BaseURL = value
	case "tempo.token":
		cfg.Tempo.Token = value
	case "tempo.account_id":
		cfg.Tempo.AccountID = value
	case "jira.base_url":
		cfg.Jira.BaseURL = value
	case "jira.token":
		cfg.Jira.Token = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

func isValidConfigKey(key string) bool {
	switch key {
	case "worklog_file", "tempo.base_url", "tempo.token", "tempo.account_id",
		"jira.base_url", "jira.token":
		return true
	default:
		return false
	}
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func resolvePassSecret(key string) (string, error) {
	cmd := exec.Command("pass", key)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pass %s: %w", key, err)
	}
	return strings.TrimSpace(string(out)), nil
}
