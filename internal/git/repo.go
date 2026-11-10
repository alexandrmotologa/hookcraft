package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrNotGitRepo = errors.New("not a git repository")
	ErrGitTimeout = errors.New("git command timed out")
)

// Runner executes git commands within a repository.
type Runner struct {
	RepoRoot string
	GitDir   string
}

// NewRunner creates a git runner for the specified path or current working directory.
func NewRunner(path string) (*Runner, error) {
	root, err := FindRepoRoot(path)
	if err != nil {
		return nil, err
	}

	gitDir, err := FindGitDir(root)
	if err != nil {
		return nil, err
	}

	return &Runner{
		RepoRoot: root,
		GitDir:   gitDir,
	}, nil
}

// FindRepoRoot determines the root directory of the git repository.
func FindRepoRoot(startDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if startDir != "" {
		cmd.Dir = startDir
	}
	out, err := cmd.Output()
	if err != nil {
		return "", ErrNotGitRepo
	}
	return filepath.Clean(strings.TrimSpace(string(out))), nil
}

// FindGitDir determines the actual .git directory (supporting worktrees and submodules).
func FindGitDir(repoRoot string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	if repoRoot != "" {
		cmd.Dir = repoRoot
	}
	out, err := cmd.Output()
	if err != nil {
		return "", ErrNotGitRepo
	}
	rawDir := strings.TrimSpace(string(out))
	if filepath.IsAbs(rawDir) {
		return filepath.Clean(rawDir), nil
	}
	return filepath.Clean(filepath.Join(repoRoot, rawDir)), nil
}

// Exec runs a git command with a standard timeout of 15 seconds.
func (r *Runner) Exec(args ...string) (string, error) {
	return r.ExecWithTimeout(15*time.Second, args...)
}

// ExecWithTimeout executes a git command in the repository root with a specified timeout.
func (r *Runner) ExecWithTimeout(timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.RepoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", ErrGitTimeout
	}
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), errMsg)
	}

	return stdout.String(), nil
}

// ExecBytes runs a git command returning raw bytes.
func (r *Runner) ExecBytes(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.RepoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, ErrGitTimeout
	}
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("git %s failed: %s", strings.Join(args, " "), errMsg)
	}

	return stdout.Bytes(), nil
}
