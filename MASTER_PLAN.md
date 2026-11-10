# `hookcraft` — Architecture & Engineering Master Plan

> **Ultra-Fast Zero-Dependency Git Hook Orchestrator & Staged Secret Leak Shield (< 15MB RAM)**  
> *Husky, Python Pre-Commit, and Lefthook Alternative in a Single Binary (Go + React 19 / TUI)*

---

## 1. Executive Summary & Problem Space

### The Core Problem
Enforcing code quality, formatting, and security before code is committed to Git is an essential engineering discipline. However, current git hook tools create constant developer frustration:
1. **Runtime Dependency Overhead (Husky)**: Requires Node.js and npm to be installed on every machine, making it completely unsuitable for pure Go, Rust, Python, or C++ repositories.
2. **Glacial Execution Speed (Python `pre-commit`)**: Takes 3–8 seconds per commit because it creates and manages isolated Python virtual environments for every single hook, disrupting developer flow.
3. **Accidental Secret Leaks**: Developers frequently commit `.env` files, AWS credentials, or OpenAI API keys by mistake because standard hook runners do not provide built-in high-speed entropy scanners.
4. **Formatting Stash Conflicts**: When a formatter (like Prettier, Ruff, or Black) fixes code, developers must manually unstage, re-stage, and re-run the commit process.

### The Solution: `hookcraft`
`hookcraft` is an ultra-fast, zero-dependency git hook orchestrator and secret leak shield compiled as a single Go binary. It installs into `.git/hooks` with a single command (`hookcraft install`).

`hookcraft` runs configured linters, formatters, and type-checkers **concurrently across CPU cores in sub-100 milliseconds**, operating exclusively on staged files (`git diff --staged`). It includes an **Automated Secret Leak Shield** that blocks commits containing API keys or credentials, and an **Interactive Fix Approver** that automatically re-stages clean formatted code with zero manual friction.

---

## 2. System Architecture & Hook Lifecycle

```
[ Developer runs `git commit -m "feat: user auth"` ]
                         │
                         ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        `hookcraft` Pre-Commit Hook                     │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Staged File Scanner (git diff --staged --name-only)              │  │
│  │ - Identifies modified files by extension (.ts, .go, .py, .rs)    │  │
│  │ - Stashes unstaged modifications to prevent dirty file leakage   │  │
│  └──────────────────────────────────┬───────────────────────────────┘  │
│                                     │                                  │
│                                     ▼                                  │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ High-Speed Secret Leak Shield (Shannon Entropy + Regex Scanner)  │  │
│  │ - Scans staged diffs for AWS keys, OpenAI tokens, private keys   │  │
│  │ - If SECRET FOUND -> Abort commit immediately with line pointer  │  │
│  └──────────────────────────────────┬───────────────────────────────┘  │
│                                     │ Clean Diffs                      │
│                                     ▼                                  │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Concurrent Hook Worker Pool (Goroutines across CPU Cores)        │  │
│  │                                                                  │  │
│  │  [ Worker 1: Biome / Prettier ] --> Format check / auto-fix      │  │
│  │  [ Worker 2: ESLint / Ruff ]    --> Code linting                 │  │
│  │  [ Worker 3: tsc / go vet ]     --> Type and syntax validation   │  │
│  └──────────────────────────────────┬───────────────────────────────┘  │
│                                     │                                  │
│         ┌───────────────────────────┴──────────────────────────┐       │
│         │ Any files auto-formatted?                            │ All passed?
│         ▼                                                      ▼       │
│  ┌───────────────────────────────┐           ┌──────────────────────┐  │
│  │ Interactive Fix Approver      │           │ Commit Proceeds!     │  │
│  │ - Shows clean TUI diff        │           │ (Exit 0, < 100ms)    │  │
│  │ - Auto-stages formatted files │           └──────────────────────┘  │
│  └───────────────────────────────┘                                     │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Embedded React 19 & TUI Studio                                   │  │
│  │ - Historical Hook Benchmark Waterfall, Secret Rules Configurator │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Technology Stack

| Layer | Component | Specification |
| :--- | :--- | :--- |
| **Language & Runtime** | Go 1.23+ | Ultra-fast native execution (< 15MB RAM), zero runtime dependencies |
| **Git Integration** | Direct Git Plumbing | Interacts via `git rev-parse`, `git diff --staged`, and `git update-index` |
| **Secret Scanner Engine**| Shannon Entropy + Regex | Scans diff chunks for high-entropy strings and known cloud provider patterns |
| **Concurrency Engine** | Goroutine Worker Pool | Parallelizes independent commands across available CPU cores |
| **Terminal TUI** | ANSI Terminal Engine | Clean terminal spinners, colored diffs, and interactive keystroke prompts |
| **Embedded Web UI** | React 19 + TypeScript + Vite | Built as a single bundle and served via Go `//go:embed` |
| **UI Styling** | Tailwind CSS + Lucide Icons | High-contrast modern developer tool cockpit |

---

## 4. Repository & Directory Structure

```
B:\workgit\hookcraft\
├── cmd\
│   └── hookcraft\
│       └── main.go                 # CLI entrypoint (`run`, `install`, `lint`, `serve`)
├── internal\
│   ├── config\
│   │   ├── config.go               # YAML configuration parser (`.hookcraft.yaml`)
│   │   └── defaults.go             # Built-in presets for TypeScript, Go, Python, and Rust
│   ├── git\
│   │   ├── hooks.go                # Installs/uninstalls hook scripts in `.git/hooks/`
│   │   ├── staged.go               # Inspects staged files and diff hunks
│   │   └── stash.go                # Stashes and unstashes unstaged changes safely
│   ├── runner\
│   │   ├── engine.go               # Concurrent hook execution orchestrator
│   │   ├── worker.go               # Individual command runner with timeout and exit code check
│   │   └── formatter.go            # Auto-formatting detector and staging helper
│   ├── secrets\
│   │   ├── scanner.go              # Scans diff hunks for sensitive patterns
│   │   ├── entropy.go              # Shannon entropy calculator for random token detection
│   │   └── rules.go                # Regex rules: AWS, Stripe, GitHub, OpenAI, Private Keys
│   ├── tui\
│   │   ├── terminal.go             # Terminal output formatter, spinners, and progress bar
│   │   └── prompt.go               # Interactive prompt for approving auto-formatted diffs
│   └── server\
│       └── server.go               # Embedded web server for visual hook management
├── web\
│   ├── src\
│   │   ├── components\
│   │   │   ├── HookTimeline.tsx    # Benchmark waterfall of execution times per hook
│   │   │   ├── SecretShieldLog.tsx # Log of blocked secret leaks with file and line numbers
│   │   │   └── StagedDiffViewer.tsx# Visual diff viewer of staged modifications
│   │   ├── App.tsx                 # Root developer cockpit
│   │   ├── main.tsx                # React 19 entrypoint
│   │   └── index.css               # Tailwind CSS declarations
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── Makefile                        # Multi-platform build and release targets
├── Dockerfile                      # Scratch container image (< 15MB)
├── README.md                       # Comprehensive guide and `.hookcraft.yaml` configuration examples
└── MASTER_PLAN.md                  # This master specification
```

---

## 5. Configuration Specification (`.hookcraft.yaml`)

```yaml
version: 1
settings:
  concurrency: true
  auto_stage_fixes: true
  block_secret_leaks: true

hooks:
  pre-commit:
    - name: "Format (Prettier)"
      glob: "*.{ts,tsx,js,json,css}"
      run: "npx prettier --write {staged_files}"
      stage_fixed: true

    - name: "Lint (ESLint)"
      glob: "*.{ts,tsx}"
      run: "npx eslint --max-warnings 0 {staged_files}"

    - name: "Go Vet & Format"
      glob: "*.go"
      run: "gofmt -w {staged_files}"
      stage_fixed: true

    - name: "Secret Leak Shield"
      type: "builtin:secret_scanner"
      action: "block"

  pre-push:
    - name: "Type Check"
      run: "npx tsc --noEmit"
    - name: "Go Tests"
      run: "go test -short ./..."
```

---

## 6. Implementation Roadmap

### Phase 1: Git Plumbing & Staged File Scanner
- [ ] Initialize Go module `github.com/alexandrmotologa/hookcraft`.
- [ ] Implement `.git/hooks` installer creating lightweight shim scripts.
- [ ] Implement `git diff --staged` parser extracting modified filenames and diff hunks.
- [ ] Implement safe stashing of unstaged changes.

### Phase 2: Concurrent Execution Engine
- [ ] Implement parallel worker pool dispatching commands across staged files.
- [ ] Replace `{staged_files}` placeholder dynamically in configured commands.
- [ ] Capture exit codes and streaming output.
- [ ] Benchmark execution speed (verify sub-100ms run times).

### Phase 3: Built-in Secret Leak Shield
- [ ] Implement Shannon entropy calculator for detecting randomized secrets.
- [ ] Implement regex patterns for AWS (`AKIA...`), GitHub PAT (`ghp_...`), OpenAI (`sk-...`), and RSA private keys.
- [ ] Abort commit with colorized terminal line highlights when secrets are found.

### Phase 4: Interactive Fix Approver & TUI
- [ ] Implement auto-formatting detector (`git status --porcelain` on staged files).
- [ ] Implement interactive terminal prompt to auto-stage fixes with 1 keystroke (`[y/N]`).
- [ ] Build terminal spinners and ANSI execution summary table.

### Phase 5: Embedded React 19 Studio & Documentation
- [ ] Scaffold Vite + React 19 dashboard in `web/`.
- [ ] Build hook execution time benchmark charts.
- [ ] Embed static assets directly into the Go executable via `//go:embed`.
- [ ] Release single-binary executables for Linux, macOS, and Windows.
