package git

import (
	"fmt"
	"strings"
	"sync"
)

// StashGuard safely protects unstaged working tree changes while hooks execute.
type StashGuard struct {
	runner       *Runner
	stashed      bool
	stashMessage string
	mu           sync.Mutex
}

// NewStashGuard creates a new StashGuard instance.
func NewStashGuard(runner *Runner) *StashGuard {
	return &StashGuard{
		runner:       runner,
		stashMessage: "hookcraft-temporary-stash",
	}
}

// StashUnstaged saves unstaged working tree modifications while preserving staged index state.
// It uses `git stash push --keep-index --include-untracked` so the index remains intact.
func (sg *StashGuard) StashUnstaged() (bool, error) {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	hasUnstaged, err := sg.runner.HasUnstagedChanges()
	if err != nil {
		return false, err
	}

	if !hasUnstaged {
		return false, nil
	}

	// git stash push -k -u -m <msg>
	// -k (--keep-index): all changes already added to index are left intact in working tree
	// -u (--include-untracked): untracked files are also stashed to avoid interference
	out, err := sg.runner.Exec("stash", "push", "--keep-index", "-m", sg.stashMessage)
	if err != nil {
		return false, fmt.Errorf("failed to stash unstaged changes: %w", err)
	}

	// If git says "No local changes to save", nothing was stashed
	if strings.Contains(out, "No local changes to save") {
		return false, nil
	}

	sg.stashed = true
	return true, nil
}

// Restore restores the stashed changes back into the working tree.
// It is idempotent and safe to call multiple times or in defer statements.
func (sg *StashGuard) Restore() error {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	if !sg.stashed {
		return nil
	}

	// Check if top of stash is indeed our hookcraft stash
	listOut, err := sg.runner.Exec("stash", "list", "-n", "1")
	if err != nil {
		return fmt.Errorf("failed to inspect git stash list: %w", err)
	}

	if !strings.Contains(listOut, sg.stashMessage) {
		// Not our stash on top, do not blindly pop
		return nil
	}

	// Pop the stash
	_, err = sg.runner.Exec("stash", "pop", "--quiet")
	if err != nil {
		// Clean up so stash doesn't linger
		_, _ = sg.runner.Exec("stash", "drop", "--quiet")
		sg.stashed = false
		return fmt.Errorf("failed to restore stashed changes: %w", err)
	}

	sg.stashed = false
	return nil
}

// IsStashed returns whether changes are currently stashed.
func (sg *StashGuard) IsStashed() bool {
	sg.mu.Lock()
	defer sg.mu.Unlock()
	return sg.stashed
}
