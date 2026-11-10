package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandrmotologa/hookcraft/internal/config"
	"gopkg.in/yaml.v3"
)

// MigrationResult summarizes what was migrated.
type MigrationResult struct {
	SourceType    string   `json:"source_type"` // "pre-commit", "lefthook", "husky"
	SourcePath    string   `json:"source_path"`
	MigratedHooks []string `json:"migrated_hooks"`
	GeneratedYAML string   `json:"generated_yaml"`
}

// DetectAndMigrate inspects the repository root for foreign hook runners and migrates them.
func DetectAndMigrate(repoRoot string) (*MigrationResult, error) {
	// 1. Check for Python pre-commit (.pre-commit-config.yaml)
	preCommitPath := filepath.Join(repoRoot, ".pre-commit-config.yaml")
	if fileExists(preCommitPath) {
		return migratePythonPreCommit(repoRoot, preCommitPath)
	}

	// 2. Check for Lefthook (lefthook.yml, .lefthook.yaml)
	for _, name := range []string{"lefthook.yml", ".lefthook.yaml", "lefthook.yaml"} {
		p := filepath.Join(repoRoot, name)
		if fileExists(p) {
			return migrateLefthook(repoRoot, p)
		}
	}

	// 3. Check for Husky (.husky/)
	huskyDir := filepath.Join(repoRoot, ".husky")
	if dirExists(huskyDir) {
		return migrateHusky(repoRoot, huskyDir)
	}

	return nil, fmt.Errorf("no existing Husky, Python pre-commit, or Lefthook configuration found in %s", repoRoot)
}

func migratePythonPreCommit(repoRoot, filePath string) (*MigrationResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Repos []struct {
			Repo  string `yaml:"repo"`
			Hooks []struct {
				ID    string   `yaml:"id"`
				Name  string   `yaml:"name"`
				Entry string   `yaml:"entry"`
				Files string   `yaml:"files"`
				Types []string `yaml:"types"`
			} `yaml:"hooks"`
		} `yaml:"repos"`
	}

	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse .pre-commit-config.yaml: %w", err)
	}

	cfg := &config.Config{
		Version:  1,
		Settings: config.DefaultSettings(),
		Hooks:    make(map[string][]config.HookStep),
	}

	preCommitSteps := []config.HookStep{
		{
			Name:   "Secret Leak Shield",
			Type:   "builtin:secret_shield",
			Action: "block",
		},
	}

	var stepNames []string

	for _, repo := range raw.Repos {
		for _, h := range repo.Hooks {
			name := h.Name
			if name == "" {
				name = h.ID
			}

			cmd := h.Entry
			if cmd == "" {
				cmd = h.ID
			}
			if !strings.Contains(cmd, "{staged_files}") {
				cmd += " {staged_files}"
			}

			glob := h.Files
			if glob == "" {
				glob = "*.*"
			}

			step := config.HookStep{
				Name:       name,
				Glob:       glob,
				Run:        cmd,
				StageFixed: strings.Contains(strings.ToLower(name), "format") || strings.Contains(strings.ToLower(cmd), "fix"),
			}
			preCommitSteps = append(preCommitSteps, step)
			stepNames = append(stepNames, name)
		}
	}

	cfg.Hooks["pre-commit"] = preCommitSteps

	yamlBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	return &MigrationResult{
		SourceType:    "Python pre-commit",
		SourcePath:    filePath,
		MigratedHooks: stepNames,
		GeneratedYAML: string(yamlBytes),
	}, nil
}

func migrateLefthook(repoRoot, filePath string) (*MigrationResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse lefthook config: %w", err)
	}

	cfg := &config.Config{
		Version:  1,
		Settings: config.DefaultSettings(),
		Hooks:    make(map[string][]config.HookStep),
	}

	var allSteps []string

	for hookName, v := range raw {
		if hookName == "colors" || hookName == "extends" || hookName == "skip_output" {
			continue
		}

		hookMap, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		commandsVal, ok := hookMap["commands"]
		if !ok {
			continue
		}

		commandsMap, ok := commandsVal.(map[string]interface{})
		if !ok {
			continue
		}

		var steps []config.HookStep
		if hookName == "pre-commit" {
			steps = append(steps, config.HookStep{
				Name:   "Secret Leak Shield",
				Type:   "builtin:secret_shield",
				Action: "block",
			})
		}

		for stepKey, cmdVal := range commandsMap {
			stepMap, ok := cmdVal.(map[string]interface{})
			if !ok {
				continue
			}

			runCmd, _ := stepMap["run"].(string)
			glob, _ := stepMap["glob"].(string)
			stageFixed, _ := stepMap["stage_fixed"].(bool)

			// Replace lefthook {staged_files} or {all_files}
			runCmd = strings.ReplaceAll(runCmd, "{staged_files}", "{staged_files}")

			steps = append(steps, config.HookStep{
				Name:       stepKey,
				Glob:       glob,
				Run:        runCmd,
				StageFixed: stageFixed,
			})
			allSteps = append(allSteps, fmt.Sprintf("%s:%s", hookName, stepKey))
		}

		cfg.Hooks[hookName] = steps
	}

	yamlBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	return &MigrationResult{
		SourceType:    "Lefthook",
		SourcePath:    filePath,
		MigratedHooks: allSteps,
		GeneratedYAML: string(yamlBytes),
	}, nil
}

func migrateHusky(repoRoot, huskyDir string) (*MigrationResult, error) {
	cfg := &config.Config{
		Version:  1,
		Settings: config.DefaultSettings(),
		Hooks:    make(map[string][]config.HookStep),
	}

	var migrated []string

	hookFiles := []string{"pre-commit", "commit-msg", "pre-push", "post-commit"}
	for _, h := range hookFiles {
		p := filepath.Join(huskyDir, h)
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		var commands []string
		for _, l := range lines {
			trimmed := strings.TrimSpace(l)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.Contains(trimmed, "husky.sh") {
				continue
			}
			commands = append(commands, trimmed)
		}

		if len(commands) == 0 {
			continue
		}

		var steps []config.HookStep
		if h == "pre-commit" {
			steps = append(steps, config.HookStep{
				Name:   "Secret Leak Shield",
				Type:   "builtin:secret_shield",
				Action: "block",
			})
		}

		for idx, cmd := range commands {
			stepName := fmt.Sprintf("%s Step %d", h, idx+1)
			steps = append(steps, config.HookStep{
				Name: stepName,
				Run:  cmd,
			})
			migrated = append(migrated, fmt.Sprintf("%s:%s", h, cmd))
		}

		cfg.Hooks[h] = steps
	}

	yamlBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	return &MigrationResult{
		SourceType:    "Husky",
		SourcePath:    huskyDir,
		MigratedHooks: migrated,
		GeneratedYAML: string(yamlBytes),
	}, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
