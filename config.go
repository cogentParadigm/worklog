package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WorklogFile string      `yaml:"worklog_file"`
	Tempo       TempoConfig `yaml:"tempo"`
	Jira        JiraConfig  `yaml:"jira"`
}

type RoundingStep struct {
	Step string `yaml:"step"`
	To   string `yaml:"to"`
}

type TempoConfig struct {
	BaseURL    string            `yaml:"base_url"`
	Token      string            `yaml:"token"`
	AccountID  string            `yaml:"account_id"`
	Rounding   []RoundingStep    `yaml:"rounding,omitempty"`
	Attributes map[string]string `yaml:"attributes,omitempty"`
}

type JiraConfig struct {
	BaseURL  string `yaml:"base_url"`
	Username string `yaml:"username"`
	Token    string `yaml:"token"`
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

var sensitiveConfigKeys = map[string]bool{
	"tempo.token": true,
	"jira.token":  true,
}

func (cfg *Config) Get(key string) (string, error) {
	val, err := cfg.GetUnmasked(key)
	if err != nil {
		return "", err
	}
	if sensitiveConfigKeys[key] && val != "" {
		return "<hidden>", nil
	}
	return val, nil
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
	case "tempo.rounding":
		return roundingStepsString(cfg.Tempo.Rounding), nil
	case "tempo.attributes":
		return tempoAttributesString(cfg.Tempo.Attributes), nil
	case "jira.base_url":
		return cfg.Jira.BaseURL, nil
	case "jira.username":
		return cfg.Jira.Username, nil
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
	case "tempo.rounding":
		steps, err := parseRoundingSteps(value)
		if err != nil {
			return err
		}
		cfg.Tempo.Rounding = steps
	case "tempo.attributes":
		attrs, err := parseTempoAttributes(value)
		if err != nil {
			return err
		}
		cfg.Tempo.Attributes = attrs
	case "jira.base_url":
		cfg.Jira.BaseURL = value
	case "jira.username":
		cfg.Jira.Username = value
	case "jira.token":
		cfg.Jira.Token = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

func isValidConfigKey(key string) bool {
	switch key {
	case "worklog_file", "tempo.base_url", "tempo.token", "tempo.account_id", "tempo.rounding", "tempo.attributes",
		"jira.base_url", "jira.username", "jira.token":
		return true
	default:
		return false
	}
}

func parseTempoAttributes(s string) (map[string]string, error) {
	if s == "" {
		return nil, nil
	}
	attrs := make(map[string]string)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		eqIdx := strings.Index(part, "=")
		if eqIdx < 0 {
			return nil, fmt.Errorf("invalid tempo attribute %q: expected format key=value", part)
		}
		key := strings.TrimSpace(part[:eqIdx])
		value := strings.TrimSpace(part[eqIdx+1:])
		if key == "" {
			return nil, fmt.Errorf("invalid tempo attribute %q: missing key", part)
		}
		attrs[key] = value
	}
	return attrs, nil
}

func tempoAttributesString(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}
	var parts []string
	for k, v := range attrs {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
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

func parseRoundingSteps(s string) ([]RoundingStep, error) {
	if s == "" {
		return nil, nil
	}
	var steps []RoundingStep
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		colonIdx := strings.Index(part, ":")
		if colonIdx < 0 {
			return nil, fmt.Errorf("invalid rounding step %q: expected format step:to (e.g. floor:1m)", part)
		}
		step := strings.ToLower(strings.TrimSpace(part[:colonIdx]))
		to := strings.TrimSpace(part[colonIdx+1:])
		if to == "" {
			return nil, fmt.Errorf("invalid rounding step %q: missing duration", part)
		}
		if step != "floor" && step != "ceil" && step != "round" {
			return nil, fmt.Errorf("invalid rounding step %q: step must be floor, ceil, or round", part)
		}
		if _, err := parseDurationFlag(to); err != nil {
			return nil, fmt.Errorf("invalid rounding step %q: %w", part, err)
		}
		steps = append(steps, RoundingStep{Step: step, To: to})
	}
	return steps, nil
}

func roundingStepsString(steps []RoundingStep) string {
	if len(steps) == 0 {
		return ""
	}
	var parts []string
	for _, s := range steps {
		parts = append(parts, fmt.Sprintf("%s:%s", s.Step, s.To))
	}
	return strings.Join(parts, ",")
}
