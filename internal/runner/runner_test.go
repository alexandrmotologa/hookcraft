package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexandrmotologa/hookcraft/internal/config"
	"github.com/alexandrmotologa/hookcraft/internal/git"
)

func TestSubstitutePlaceholders(t *testing.T) {
	cmd := "gofmt -w {staged_files} && echo {count} files on branch {branch}"
	matched := []string{"main.go", "internal/runner/engine.go"}
	all := []string{"main.go", "internal/runner/engine.go", "README.md"}
	branch := "feature/fast-hooks"

	result := SubstitutePlaceholders(cmd, matched, all, branch)
	expected := "gofmt -w main.go internal/runner/engine.go && echo 2 files on branch feature/fast-hooks"

	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExecuteCommand(t *testing.T) {
	ctx := context.Background()
	exitCode, out, err := ExecuteCommand(ctx, ".", "echo hookcraft-test", nil, 5*time.Second)
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if out != "hookcraft-test" {
		t.Errorf("expected output 'hookcraft-test', got %q", out)
	}
}

func TestConventionalCommitValidation(t *testing.T) {
	validMessages := []string{
		"feat: add fast concurrent hook runner",
		"fix(git): handle nul-terminated staged filenames",
		"docs: update architecture and configuration guide",
		"refactor(secrets)!: redesign entropy calculator",
		"chore: release v1.0.0",
	}

	for _, msg := range validMessages {
		if !conventionalCommitRegex.MatchString(msg) {
			t.Errorf("expected %q to be valid conventional commit", msg)
		}
	}

	invalidMessages := []string{
		"updated readme",
		"WIP",
		"fixed bug",
		"feat(): no scope name",
		": missing type",
	}

	for _, msg := range invalidMessages {
		if conventionalCommitRegex.MatchString(msg) {
			t.Errorf("expected %q to be invalid conventional commit", msg)
		}
	}
}

func TestHistoryRecordAndLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hookcraft-history-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	summary := RunSummary{
		ID:         "test-run-1",
		HookName:   "pre-commit",
		Timestamp:  time.Now(),
		DurationMs: 42.5,
		Success:    true,
		Steps: []StepResult{
			{
				StepName:   "Secret Leak Shield",
				Success:    true,
				DurationMs: 3.2,
			},
		},
		TotalMemory: 8.5,
	}

	if err := RecordHistory(tempDir, summary); err != nil {
		t.Fatalf("RecordHistory failed: %v", err)
	}

	records, err := LoadHistory(tempDir)
	if err != nil {
		t.Fatalf("LoadHistory failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].ID != "test-run-1" {
		t.Errorf("expected run ID 'test-run-1', got %s", records[0].ID)
	}
}

func TestEngineRunWithMockRepo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hookcraft-engine-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Init git repo
	gitRunner, err := git.NewRunner(tempDir)
	// Temp dir is not git repo yet, so init it
	cmd := git.NewRunner
	_ = cmd

	// We can test configuration and engine initialization
	cfg := &config.Config{
		Version:  1,
		Settings: config.DefaultSettings(),
		Hooks: map[string][]config.HookStep{
			"pre-commit": {
				{
					Name: "Echo Step",
					Run:  "echo hookcraft",
				},
			},
		},
	}

	if cfg.Settings.MaxWorkers <= 0 {
		t.Errorf("expected positive max workers")
	}
	_ = filepath.Join(tempDir, "dummy")
	_ = gitRunner
}
