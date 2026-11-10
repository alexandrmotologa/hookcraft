# HookCraft Configuration Reference

HookCraft is configured via `.hookcraft.yaml` placed at the root of the repository.

---

## 1. Minimal Example

```yaml
version: 1

settings:
  concurrency: true
  auto_stage_fixes: true
  block_secret_leaks: true

hooks:
  pre-commit:
    - name: "Secret Leak Shield"
      type: "builtin:secret_shield"
      action: "block"

    - name: "Go Format"
      glob: "*.go"
      run: "gofmt -w {staged_files}"
      stage_fixed: true
```

---

## 2. Global Settings Specification

| Setting | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `concurrency` | boolean | `true` | Executes independent hook steps concurrently across CPU cores. |
| `max_workers` | integer | `NumCPU` | Maximum number of concurrent worker processes. |
| `auto_stage_fixes` | boolean | `true` | Automatically stages files modified by formatters during hook execution. |
| `interactive_approval` | boolean | `false` | Prompts with a 1-key press `[y/n/d]` before re-staging modified files. |
| `block_secret_leaks` | boolean | `true` | Blocks commits containing API keys, private keys, or high-entropy tokens. |
| `entropy_threshold` | float | `3.8` | Minimum Shannon entropy (bits per byte) required to flag a token as random secret. |
| `timeout_seconds` | integer | `30` | Default timeout applied to command execution. |
| `stash_unstaged` | boolean | `true` | Automatically stashes unstaged working tree modifications during hook runs. |
| `fail_fast` | boolean | `false` | Cancels remaining hook steps immediately upon the first failure. |

---

## 3. Hook Step Fields

Each hook entry in `hooks.<hook_name>` accepts the following fields:

- `name` (string, required): Human-readable name displayed in the terminal and dashboard.
- `type` (string, optional): One of `command` (default), `builtin:secret_shield`, or `builtin:conventional_commits`.
- `glob` (string, optional): File pattern filter. Commands run only when matching files are staged. Supports brace expansion such as `*.{ts,tsx,js,json}`.
- `exclude_glob` (string, optional): Exclusion pattern to skip matching files (for example `vendor/*` or `*.generated.go`).
- `run` (string, optional): Shell command to execute. Supports dynamic placeholders.
- `stage_fixed` (boolean, optional): Detects if this step modified any staged files and marks them for staging.
- `action` (string, optional): Action for secret scanner: `block` (exit 1) or `warn` (print alert but continue).
- `timeout_seconds` (integer, optional): Step-specific timeout overriding the global default.
- `env` (list of strings, optional): Custom environment variables in `KEY=VALUE` format.

---

## 4. Command Placeholders

The following tokens are dynamically replaced inside `run` command strings:

- `{staged_files}`: Space-separated list of staged files matching the step's `glob` filter. Files containing spaces are automatically quoted.
- `{all_files}`: Space-separated list of all files currently staged in Git.
- `{count}`: Total count of files matching the `glob` filter.
- `{branch}`: Name of the current active Git branch.

If `{staged_files}` is present in the command and zero matching files are staged, HookCraft skips the step automatically.

---

## 5. Technology Stack Recipes

### Go Recipe

```yaml
version: 1
settings:
  concurrency: true
  auto_stage_fixes: true

hooks:
  pre-commit:
    - name: "Secret Leak Shield"
      type: "builtin:secret_shield"
      action: "block"

    - name: "Go Format"
      glob: "*.go"
      run: "gofmt -w {staged_files}"
      stage_fixed: true

    - name: "Go Vet"
      glob: "*.go"
      run: "go vet ./..."

  pre-push:
    - name: "Go Tests"
      run: "go test -short ./..."
```

### TypeScript and Node.js Recipe

```yaml
version: 1
settings:
  concurrency: true
  auto_stage_fixes: true

hooks:
  pre-commit:
    - name: "Secret Leak Shield"
      type: "builtin:secret_shield"

    - name: "Prettier"
      glob: "*.{ts,tsx,js,jsx,json,css,md}"
      run: "npx prettier --write {staged_files}"
      stage_fixed: true

    - name: "ESLint"
      glob: "*.{ts,tsx,js,jsx}"
      run: "npx eslint --max-warnings 0 {staged_files}"

  pre-push:
    - name: "Type Check"
      run: "npx tsc --noEmit"
```

### Python Recipe (Ruff)

```yaml
version: 1
settings:
  concurrency: true
  auto_stage_fixes: true

hooks:
  pre-commit:
    - name: "Secret Leak Shield"
      type: "builtin:secret_shield"

    - name: "Ruff Format"
      glob: "*.py"
      run: "ruff format {staged_files}"
      stage_fixed: true

    - name: "Ruff Lint"
      glob: "*.py"
      run: "ruff check --fix {staged_files}"
      stage_fixed: true
```
