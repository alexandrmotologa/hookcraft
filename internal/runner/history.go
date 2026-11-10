package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RunSummary encapsulates the complete execution run of a hook.
type RunSummary struct {
	ID          string       `json:"id"`
	HookName    string       `json:"hook_name"`
	Timestamp   time.Time    `json:"timestamp"`
	DurationMs  float64      `json:"duration_ms"`
	Success     bool         `json:"success"`
	StagedFiles []string     `json:"staged_files"`
	Steps       []StepResult `json:"steps"`
	TotalMemory float64      `json:"total_memory_mb"`
}

var historyMutex sync.Mutex

// RecordHistory appends a run summary into .hookcraft/history.json, keeping at most maxRecords.
func RecordHistory(repoRoot string, summary RunSummary) error {
	historyMutex.Lock()
	defer historyMutex.Unlock()

	dir := filepath.Join(repoRoot, ".hookcraft")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	historyFile := filepath.Join(dir, "history.json")
	var records []RunSummary

	if data, err := os.ReadFile(historyFile); err == nil {
		_ = json.Unmarshal(data, &records)
	}

	// Prepend new run summary
	records = append([]RunSummary{summary}, records...)

	// Cap at 50 historical runs
	if len(records) > 50 {
		records = records[:50]
	}

	marshaled, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(historyFile, marshaled, 0644)
}

// LoadHistory reads past run summaries from .hookcraft/history.json.
func LoadHistory(repoRoot string) ([]RunSummary, error) {
	historyMutex.Lock()
	defer historyMutex.Unlock()

	historyFile := filepath.Join(repoRoot, ".hookcraft", "history.json")
	data, err := os.ReadFile(historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []RunSummary{}, nil
		}
		return nil, err
	}

	var records []RunSummary
	if err := json.Unmarshal(data, &records); err != nil {
		return []RunSummary{}, nil
	}
	return records, nil
}
