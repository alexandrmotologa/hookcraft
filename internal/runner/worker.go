package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/alexandrmotologa/hookcraft/internal/secrets"
)

// StepResult captures the execution outcome of an individual hook step.
type StepResult struct {
	StepName       string            `json:"step_name"`
	Type           string            `json:"type"`
	Command        string            `json:"command"`
	Duration       time.Duration     `json:"duration_ns"`
	DurationMs     float64           `json:"duration_ms"`
	ExitCode       int               `json:"exit_code"`
	Success        bool              `json:"success"`
	Cached         bool              `json:"cached"`
	Output         string            `json:"output"`
	MatchedFiles   []string          `json:"matched_files"`
	ModifiedFiles  []string          `json:"modified_files"`
	Skipped        bool              `json:"skipped"`
	SkipReason     string            `json:"skip_reason"`
	SecretFindings []secrets.Finding `json:"secret_findings,omitempty"`
}

// SubstitutePlaceholders replaces template variables like {staged_files} in the command string.
func SubstitutePlaceholders(cmdStr string, matchedFiles []string, allFiles []string, branchName string) string {
	quotedMatched := make([]string, len(matchedFiles))
	for i, f := range matchedFiles {
		// Quote files containing spaces
		if strings.Contains(f, " ") {
			quotedMatched[i] = fmt.Sprintf("%q", f)
		} else {
			quotedMatched[i] = f
		}
	}

	stagedFilesStr := strings.Join(quotedMatched, " ")
	allFilesStr := strings.Join(allFiles, " ")

	res := strings.ReplaceAll(cmdStr, "{staged_files}", stagedFilesStr)
	res = strings.ReplaceAll(res, "{all_files}", allFilesStr)
	res = strings.ReplaceAll(res, "{count}", fmt.Sprintf("%d", len(matchedFiles)))
	res = strings.ReplaceAll(res, "{branch}", branchName)

	return res
}

// ExecuteCommand runs a shell command with the given working directory and timeout.
func ExecuteCommand(ctx context.Context, repoRoot string, commandStr string, env []string, timeout time.Duration) (int, string, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Use cmd.exe /C or powershell -Command
		cmd = exec.CommandContext(ctxTimeout, "cmd.exe", "/C", commandStr)
	} else {
		cmd = exec.CommandContext(ctxTimeout, "sh", "-c", commandStr)
	}

	cmd.Dir = repoRoot
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), env...)
	}

	var combinedOutput bytes.Buffer
	cmd.Stdout = &combinedOutput
	cmd.Stderr = &combinedOutput

	err := cmd.Run()
	outputStr := strings.TrimSpace(combinedOutput.String())

	if ctxTimeout.Err() == context.DeadlineExceeded {
		return 124, outputStr, fmt.Errorf("command timed out after %v", timeout)
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), outputStr, nil
		}
		return 1, outputStr, err
	}

	return 0, outputStr, nil
}
