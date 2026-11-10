package runner

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/hookcraft/internal/cache"
	"github.com/alexandrmotologa/hookcraft/internal/config"
	"github.com/alexandrmotologa/hookcraft/internal/git"
	"github.com/alexandrmotologa/hookcraft/internal/secrets"
)

var conventionalCommitRegex = regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-zA-Z0-9_\-\./]+\))?(!)?:\s+[^\s].*`)

// Engine orchestrates concurrent hook execution.
type Engine struct {
	Config  *config.Config
	Git     *git.Runner
	Scanner *secrets.Scanner
	Cache   *cache.Manager
}

// NewEngine creates an execution engine.
func NewEngine(cfg *config.Config, gitRunner *git.Runner) *Engine {
	scanner := secrets.NewScanner(cfg.Settings.EntropyThreshold)
	var cacheMgr *cache.Manager
	if gitRunner != nil {
		scanner.SetIgnorePatterns(secrets.LoadIgnorePatterns(gitRunner.RepoRoot))
		cacheMgr = cache.NewManager(gitRunner.RepoRoot)
	}
	return &Engine{
		Config:  cfg,
		Git:     gitRunner,
		Scanner: scanner,
		Cache:   cacheMgr,
	}
}

// Run executes the specified hook name (e.g. "pre-commit", "commit-msg", "pre-push").
func (e *Engine) Run(ctx context.Context, hookName string, hookArgs []string) (*RunSummary, error) {
	startTime := time.Now()
	summary := &RunSummary{
		ID:        fmt.Sprintf("run-%d", startTime.UnixNano()),
		HookName:  hookName,
		Timestamp: startTime,
		Steps:     make([]StepResult, 0),
		Success:   true,
	}

	steps, ok := e.Config.Hooks[hookName]
	if !ok || len(steps) == 0 {
		// Nothing to run for this hook
		summary.DurationMs = float64(time.Since(startTime).Microseconds()) / 1000.0
		return summary, nil
	}

	// 1. Stash Guard for pre-commit
	var stashGuard *git.StashGuard
	if hookName == "pre-commit" && e.Config.Settings.StashUnstaged {
		stashGuard = git.NewStashGuard(e.Git)
		_, err := stashGuard.StashUnstaged()
		if err != nil {
			return nil, fmt.Errorf("stash guard failed: %w", err)
		}
		defer func() {
			_ = stashGuard.Restore()
		}()
	}

	// 2. Fetch Staged Files
	stagedFiles, err := e.Git.GetStagedFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve staged files: %w", err)
	}
	summary.StagedFiles = stagedFiles

	// Get current branch
	branchName, _ := e.Git.Exec("rev-parse", "--abbrev-ref", "HEAD")
	branchName = strings.TrimSpace(branchName)

	// 3. Execution Pipeline
	if e.Config.Settings.Concurrency && len(steps) > 1 && hookName != "commit-msg" {
		summary.Steps = e.runConcurrent(ctx, steps, stagedFiles, branchName, hookArgs)
	} else {
		summary.Steps = e.runSequential(ctx, steps, stagedFiles, branchName, hookArgs)
	}

	// Check overall success
	for _, res := range summary.Steps {
		if !res.Success && !res.Skipped {
			summary.Success = false
			break
		}
	}

	// 4. Auto-Staging Formatted Files
	if summary.Success && hookName == "pre-commit" && e.Config.Settings.AutoStageFixes {
		var allModified []string
		for _, res := range summary.Steps {
			if len(res.ModifiedFiles) > 0 {
				allModified = append(allModified, res.ModifiedFiles...)
			}
		}
		if len(allModified) > 0 {
			_ = e.Git.StageFiles(allModified...)
		}
	}

	// Record memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	summary.TotalMemory = float64(m.Alloc) / (1024.0 * 1024.0)

	summary.DurationMs = float64(time.Since(startTime).Microseconds()) / 1000.0

	// Save history log
	_ = RecordHistory(e.Git.RepoRoot, *summary)

	if !summary.Success {
		return summary, fmt.Errorf("hook %s failed", hookName)
	}

	return summary, nil
}

// runSequential runs steps one after another.
func (e *Engine) runSequential(ctx context.Context, steps []config.HookStep, stagedFiles []string, branchName string, hookArgs []string) []StepResult {
	results := make([]StepResult, 0, len(steps))

	for _, step := range steps {
		res := e.executeStep(ctx, step, stagedFiles, branchName, hookArgs)
		results = append(results, res)

		if !res.Success && e.Config.Settings.FailFast {
			break
		}
	}

	return results
}

// runConcurrent runs steps in parallel using a goroutine worker pool.
func (e *Engine) runConcurrent(ctx context.Context, steps []config.HookStep, stagedFiles []string, branchName string, hookArgs []string) []StepResult {
	maxWorkers := e.Config.Settings.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	results := make([]StepResult, len(steps))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	for i, step := range steps {
		wg.Add(1)
		go func(idx int, s config.HookStep) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			results[idx] = e.executeStep(ctx, s, stagedFiles, branchName, hookArgs)
		}(i, step)
	}

	wg.Wait()
	return results
}

// executeStep processes a single hook step (builtin or external shell command).
func (e *Engine) executeStep(ctx context.Context, step config.HookStep, stagedFiles []string, branchName string, hookArgs []string) StepResult {
	start := time.Now()
	res := StepResult{
		StepName: step.Name,
		Type:     step.Type,
		Success:  true,
	}

	// 1. Built-in: Secret Leak Shield
	if step.Type == "builtin:secret_shield" || step.Type == "builtin:secret_scanner" {
		res.Command = "builtin:secret_shield"
		hunks, err := e.Git.GetParsedStagedHunks()
		if err != nil {
			res.Success = false
			res.Output = fmt.Sprintf("Failed to inspect staged diff hunks: %v", err)
			res.Duration = time.Since(start)
			res.DurationMs = float64(res.Duration.Microseconds()) / 1000.0
			return res
		}

		findings := e.Scanner.ScanHunks(hunks)
		res.SecretFindings = findings

		if len(findings) > 0 {
			if step.Action == "warn" {
				res.Success = true
				res.Output = fmt.Sprintf("Warning: detected %d potential secrets in staged changes", len(findings))
			} else {
				res.Success = false
				res.ExitCode = 1
				var b strings.Builder
				b.WriteString(fmt.Sprintf("Secret Leak Shield blocked commit! Found %d secret leak(s):\n", len(findings)))
				for _, f := range findings {
					b.WriteString(fmt.Sprintf("  - %s:%d [%s] %s\n", f.FilePath, f.LineNumber, f.RuleName, f.RedactedValue))
				}
				res.Output = b.String()
			}
		} else {
			res.Output = "No secrets detected in staged diff"
		}

		res.Duration = time.Since(start)
		res.DurationMs = float64(res.Duration.Microseconds()) / 1000.0
		return res
	}

	// 2. Built-in: Conventional Commits validator
	if step.Type == "builtin:conventional_commits" {
		res.Command = "builtin:conventional_commits"
		var commitMsg string

		if len(hookArgs) > 0 {
			// hookArgs[0] is path to .git/COMMIT_EDITMSG
			msgBytes, err := os.ReadFile(hookArgs[0])
			if err == nil {
				commitMsg = strings.TrimSpace(string(msgBytes))
			}
		}

		if commitMsg == "" {
			res.Skipped = true
			res.SkipReason = "no commit message provided to validate"
			res.Duration = time.Since(start)
			res.DurationMs = float64(res.Duration.Microseconds()) / 1000.0
			return res
		}

		// Take first non-comment line
		lines := strings.Split(commitMsg, "\n")
		firstLine := ""
		for _, l := range lines {
			t := strings.TrimSpace(l)
			if t != "" && !strings.HasPrefix(t, "#") {
				firstLine = t
				break
			}
		}

		if !conventionalCommitRegex.MatchString(firstLine) {
			res.Success = false
			res.ExitCode = 1
			res.Output = fmt.Sprintf("Invalid commit message: %q\nDoes not follow Conventional Commits format (e.g. feat(scope): subject, fix: description)", firstLine)
		} else if step.RequireTicket {
			pattern := step.TicketPattern
			if pattern == "" {
				pattern = `[A-Za-z0-9]+-[0-9]+`
			}
			re, err := regexp.Compile(pattern)
			if err == nil && !re.MatchString(firstLine) {
				res.Success = false
				res.ExitCode = 1
				res.Output = fmt.Sprintf("Commit message missing required issue ticket matching %q: %q", pattern, firstLine)
			} else {
				res.Output = fmt.Sprintf("Valid Conventional Commit with ticket: %q", firstLine)
			}
		} else {
			res.Output = fmt.Sprintf("Valid Conventional Commit message: %q", firstLine)
		}

		res.Duration = time.Since(start)
		res.DurationMs = float64(res.Duration.Microseconds()) / 1000.0
		return res
	}

	// 3. Built-in: File Size Shield
	if step.Type == "builtin:file_size_shield" {
		res.Command = "builtin:file_size_shield"
		maxMB := step.MaxFileSizeMB
		if maxMB <= 0 {
			maxMB = e.Config.Settings.MaxFileSizeMB
		}
		if maxMB <= 0 {
			maxMB = 5.0
		}
		maxBytes := int64(maxMB * 1024 * 1024)

		filtered := config.FilterFilesByGlob(stagedFiles, step.Glob, step.ExcludeGlob)
		sizes, err := e.Git.GetStagedFileSizes(filtered)
		if err != nil {
			res.Success = false
			res.Output = fmt.Sprintf("Failed to check staged file sizes: %v", err)
			res.Duration = time.Since(start)
			res.DurationMs = float64(res.Duration.Microseconds()) / 1000.0
			return res
		}

		var oversized []string
		for file, size := range sizes {
			if size > maxBytes {
				fileMB := float64(size) / (1024.0 * 1024.0)
				oversized = append(oversized, fmt.Sprintf("  - %s (%.2f MB) exceeds limit of %.1f MB", file, fileMB, maxMB))
			}
		}

		if len(oversized) > 0 {
			res.Success = false
			res.ExitCode = 1
			res.Output = fmt.Sprintf("File Size Shield blocked commit! Found %d large staged file(s):\n%s", len(oversized), strings.Join(oversized, "\n"))
		} else {
			res.Output = fmt.Sprintf("All staged files are within the %.1f MB size limit", maxMB)
		}

		res.Duration = time.Since(start)
		res.DurationMs = float64(res.Duration.Microseconds()) / 1000.0
		return res
	}

	// 4. Built-in: Branch Protection Shield
	if step.Type == "builtin:branch_protection" {
		res.Command = "builtin:branch_protection"
		protected := step.ProtectedBranches
		if len(protected) == 0 {
			protected = []string{"main", "master"}
		}

		targetBranch := branchName
		for _, arg := range hookArgs {
			for _, p := range protected {
				if strings.Contains(arg, p) {
					targetBranch = p
				}
			}
		}

		isProtected := false
		for _, p := range protected {
			if targetBranch == p || config.MatchGlob(p, targetBranch) {
				isProtected = true
				break
			}
		}

		if isProtected {
			res.Success = false
			res.ExitCode = 1
			res.Output = fmt.Sprintf("Branch Protection Shield blocked push!\nDirect push to protected branch %q is not allowed. Please submit a Pull Request.", targetBranch)
		} else {
			res.Output = fmt.Sprintf("Branch %q is not protected", targetBranch)
		}

		res.Duration = time.Since(start)
		res.DurationMs = float64(res.Duration.Microseconds()) / 1000.0
		return res
	}

	// 5. Command Step
	matchedFiles := config.FilterFilesByGlob(stagedFiles, step.Glob, step.ExcludeGlob)
	res.MatchedFiles = matchedFiles

	// If command targets {staged_files} and no matching files exist, skip gracefully
	if strings.Contains(step.Run, "{staged_files}") && len(matchedFiles) == 0 {
		res.Skipped = true
		res.SkipReason = fmt.Sprintf("no staged files matching pattern %q", step.Glob)
		res.Duration = time.Since(start)
		res.DurationMs = float64(res.Duration.Microseconds()) / 1000.0
		return res
	}

	finalCommand := SubstitutePlaceholders(step.Run, matchedFiles, stagedFiles, branchName)
	res.Command = finalCommand

	// Content-addressable cache check
	var cacheKey string
	if e.Config.Settings.EnableCache && e.Cache != nil && len(matchedFiles) > 0 && !step.StageFixed {
		var err error
		cacheKey, err = cache.ComputeKey(step.Name, finalCommand, matchedFiles, e.Git.RepoRoot)
		if err == nil {
			if entry, ok := e.Cache.Get(cacheKey); ok && entry.ExitCode == 0 {
				res.Cached = true
				res.Success = true
				res.Output = entry.Output
				res.Duration = time.Since(start)
				res.DurationMs = float64(res.Duration.Microseconds()) / 1000.0
				return res
			}
		}
	}

	timeout := step.GetTimeout(time.Duration(e.Config.Settings.TimeoutSeconds) * time.Second)
	exitCode, output, err := ExecuteCommand(ctx, e.Git.RepoRoot, finalCommand, step.Env, timeout)

	res.ExitCode = exitCode
	res.Output = output
	res.Duration = time.Since(start)
	res.DurationMs = float64(res.Duration.Microseconds()) / 1000.0

	if exitCode != 0 || err != nil {
		res.Success = false
		if err != nil && !strings.Contains(output, err.Error()) {
			if res.Output != "" {
				res.Output += "\n"
			}
			res.Output += err.Error()
		}
	} else {
		// Cache successful execution
		if e.Config.Settings.EnableCache && e.Cache != nil && cacheKey != "" && !step.StageFixed {
			e.Cache.Put(cacheKey, step.Name, finalCommand, exitCode, output)
		}

		if step.StageFixed && len(matchedFiles) > 0 {
			// Check if any of the matched files were modified on disk
			modified, _ := e.Git.GetModifiedStagedFiles(matchedFiles)
			res.ModifiedFiles = modified
		}
	}

	return res
}
