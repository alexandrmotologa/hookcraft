# HookCraft System Architecture

HookCraft is a standalone Git hook orchestrator and secret leak shield compiled as a single native Go binary. It manages pre-commit, commit-msg, and pre-push hooks without requiring Node.js, Python, or external package managers.

---

## 1. Design Priorities

1. Zero external runtime dependencies: The binary does not require node, npm, python, or virtual environments to run.
2. Low execution latency: Independent checks execute in parallel across available CPU cores. Execution overhead remains under 50 milliseconds on standard developer workstations.
3. Memory efficiency: The entire runner maintains a memory footprint under 15MB RAM.
4. Safe index isolation: Formatters and linters must never accidentally commit unstaged modifications or clobber working tree state.

---

## 2. Component Structure

The codebase is organized into modular packages under `internal/`:

### `internal/git`
Handles all interaction with Git plumbing commands.
- `repo.go`: Discovers repository root and resolves the actual `.git` directory across regular repositories, worktrees, and submodules.
- `staged.go`: Extracts staged files using `git diff --staged --name-only -z --diff-filter=ACMR` to safely handle spaces, UTF-8 paths, and renames. Parses unified diffs with context `-U0` to track exact line numbers for added lines.
- `stash.go`: Guards unstaged working tree changes using `git stash push --keep-index`. Unstaged changes are isolated before commands run and restored when the hook exits.
- `hooks.go`: Installs POSIX shims into `.git/hooks/` and maintains `.hookcraft-backup` files to restore preexisting hooks when uninstalled.

### `internal/config`
Parses and validates `.hookcraft.yaml`.
- Supports globs with brace expansion (for example `*.{ts,tsx,js}`).
- Detects the project tech stack (Go, TypeScript, Python, Rust) when no configuration file is present.
- Applies per-step timeouts, concurrency toggles, and failure policies.

### `internal/secrets`
Evaluates staged diff hunks for sensitive credentials.
- `entropy.go`: Calculates Shannon entropy in bits per byte across candidate tokens. Tokens exceeding the configured threshold (default 3.8) are flagged.
- `rules.go`: Houses pre-compiled regex signatures for 25+ cloud and SaaS providers, including AWS, GitHub, GitLab, OpenAI, Anthropic, Stripe, Slack, and SSH keys.
- `scanner.go`: Inspects only added lines (`+` lines) in staged diff hunks, checking for both signature matches and raw high-entropy tokens.
- `allowlist.go`: Suppresses false positives via file extension filters, lockfile exclusions, and inline comments (`// hookcraft:ignore`).

### `internal/runner`
Coordinates concurrent process execution.
- `engine.go`: Dispatches hook steps sequentially or concurrently across goroutine worker pools sized to `runtime.NumCPU()`.
- `worker.go`: Substitutes variables (`{staged_files}`, `{all_files}`, `{count}`, `{branch}`) and executes commands with context timeouts.
- `history.go`: Records run statistics to `.hookcraft/history.json` for performance tracking.

### `internal/tui`
Provides colorized console output and interactive terminal controls.
- Detects TTY capability and respects `NO_COLOR` and `TERM=dumb`.
- Prints execution status tables, line pointers, and security warnings.
- Prompts developers with single-keystroke approval (`[y/n/d]`) when formatters modify staged files.

### `internal/server`
Houses the lightweight embedded web server for HookCraft Studio.
- Serves the compiled React 19 single-page dashboard directly from Go `embed.FS`.
- Exposes JSON endpoints for repository status, historical run waterfalls, staged diffs, configuration details, and on-demand secret audits.

---

## 3. Hook Execution Sequence

```
1. Developer runs git commit
2. Git executes .git/hooks/pre-commit shim
3. HookCraft resolves repository root and loads .hookcraft.yaml
4. StashGuard detects unstaged working tree changes and stores them
5. GitRunner collects staged files and diff hunks (-U0)
6. SecretShield scans added lines:
   - If secrets found: abort immediately with exit code 1
7. Worker pool runs configured linters and formatters concurrently
8. If formatters modified files:
   - If auto_stage_fixes is true: stage modified files automatically
   - If interactive: display diff and prompt user
9. StashGuard restores stashed working tree changes
10. Run metrics append to .hookcraft/history.json
11. Exit with status 0 to allow commit creation
```
