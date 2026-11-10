export interface StatusResponse {
  repo_root: string;
  branch: string;
  hook_status: Record<string, boolean>;
  staged_files: string[];
  staged_count: number;
  has_unstaged: boolean;
  config_path: string;
  secret_rule_count: number;
  version: string;
}

export interface SecretFinding {
  file_path: string;
  line_number: number;
  rule_id: string;
  rule_name: string;
  redacted_value: string;
  raw_line: string;
  entropy?: number;
  description: string;
}

export interface StepResult {
  step_name: string;
  type: string;
  command: string;
  duration_ms: number;
  exit_code: number;
  success: boolean;
  output: string;
  matched_files: string[];
  modified_files?: string[];
  skipped: boolean;
  skip_reason?: string;
  secret_findings?: SecretFinding[];
}

export interface RunSummary {
  id: string;
  hook_name: string;
  timestamp: string;
  duration_ms: number;
  success: boolean;
  staged_files: string[];
  steps: StepResult[];
  total_memory_mb: number;
}

export interface DiffLine {
  LineNumber: number;
  Content: string;
}

export interface FileHunk {
  FilePath: string;
  OldPath?: string;
  IsNew: boolean;
  IsDeleted: boolean;
  AddedLines: DiffLine[];
}

export interface DiffResponse {
  raw_diff: string;
  hunks: FileHunk[];
}

export interface ConfigResponse {
  config: {
    version: number;
    settings: {
      concurrency: boolean;
      max_workers: number;
      auto_stage_fixes: boolean;
      interactive_approval: boolean;
      block_secret_leaks: boolean;
      entropy_threshold: number;
      timeout_seconds: number;
      stash_unstaged: boolean;
      fail_fast: boolean;
    };
    hooks: Record<string, Array<{
      name: string;
      type: string;
      glob?: string;
      run?: string;
      stage_fixed?: boolean;
      action?: string;
    }>>;
  };
  raw_yaml: string;
  path: string;
}

export interface ScanResponse {
  findings: SecretFinding[];
  count: number;
  threshold: number;
  rules_count: number;
}
