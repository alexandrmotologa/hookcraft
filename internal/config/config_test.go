package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern  string
		path     string
		expected bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "internal/git/repo.go", true},
		{"*.{ts,tsx,js}", "src/index.tsx", true},
		{"*.{ts,tsx,js}", "src/utils.ts", true},
		{"*.{ts,tsx,js}", "index.js", true},
		{"*.{ts,tsx,js}", "main.go", false},
		{"*.py", "scripts/test.py", true},
		{"*.py", "scripts/test.pyc", false},
		{"*", "anything.txt", true},
		{"", "anything.txt", true},
	}

	for _, tc := range tests {
		got := MatchGlob(tc.pattern, tc.path)
		if got != tc.expected {
			t.Errorf("MatchGlob(%q, %q) = %v, expected %v", tc.pattern, tc.path, got, tc.expected)
		}
	}
}

func TestFilterFilesByGlob(t *testing.T) {
	files := []string{
		"main.go",
		"internal/repo.go",
		"src/App.tsx",
		"README.md",
		"vendor/dep.go",
	}

	matched := FilterFilesByGlob(files, "*.go", "vendor/*")
	expected := []string{"main.go", "internal/repo.go"}

	if len(matched) != len(expected) {
		t.Fatalf("expected %d matched files, got %d: %v", len(expected), len(matched), matched)
	}
}

func TestLoadConfigCustom(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hookcraft-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configYAML := `version: 1
settings:
  concurrency: true
  max_workers: 4
  entropy_threshold: 4.2
hooks:
  pre-commit:
    - name: "Linter"
      glob: "*.go"
      run: "golangci-lint run"
`
	configPath := filepath.Join(tempDir, ".hookcraft.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, path, err := LoadConfig(tempDir, "")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if path != configPath {
		t.Errorf("expected path %s, got %s", configPath, path)
	}

	if cfg.Settings.MaxWorkers != 4 {
		t.Errorf("expected max_workers 4, got %d", cfg.Settings.MaxWorkers)
	}

	if cfg.Settings.EntropyThreshold != 4.2 {
		t.Errorf("expected entropy_threshold 4.2, got %f", cfg.Settings.EntropyThreshold)
	}

	if len(cfg.Hooks["pre-commit"]) != 1 {
		t.Fatalf("expected 1 pre-commit step, got %d", len(cfg.Hooks["pre-commit"]))
	}
}

func TestDetectProjectConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hookcraft-detect-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create go.mod
	_ = os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module test"), 0644)

	cfg := DetectProjectConfig(tempDir)
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	steps := cfg.Hooks["pre-commit"]
	if len(steps) < 2 {
		t.Fatalf("expected at least 2 pre-commit steps for Go project, got %d", len(steps))
	}

	foundGoFormat := false
	for _, s := range steps {
		if s.Name == "Go Format" {
			foundGoFormat = true
		}
	}
	if !foundGoFormat {
		t.Errorf("expected 'Go Format' step in detected Go config")
	}
}
