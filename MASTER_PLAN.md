# hookcraft: Architecture and Engineering Master Plan

Ultra-Fast Zero-Dependency Git Hook Orchestrator and Staged Secret Leak Shield in Go (< 15MB RAM).
A modern, single-binary alternative to Husky, Python pre-commit, and Lefthook.

---

## 1. System Overview

HookCraft solves four core problems in developer workflows:
1. Runtime dependency overhead: Tools like Husky require Node.js and npm in non-JS projects (Go, Rust, Python, C++).
2. Execution latency: Python pre-commit takes 3 to 8 seconds per commit due to virtualenv setup and interpreter startup.
3. Secret leakage: Standard hook runners lack built-in real-time secret scanning on staged diffs before commit creation.
4. Formatter stash conflicts: Formatters modify code during hooks, leaving changes unstaged or clobbering dirty working trees.

HookCraft compiles to a single native Go binary without runtime dependencies. It installs POSIX shims into `.git/hooks` with `hookcraft install`, evaluates staged diffs (`git diff --staged`) concurrently in under 100 milliseconds, blocks credentials using Shannon entropy and pattern matching, and provides one-key interactive staging for auto-formatted files.

---

## 2. Architecture and Data Flow

```
[ git commit -m "feat: core" ]
             |
             v
+-------------------------------------------------------------------------+
| hookcraft pre-commit (Binary entrypoint)                                |
|                                                                         |
| 1. Environment & Git Discovery                                          |
|    - Verify git repository root via git rev-parse                       |
|    - Parse .hookcraft.yaml (or embedded language presets)               |
|                                                                         |
| 2. Stash Guard (Safe Partial Staging)                                   |
|    - Detect unstaged modifications on staged files                      |
|    - Stash unstaged changes with git stash create / store               |
|    - Guarantee cleanup via deferred signal handlers (SIGINT/SIGTERM)   |
|                                                                         |
| 3. Staged Diff Extraction                                               |
|    - git diff --staged --name-only -z (nul-byte safe paths)            |
|    - Parse git diff --staged -U0 for hunk-level inspection             |
|                                                                         |
| 4. Secret Leak Shield (Shannon Entropy + 25+ Signatures)                |
|    - Evaluate added lines only ('+' lines in diff hunks)                |
|    - High entropy verification (> 3.8 bits/byte on raw tokens)          |
|    - Provider signatures: AWS, OpenAI, GitHub PAT, Slack, Stripe, etc.  |
|    - Check inline ignore comments (hookcraft:ignore)                    |
|    - If leak detected: Print file, line, redacted token, abort (exit 1) |
|                                                                         |
| 5. Concurrent Hook Execution Engine                                     |
|    - Worker pool sized to runtime.NumCPU()                             |
|    - Filter staged files against hook glob rules                        |
|    - Dynamic variable substitution ({staged_files}, {all_files})        |
|    - Stream command output, capture elapsed runtimes                    |
|                                                                         |
| 6. Formatter Auto-Staging & Interactive Approver                        |
|    - Detect working tree file modifications post-execution              |
|    - If stage_fixed is true: auto-stage with git add                    |
|    - If interactive: display concise diff and prompt [Y/n]              |
|                                                                         |
| 7. Stash Restore & Metric Recording                                     |
|    - Restore unstaged working tree changes cleanly                      |
|    - Append run metrics to .hookcraft/history.json                      |
|    - Exit 0 to complete commit in sub-100ms                             |
+-------------------------------------------------------------------------+
```

---

## 3. Technology Stack

- Language: Go 1.23+ (Compiled native binary, memory footprint under 15MB)
- Git Interface: Direct Git plumbing commands (`git rev-parse`, `git diff`, `git update-index`, `git stash`)
- Secret Analysis: Shannon entropy calculation and pre-compiled regex signatures for 25+ cloud credentials
- Concurrency: Goroutine worker pool with context cancellation and error groups
- Terminal Interface: ANSI 256-color terminal engine, spinner animations, and raw-mode keypress reader
- Embedded Dashboard: React 19 + TypeScript + Vite + Tailwind CSS compiled into Go binary via `embed.FS`
- Storage: Flat JSON execution log (`.hookcraft/history.json`) for timeline visualization

---

## 4. Repository Structure

```
B:\workgit\hookcraft\
├── cmd\
│   └── hookcraft\
│       └── main.go                 # CLI entrypoint (run, install, uninstall, scan, init, serve)
├── internal\
│   ├── config\
│   │   ├── config.go               # Configuration parser and schema definition
│   │   ├── defaults.go             # Preset detectors for Go, Node/TS, Python, Rust
│   │   └── validator.go            # Config validation and glob syntax checker
│   ├── git\
│   │   ├── repo.go                 # Repository discovery and git plumbing executor
│   │   ├── hooks.go                # Hook shim installer and uninstaller
│   │   ├── staged.go               # Staged file lister and diff hunk parser
│   │   └── stash.go                # Safe partial staging and working tree isolation
│   ├── secrets\
│   │   ├── entropy.go              # Shannon entropy analyzer for randomized tokens
│   │   ├── rules.go                # Signatures for AWS, OpenAI, GitHub, Stripe, SSH, etc.
│   │   ├── scanner.go              # Diff scanner, inline suppression, line mapper
│   │   └── allowlist.go            # Path exclusions and token false-positive filters
│   ├── runner\
│   │   ├── engine.go               # Concurrent worker pool and execution pipeline
│   │   ├── worker.go               # Process runner with context timeout and output pipes
│   │   ├── substitution.go         # Variable replacement ({staged_files}, {count}, etc.)
│   │   └── history.go              # Execution run metrics logger
│   ├── tui\
│   │   ├── terminal.go             # ANSI colors, status indicators, and summary tables
│   │   ├── spinner.go              # Lightweight terminal spinner
│   │   └── prompt.go               # Interactive keyboard reader for fix approval
│   └── server\
│       ├── server.go               # HTTP API server and embedded asset handler
│       ├── handlers.go             # Endpoints for metrics, config, diffs, and health
│       └── embedded.go             # Go embed integration with fallback development proxy
├── web\
│   ├── src\
│   │   ├── components\
│   │   │   ├── Header.tsx          # Top navigation and status bar
│   │   │   ├── WaterfallChart.tsx  # Execution time breakdown per hook
│   │   │   ├── SecretShieldView.tsx# Secret audit log with file line highlights
│   │   │   ├── StagedDiffView.tsx  # Colorized staged diff browser
│   │   │   ├── ConfigEditor.tsx    # Live YAML config inspector and validator
│   │   │   └── MetricCards.tsx     # Summary statistics (memory, speed, commit counts)
│   │   ├── types\
│   │   │   └── index.ts            # TypeScript interfaces matching Go backend
│   │   ├── App.tsx                 # Main dashboard view
│   │   ├── main.tsx                # React 19 bootstrap
│   │   └── index.css               # Tailwind CSS rules
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── docs\
│   ├── ARCHITECTURE.md             # Detailed engineering specifications
│   ├── CONFIGURATION.md            # Configuration reference and presets
│   └── SECRET_SHIELD.md            # Entropy heuristics and signature database
├── .hookcraft.yaml                 # Repository configuration
├── .gitignore
├── Makefile                        # Multi-platform compilation and test runner
├── Dockerfile                      # Scratch container image
├── LICENSE                         # MIT License
├── README.md                       # Documentation following humanizer guidelines
└── MASTER_PLAN.md                  # This file
```

---

## 5. Implementation Roadmap and Progress Tracking

### Phase 1: Git Plumbing and Environment Detection
- [x] Create public GitHub repository with 14 topic tags and description.
- [x] Register HookCraft in developer portfolio (mtlg-site & mtlglabs-space Phase 1).
- [x] Initialize Go module `github.com/alexandrmotologa/hookcraft`.
- [x] Implement `internal/git/repo.go` for git discovery and command execution.
- [x] Implement `internal/git/staged.go` for nul-terminated staged file extraction and hunk parsing.
- [x] Implement `internal/git/stash.go` for safe stash preservation of unstaged modifications.
- [x] Implement `internal/git/hooks.go` for shim generation in `.git/hooks/`.
- [x] Write unit tests for git plumbing operations.

### Phase 2: Configuration and Preset Engine
- [x] Define YAML schema in `internal/config/config.go` supporting hooks, globs, parallel execution, and timeouts.
- [x] Implement built-in auto-detection presets for Go, TypeScript, Python, and Rust in `internal/config/defaults.go`.
- [x] Add config validation and glob matching logic.
- [x] Implement `hookcraft init` CLI command to bootstrap tailored configs.
- [x] Write unit tests for configuration parsing, fallback defaults, and validation.

### Phase 3: High-Speed Secret Leak Shield
- [x] Implement Shannon entropy calculator in `internal/secrets/entropy.go`.
- [x] Create pattern database in `internal/secrets/rules.go` covering 25+ cloud credentials.
- [x] Implement diff hunk scanner in `internal/secrets/scanner.go` checking only added lines.
- [x] Implement false positive suppression (`hookcraft:ignore` comments, test fixture paths, hashes).
- [x] Build standalone `hookcraft scan` CLI command for testing diffs or directory trees.
- [x] Write comprehensive unit tests for secret detection, entropy scoring, and suppression.

### Phase 4: Concurrent Execution Engine
- [x] Build worker pool and execution scheduler in `internal/runner/engine.go`.
- [x] Implement file glob filtering and placeholder substitution (`{staged_files}`).
- [x] Implement process execution with context timeouts, exit code handling, and output capture.
- [x] Implement post-execution file modification detector for linters and formatters.
- [x] Implement `.hookcraft/history.json` run metrics logger.
- [x] Write unit tests for parallel execution, command timeout handling, and file substitution.

### Phase 5: Terminal User Interface (TUI)
- [x] Implement ANSI color palette, spinner animations, and run summary tables in `internal/tui/terminal.go`.
- [x] Implement interactive 1-key approval prompt (`[Y]es / [n]o / [d]iff`) in `internal/tui/prompt.go`.
- [x] Support non-interactive CI mode (`--non-interactive` or detecting absence of TTY).
- [x] Verify execution time output matches sub-100ms targets.

### Phase 6: Embedded Web Dashboard (React 19 + TypeScript + Vite)
- [x] Scaffold React 19 application in `web/` using Tailwind CSS and Lucide icons.
- [x] Build hook waterfall timeline component displaying historical and current execution times.
- [x] Build secret shield audit viewer displaying flagged files, line numbers, and redacted tokens.
- [x] Build staged diff viewer for visual code inspection.
- [x] Build live configuration viewer with syntax validation.
- [x] Implement Go HTTP server in `internal/server/` serving embedded UI and JSON REST endpoints.
- [x] Implement `hookcraft serve` and `hookcraft studio` commands.

### Phase 7: Verification, Documentation, and Release
- [x] Run full test suite (`go test -v -race ./...`).
- [x] Write `docs/ARCHITECTURE.md`, `docs/CONFIGURATION.md`, and `docs/SECRET_SHIELD.md`.
- [x] Write humanized `README.md` without AI cliches, em dashes, or empty claims.
- [x] Create Makefile and cross-compilation release scripts.

### Phase 8: Advanced Security and Git Governance Shields
- [x] Implement `.hookcraftignore` parser supporting global exclusions for secret scanning and hooks.
- [x] Implement `builtin:file_size_shield` to prevent accidental commits of large files (> 5MB).
- [x] Implement `builtin:branch_protection` for `pre-push` hooks to guard against direct pushes to `main`/`master`.
- [x] Write unit tests for `.hookcraftignore`, file size inspection, and branch protection.

### Phase 9: Content-Addressable Caching Engine
- [x] Implement `internal/cache/cache.go` computing SHA-256 hashes of command strings and file contents.
- [x] Store cache index in `.hookcraft/cache/index.json`.
- [x] Skip unmodified steps with instant cache hit (`⚡ CACHED (< 1ms)`).
- [x] Add `hookcraft cache clean` CLI command.
- [x] Write unit tests for cache hits, file modification invalidations, and command changes.

### Phase 10: Automatic Migration Tool and Issue Ticket Enforcement
- [x] Implement `internal/migrate/migrate.go` parsing `.pre-commit-config.yaml`, `lefthook.yml`, and `.husky/`.
- [x] Add `hookcraft migrate` CLI command generating `.hookcraft.yaml`.
- [x] Extend `builtin:conventional_commits` to support issue ticket validation (`ticket_pattern`).
- [x] Write unit tests for migration parsers and ticket pattern matching.

### Phase 11: Interactive Studio Actions and Real-Time Runner
- [x] Implement `POST /api/run` in `internal/server/` to execute hooks on demand.
- [x] Implement `POST /api/stage` to stage modified files directly from the browser.
- [x] Add interactive "Run Hook" button in React 19 Studio Overview.
- [x] Add "Stage All Formatted Files" button in Staged Diff View.
- [x] Rebuild React 19 production distribution (`npm run build`).
