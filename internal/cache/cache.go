package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Entry stores the cached result of an executed command step.
type Entry struct {
	Key       string    `json:"key"`
	StepName  string    `json:"step_name"`
	Command   string    `json:"command"`
	FilesHash string    `json:"files_hash"`
	ExitCode  int       `json:"exit_code"`
	Output    string    `json:"output"`
	Timestamp time.Time `json:"timestamp"`
}

// Manager coordinates reading and writing cache entries to disk.
type Manager struct {
	CacheDir string
	mu       sync.RWMutex
	entries  map[string]Entry
}

// NewManager creates or loads a cache manager for the repository.
func NewManager(repoRoot string) *Manager {
	dir := filepath.Join(repoRoot, ".hookcraft", "cache")
	mgr := &Manager{
		CacheDir: dir,
		entries:  make(map[string]Entry),
	}
	_ = mgr.loadIndex()
	return mgr
}

func (m *Manager) loadIndex() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	indexPath := filepath.Join(m.CacheDir, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &m.entries)
}

func (m *Manager) saveIndex() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(m.CacheDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(m.CacheDir, "index.json"), data, 0644)
}

// ComputeKey calculates a unique SHA-256 key for a step based on its command and file contents.
func ComputeKey(stepName, command string, files []string, repoRoot string) (string, error) {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s\n%s\n", stepName, command)))

	// Sort files for deterministic order
	sortedFiles := make([]string, len(files))
	copy(sortedFiles, files)
	sort.Strings(sortedFiles)

	for _, f := range sortedFiles {
		h.Write([]byte(f + "\n"))
		fullPath := filepath.Join(repoRoot, f)

		file, err := os.Open(fullPath)
		if err == nil {
			_, _ = io.Copy(h, file)
			_ = file.Close()
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// Get retrieves a cached entry if available.
func (m *Manager) Get(key string) (*Entry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.entries[key]
	if !ok {
		return nil, false
	}
	return &entry, true
}

// Put records a successful command execution into the cache.
func (m *Manager) Put(key, stepName, command string, exitCode int, output string) {
	m.mu.Lock()
	m.entries[key] = Entry{
		Key:       key,
		StepName:  stepName,
		Command:   command,
		ExitCode:  exitCode,
		Output:    output,
		Timestamp: time.Now(),
	}
	m.mu.Unlock()

	_ = m.saveIndex()
}

// Clean removes all cached execution entries.
func (m *Manager) Clean() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = make(map[string]Entry)
	return os.RemoveAll(m.CacheDir)
}

// Size returns the count of cached step entries.
func (m *Manager) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entries)
}
