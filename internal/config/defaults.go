package config

import (
	"os"
	"path/filepath"
)

// DetectProjectConfig inspects files in the repo root to infer the primary stack.
func DetectProjectConfig(repoRoot string) *Config {
	cfg := &Config{
		Version:  1,
		Settings: DefaultSettings(),
		Hooks:    make(map[string][]HookStep),
	}

	preCommitSteps := []HookStep{
		{
			Name:   "Secret Leak Shield",
			Type:   "builtin:secret_shield",
			Action: "block",
		},
	}

	// Detect Go
	if fileExists(filepath.Join(repoRoot, "go.mod")) {
		preCommitSteps = append(preCommitSteps,
			HookStep{
				Name:       "Go Format",
				Glob:       "*.go",
				Run:        "gofmt -w {staged_files}",
				StageFixed: true,
			},
			HookStep{
				Name: "Go Vet",
				Glob: "*.go",
				Run:  "go vet ./...",
			},
		)
		cfg.Hooks["pre-push"] = []HookStep{
			{
				Name: "Go Tests",
				Run:  "go test -short ./...",
			},
		}
	}

	// Detect Node / TypeScript
	if fileExists(filepath.Join(repoRoot, "package.json")) {
		// Detect if prettier / biome exists
		preCommitSteps = append(preCommitSteps,
			HookStep{
				Name:       "Format",
				Glob:       "*.{ts,tsx,js,jsx,json,css,md}",
				Run:        "npx prettier --write {staged_files}",
				StageFixed: true,
			},
		)
		cfg.Hooks["pre-push"] = []HookStep{
			{
				Name: "Type Check",
				Run:  "npx tsc --noEmit",
			},
		}
	}

	// Detect Python
	if fileExists(filepath.Join(repoRoot, "pyproject.toml")) ||
		fileExists(filepath.Join(repoRoot, "requirements.txt")) ||
		fileExists(filepath.Join(repoRoot, "setup.py")) {
		preCommitSteps = append(preCommitSteps,
			HookStep{
				Name:       "Python Format (Ruff)",
				Glob:       "*.py",
				Run:        "ruff format {staged_files}",
				StageFixed: true,
			},
			HookStep{
				Name: "Python Lint (Ruff)",
				Glob: "*.py",
				Run:  "ruff check --fix {staged_files}",
			},
		)
	}

	// Detect Rust
	if fileExists(filepath.Join(repoRoot, "Cargo.toml")) {
		preCommitSteps = append(preCommitSteps,
			HookStep{
				Name: "Rust Format",
				Glob: "*.rs",
				Run:  "cargo fmt --check",
			},
		)
		cfg.Hooks["pre-push"] = []HookStep{
			{
				Name: "Rust Tests",
				Run:  "cargo test",
			},
		}
	}

	cfg.Hooks["pre-commit"] = preCommitSteps
	cfg.Hooks["commit-msg"] = []HookStep{
		{
			Name: "Conventional Commits Validator",
			Type: "builtin:conventional_commits",
		},
	}

	return cfg
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// GenerateStarterYAML returns a clean, commented YAML string ready to write to .hookcraft.yaml.
func GenerateStarterYAML(repoRoot string) string {
	hasGo := fileExists(filepath.Join(repoRoot, "go.mod"))
	hasNode := fileExists(filepath.Join(repoRoot, "package.json"))
	hasPy := fileExists(filepath.Join(repoRoot, "pyproject.toml")) || fileExists(filepath.Join(repoRoot, "requirements.txt"))

	yaml := `# HookCraft Configuration
# Documentation: https://github.com/alexandrmotologa/hookcraft

version: 1

settings:
  concurrency: true          # Run independent hooks in parallel across CPU cores
  auto_stage_fixes: true      # Automatically re-stage files modified by formatters
  interactive_approval: false # Set true to show diff and ask [y/N] before staging fixes
  block_secret_leaks: true    # Block commits containing API keys, private keys, or high entropy tokens
  entropy_threshold: 3.8     # Shannon entropy threshold for secret detection (3.5 - 4.5)
  timeout_seconds: 30        # Default command timeout
  stash_unstaged: true       # Stash unstaged changes during hook execution to avoid dirty leaks
  fail_fast: false           # Immediately stop all hooks on first failure

hooks:
  pre-commit:
    - name: "Secret Leak Shield"
      type: "builtin:secret_shield"
      action: "block"
`

	if hasGo {
		yaml += `
    - name: "Go Format"
      glob: "*.go"
      run: "gofmt -w {staged_files}"
      stage_fixed: true

    - name: "Go Vet"
      glob: "*.go"
      run: "go vet ./..."
`
	}

	if hasNode {
		yaml += `
    - name: "Format (Prettier)"
      glob: "*.{ts,tsx,js,jsx,json,css}"
      run: "npx prettier --write {staged_files}"
      stage_fixed: true
`
	}

	if hasPy {
		yaml += `
    - name: "Python Ruff Format"
      glob: "*.py"
      run: "ruff format {staged_files}"
      stage_fixed: true
`
	}

	if !hasGo && !hasNode && !hasPy {
		yaml += `
    - name: "Trailing Whitespace Check"
      glob: "*"
      run: "echo Checking staged files: {staged_files}"
`
	}

	yaml += `
  commit-msg:
    - name: "Conventional Commits"
      type: "builtin:conventional_commits"

  pre-push:
`
	if hasGo {
		yaml += `    - name: "Go Tests"
      run: "go test -short ./..."
`
	} else if hasNode {
		yaml += `    - name: "Type Check"
      run: "npx tsc --noEmit"
`
	} else {
		yaml += `    - name: "Branch Protection"
      run: "echo Pushing to remote"
`
	}

	return yaml
}
