package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete HookCraft configuration.
type Config struct {
	Version  int                   `yaml:"version"`
	Settings Settings              `yaml:"settings"`
	Hooks    map[string][]HookStep `yaml:"hooks"`
}

// Settings defines global execution flags and limits.
type Settings struct {
	Concurrency         bool    `yaml:"concurrency"`
	MaxWorkers          int     `yaml:"max_workers"`
	AutoStageFixes      bool    `yaml:"auto_stage_fixes"`
	InteractiveApproval bool    `yaml:"interactive_approval"`
	BlockSecretLeaks    bool    `yaml:"block_secret_leaks"`
	EntropyThreshold    float64 `yaml:"entropy_threshold"`
	TimeoutSeconds      int     `yaml:"timeout_seconds"`
	StashUnstaged       bool    `yaml:"stash_unstaged"`
	FailFast            bool    `yaml:"fail_fast"`
	MaxFileSizeMB       float64 `yaml:"max_file_size_mb"`
	EnableCache         bool    `yaml:"enable_cache"`
}

// HookStep defines an individual hook step execution.
type HookStep struct {
	Name              string   `yaml:"name"`
	Type              string   `yaml:"type"`               // "command", "builtin:secret_shield", "builtin:file_size_shield", "builtin:branch_protection", "builtin:conventional_commits"
	Glob              string   `yaml:"glob"`               // e.g. "*.{ts,tsx,js}", "*.go"
	ExcludeGlob       string   `yaml:"exclude_glob"`       // optional exclusion pattern
	Run               string   `yaml:"run"`                // command string with {staged_files}
	StageFixed        bool     `yaml:"stage_fixed"`        // auto-stage modified files
	Action            string   `yaml:"action"`             // "block" (default) or "warn"
	TimeoutSeconds    int      `yaml:"timeout_seconds"`    // step-specific timeout
	Env               []string `yaml:"env"`                // custom environment variables KEY=VALUE
	MaxFileSizeMB     float64  `yaml:"max_file_size_mb"`   // for builtin:file_size_shield
	ProtectedBranches []string `yaml:"protected_branches"` // for builtin:branch_protection
	AllowTags         bool     `yaml:"allow_tags"`         // for builtin:branch_protection
	RequireTicket     bool     `yaml:"require_ticket"`     // for builtin:conventional_commits
	TicketPattern     string   `yaml:"ticket_pattern"`     // regex for JIRA/GitHub issue ticket e.g. "PROJ-[0-9]+"
}

// Standard configuration filenames.
var ConfigFiles = []string{
	".hookcraft.yaml",
	".hookcraft.yml",
	"hookcraft.yaml",
	"hookcraft.yml",
}

// DefaultSettings returns sensible defaults optimized for sub-100ms execution.
func DefaultSettings() Settings {
	return Settings{
		Concurrency:         true,
		MaxWorkers:          runtime.NumCPU(),
		AutoStageFixes:      true,
		InteractiveApproval: false,
		BlockSecretLeaks:    true,
		EntropyThreshold:    3.8,
		TimeoutSeconds:      30,
		StashUnstaged:       true,
		FailFast:            false,
		MaxFileSizeMB:       5.0,
		EnableCache:         true,
	}
}

// LoadConfig loads the configuration from a file in repoRoot or returns auto-detected defaults.
func LoadConfig(repoRoot string, customConfigPath string) (*Config, string, error) {
	var targetFile string

	if customConfigPath != "" {
		if filepath.IsAbs(customConfigPath) {
			targetFile = customConfigPath
		} else {
			targetFile = filepath.Join(repoRoot, customConfigPath)
		}
	} else {
		for _, name := range ConfigFiles {
			p := filepath.Join(repoRoot, name)
			if _, err := os.Stat(p); err == nil {
				targetFile = p
				break
			}
		}
	}

	// If no config file found, synthesize configuration from repository stack
	if targetFile == "" {
		cfg := DetectProjectConfig(repoRoot)
		return cfg, "", nil
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		return nil, targetFile, fmt.Errorf("failed to read configuration file: %w", err)
	}

	cfg := &Config{
		Version:  1,
		Settings: DefaultSettings(),
		Hooks:    make(map[string][]HookStep),
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, targetFile, fmt.Errorf("invalid YAML syntax in %s: %w", targetFile, err)
	}

	// Apply setting fallbacks
	if cfg.Settings.MaxWorkers <= 0 {
		cfg.Settings.MaxWorkers = runtime.NumCPU()
	}
	if cfg.Settings.EntropyThreshold <= 0 {
		cfg.Settings.EntropyThreshold = 3.8
	}
	if cfg.Settings.TimeoutSeconds <= 0 {
		cfg.Settings.TimeoutSeconds = 30
	}

	// Fill defaults in steps
	for hookName, steps := range cfg.Hooks {
		for i := range steps {
			if steps[i].Type == "" {
				steps[i].Type = "command"
			}
			if steps[i].Action == "" {
				steps[i].Action = "block"
			}
			if steps[i].TimeoutSeconds <= 0 {
				steps[i].TimeoutSeconds = cfg.Settings.TimeoutSeconds
			}
		}
		cfg.Hooks[hookName] = steps
	}

	return cfg, targetFile, nil
}

// GetTimeout returns the duration for this hook step.
func (h *HookStep) GetTimeout(defaultTimeout time.Duration) time.Duration {
	if h.TimeoutSeconds > 0 {
		return time.Duration(h.TimeoutSeconds) * time.Second
	}
	return defaultTimeout
}
