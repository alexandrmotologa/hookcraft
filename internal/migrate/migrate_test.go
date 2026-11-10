package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigratePythonPreCommit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hookcraft-migrate-pc-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	preCommitYAML := `
repos:
- repo: https://github.com/psf/black
  hooks:
  - id: black
    name: "Black Formatter"
    entry: black
    files: '\.py$'
`
	configPath := filepath.Join(tempDir, ".pre-commit-config.yaml")
	_ = os.WriteFile(configPath, []byte(preCommitYAML), 0644)

	res, err := DetectAndMigrate(tempDir)
	if err != nil {
		t.Fatalf("DetectAndMigrate failed: %v", err)
	}

	if res.SourceType != "Python pre-commit" {
		t.Errorf("expected SourceType 'Python pre-commit', got %s", res.SourceType)
	}

	if len(res.MigratedHooks) != 1 || res.MigratedHooks[0] != "Black Formatter" {
		t.Errorf("unexpected migrated hooks: %v", res.MigratedHooks)
	}

	if !strings.Contains(res.GeneratedYAML, "Black Formatter") {
		t.Errorf("expected generated YAML to contain 'Black Formatter', got:\n%s", res.GeneratedYAML)
	}
}

func TestMigrateLefthook(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hookcraft-migrate-left-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	lefthookYAML := `
pre-commit:
  commands:
    gofmt:
      run: gofmt -w {staged_files}
      glob: "*.go"
      stage_fixed: true
    linter:
      run: golangci-lint run {staged_files}
      glob: "*.go"
`
	configPath := filepath.Join(tempDir, "lefthook.yml")
	_ = os.WriteFile(configPath, []byte(lefthookYAML), 0644)

	res, err := DetectAndMigrate(tempDir)
	if err != nil {
		t.Fatalf("DetectAndMigrate failed: %v", err)
	}

	if res.SourceType != "Lefthook" {
		t.Errorf("expected SourceType 'Lefthook', got %s", res.SourceType)
	}

	if len(res.MigratedHooks) != 2 {
		t.Errorf("expected 2 migrated commands, got %d: %v", len(res.MigratedHooks), res.MigratedHooks)
	}
}

func TestMigrateHusky(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hookcraft-migrate-husky-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	huskyDir := filepath.Join(tempDir, ".husky")
	_ = os.MkdirAll(huskyDir, 0755)

	preCommitScript := `#!/usr/bin/env sh
. "$(dirname -- "$0")/_/husky.sh"

npm run lint
npx prettier --write .
`
	_ = os.WriteFile(filepath.Join(huskyDir, "pre-commit"), []byte(preCommitScript), 0755)

	res, err := DetectAndMigrate(tempDir)
	if err != nil {
		t.Fatalf("DetectAndMigrate failed: %v", err)
	}

	if res.SourceType != "Husky" {
		t.Errorf("expected SourceType 'Husky', got %s", res.SourceType)
	}

	if len(res.MigratedHooks) != 2 {
		t.Errorf("expected 2 migrated commands from Husky, got %d: %v", len(res.MigratedHooks), res.MigratedHooks)
	}
}
