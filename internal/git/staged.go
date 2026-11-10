package git

import (
	"bytes"
	"strconv"
	"strings"
)

// DiffLine represents an added or modified line in a diff hunk.
type DiffLine struct {
	LineNumber int
	Content    string
}

// FileHunk represents parsed hunk information for a single file.
type FileHunk struct {
	FilePath   string
	OldPath    string
	IsNew      bool
	IsDeleted  bool
	AddedLines []DiffLine
}

// GetStagedFiles returns all file paths staged for commit (added, copied, modified, renamed).
// Deleted files are filtered out because linters and formatters cannot operate on them.
func (r *Runner) GetStagedFiles() ([]string, error) {
	// -z separates paths by NUL bytes, avoiding issues with spaces or special characters.
	// --diff-filter=ACMR includes Added, Copied, Modified, Renamed.
	out, err := r.ExecBytes("diff", "--staged", "--name-only", "-z", "--diff-filter=ACMR")
	if err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return []string{}, nil
	}

	rawParts := bytes.Split(out, []byte{0})
	files := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		p := string(part)
		if strings.TrimSpace(p) != "" {
			files = append(files, p)
		}
	}
	return files, nil
}

// GetAllStagedFiles returns all staged files including deletions.
func (r *Runner) GetAllStagedFiles() ([]string, error) {
	out, err := r.ExecBytes("diff", "--staged", "--name-only", "-z")
	if err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return []string{}, nil
	}

	rawParts := bytes.Split(out, []byte{0})
	files := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		p := string(part)
		if strings.TrimSpace(p) != "" {
			files = append(files, p)
		}
	}
	return files, nil
}

// GetStagedDiff returns the unified staged diff for specified files or all staged changes.
func (r *Runner) GetStagedDiff(files ...string) (string, error) {
	args := []string{"diff", "--staged"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}
	return r.Exec(args...)
}

// GetParsedStagedHunks parses git diff --staged -U0 to extract added lines with line numbers.
func (r *Runner) GetParsedStagedHunks(files ...string) ([]FileHunk, error) {
	args := []string{"diff", "--staged", "-U0"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}

	out, err := r.Exec(args...)
	if err != nil {
		return nil, err
	}

	return ParseDiffHunks(out)
}

// ParseDiffHunks parses unified diff output (-U0) into structured FileHunk slices.
func ParseDiffHunks(diffOutput string) ([]FileHunk, error) {
	var hunks []FileHunk
	lines := strings.Split(diffOutput, "\n")

	var currentFile *FileHunk
	currentLineNumber := 0

	for _, line := range lines {
		// New file diff header: diff --git a/path b/path
		if strings.HasPrefix(line, "diff --git ") {
			if currentFile != nil {
				hunks = append(hunks, *currentFile)
			}
			parts := strings.Fields(line)
			filePath := ""
			if len(parts) >= 4 {
				filePath = strings.TrimPrefix(parts[3], "b/")
			}
			currentFile = &FileHunk{
				FilePath:   filePath,
				AddedLines: make([]DiffLine, 0),
			}
			continue
		}

		if currentFile == nil {
			continue
		}

		if strings.HasPrefix(line, "new file mode ") {
			currentFile.IsNew = true
			continue
		}
		if strings.HasPrefix(line, "deleted file mode ") {
			currentFile.IsDeleted = true
			continue
		}

		// Hunk header: @@ -oldStart,oldCount +newStart,newCount @@
		if strings.HasPrefix(line, "@@ ") {
			hunkHeader := line[3:]
			endIdx := strings.Index(hunkHeader, " @@")
			if endIdx != -1 {
				headerContent := hunkHeader[:endIdx]
				parts := strings.Split(headerContent, " ")
				for _, part := range parts {
					if strings.HasPrefix(part, "+") {
						newRange := strings.TrimPrefix(part, "+")
						rangeParts := strings.Split(newRange, ",")
						start, err := strconv.Atoi(rangeParts[0])
						if err == nil {
							currentLineNumber = start
						}
					}
				}
			}
			continue
		}

		// Added line in diff
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++ ") {
			content := strings.TrimPrefix(line, "+")
			currentFile.AddedLines = append(currentFile.AddedLines, DiffLine{
				LineNumber: currentLineNumber,
				Content:    content,
			})
			currentLineNumber++
		}
	}

	if currentFile != nil {
		hunks = append(hunks, *currentFile)
	}

	return hunks, nil
}

// StageFiles stages the given files with git add.
func (r *Runner) StageFiles(files ...string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, files...)
	_, err := r.Exec(args...)
	return err
}

// GetModifiedStagedFiles checks if any of the given staged files were modified in the working tree.
func (r *Runner) GetModifiedStagedFiles(stagedFiles []string) ([]string, error) {
	if len(stagedFiles) == 0 {
		return []string{}, nil
	}

	// git status --porcelain=v1 gives status codes:
	// ' M' means modified in working tree but not staged.
	// 'MM' means modified in index AND modified in working tree.
	out, err := r.Exec("status", "--porcelain=v1")
	if err != nil {
		return nil, err
	}

	stagedSet := make(map[string]bool, len(stagedFiles))
	for _, f := range stagedFiles {
		stagedSet[f] = true
	}

	var modified []string
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		worktreeStatus := line[1]
		filePath := strings.TrimSpace(line[3:])

		// If working tree has modifications ('M') and it was one of our staged files
		if worktreeStatus == 'M' && stagedSet[filePath] {
			modified = append(modified, filePath)
		}
	}

	return modified, nil
}

// HasUnstagedChanges returns true if there are unstaged modifications in the working tree.
func (r *Runner) HasUnstagedChanges() (bool, error) {
	out, err := r.Exec("status", "--porcelain=v1")
	if err != nil {
		return false, err
	}

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if len(line) >= 2 {
			worktreeStatus := line[1]
			if worktreeStatus == 'M' || worktreeStatus == 'D' || line[0] == '?' {
				return true, nil
			}
		}
	}
	return false, nil
}
