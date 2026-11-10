package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeKey(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hookcraft-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fileA := filepath.Join(tempDir, "fileA.go")
	_ = os.WriteFile(fileA, []byte("package main\n"), 0644)

	key1, err := ComputeKey("Go Format", "gofmt -w fileA.go", []string{"fileA.go"}, tempDir)
	if err != nil {
		t.Fatalf("ComputeKey failed: %v", err)
	}

	// Same command and contents should produce identical key
	key2, err := ComputeKey("Go Format", "gofmt -w fileA.go", []string{"fileA.go"}, tempDir)
	if err != nil {
		t.Fatalf("ComputeKey failed: %v", err)
	}
	if key1 != key2 {
		t.Errorf("expected deterministic keys, got %s != %s", key1, key2)
	}

	// Changing command should change key
	keyCmd, _ := ComputeKey("Go Vet", "go vet fileA.go", []string{"fileA.go"}, tempDir)
	if keyCmd == key1 {
		t.Errorf("expected different key for different command")
	}

	// Modifying file contents should invalidate key
	_ = os.WriteFile(fileA, []byte("package main\nfunc hello() {}\n"), 0644)
	keyModified, _ := ComputeKey("Go Format", "gofmt -w fileA.go", []string{"fileA.go"}, tempDir)
	if keyModified == key1 {
		t.Errorf("expected different key after file content change")
	}
}

func TestManagerPutGetClean(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hookcraft-cache-mgr-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewManager(tempDir)
	if mgr.Size() != 0 {
		t.Errorf("expected empty cache, got size %d", mgr.Size())
	}

	mgr.Put("key123", "Lint", "eslint file.ts", 0, "All clean")

	entry, ok := mgr.Get("key123")
	if !ok {
		t.Fatalf("expected cache hit for key123")
	}
	if entry.Output != "All clean" || entry.ExitCode != 0 {
		t.Errorf("unexpected entry data: %+v", entry)
	}

	// Test persistence by reloading
	mgr2 := NewManager(tempDir)
	entry2, ok2 := mgr2.Get("key123")
	if !ok2 || entry2.Output != "All clean" {
		t.Errorf("expected persistent cache hit across instances")
	}

	// Test Clean
	if err := mgr.Clean(); err != nil {
		t.Fatalf("Clean failed: %v", err)
	}
	if mgr.Size() != 0 {
		t.Errorf("expected 0 entries after clean, got %d", mgr.Size())
	}
}
