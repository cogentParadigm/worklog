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
}

type TempoConfig struct {
	BaseURL   string `yaml:"base_url"`
	Token     string `yaml:"token"`
	AccountID string `yaml:"account_id"`
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

func configPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "worklog", "config.yaml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "worklog", "config.yaml")
	}
	return filepath.Join(home, ".config", "worklog", "config.yaml")
}

func resolvePassSecret(key string) (string, error) {
	cmd := exec.Command("pass", key)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pass %s: %w", key, err)
	}
	return strings.TrimSpace(string(out)), nil
}
