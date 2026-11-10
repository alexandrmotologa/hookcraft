package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseDiffHunks(t *testing.T) {
	diffOutput := `diff --git a/main.go b/main.go
index 83a0e12..b45c110 100644
--- a/main.go
+++ b/main.go
@@ -10,0 +11,2 @@
+const APIKey = "AKIA1234567890EXAMPLE"
+println("hello world")
diff --git a/config.json b/config.json
new file mode 100644
index 0000000..70a4ef5
--- /dev/null
+++ b/config.json
@@ -0,0 +1,3 @@
+{
+  "debug": true
+}
`

	hunks, err := ParseDiffHunks(diffOutput)
	if err != nil {
		t.Fatalf("ParseDiffHunks returned error: %v", err)
	}

	if len(hunks) != 2 {
		t.Fatalf("expected 2 file hunks, got %d", len(hunks))
	}

	// File 1
	if hunks[0].FilePath != "main.go" {
		t.Errorf("expected FilePath main.go, got %s", hunks[0].FilePath)
	}
	if len(hunks[0].AddedLines) != 2 {
		t.Fatalf("expected 2 added lines in main.go, got %d", len(hunks[0].AddedLines))
	}
	if hunks[0].AddedLines[0].LineNumber != 11 {
		t.Errorf("expected line number 11, got %d", hunks[0].AddedLines[0].LineNumber)
	}
	if hunks[0].AddedLines[0].Content != `const APIKey = "AKIA1234567890EXAMPLE"` {
		t.Errorf("unexpected content: %s", hunks[0].AddedLines[0].Content)
	}

	// File 2
	if hunks[1].FilePath != "config.json" {
		t.Errorf("expected FilePath config.json, got %s", hunks[1].FilePath)
	}
	if !hunks[1].IsNew {
		t.Errorf("expected config.json to be flagged as new")
	}
	if len(hunks[1].AddedLines) != 3 {
		t.Fatalf("expected 3 added lines in config.json, got %d", len(hunks[1].AddedLines))
	}
}

func TestHookInstallationAndRemoval(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hookcraft-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize temp git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git init: %v", err)
	}

	runner, err := NewRunner(tempDir)
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	// Install pre-commit and pre-push
	installed, err := runner.InstallHooks([]string{"pre-commit", "pre-push"})
	if err != nil {
		t.Fatalf("InstallHooks failed: %v", err)
	}
	if len(installed) != 2 {
		t.Errorf("expected 2 installed hooks, got %d", len(installed))
	}

	status, err := runner.GetHookStatus()
	if err != nil {
		t.Fatalf("GetHookStatus failed: %v", err)
	}
	if !status["pre-commit"] {
		t.Errorf("expected pre-commit to be installed")
	}
	if !status["pre-push"] {
		t.Errorf("expected pre-push to be installed")
	}
	if status["commit-msg"] {
		t.Errorf("expected commit-msg to NOT be installed")
	}

	// Uninstall hooks
	removed, err := runner.UninstallHooks([]string{"pre-commit"})
	if err != nil {
		t.Fatalf("UninstallHooks failed: %v", err)
	}
	if len(removed) != 1 || removed[0] != "pre-commit" {
		t.Errorf("expected pre-commit removed, got %v", removed)
	}

	statusAfter, _ := runner.GetHookStatus()
	if statusAfter["pre-commit"] {
		t.Errorf("expected pre-commit to be uninstalled")
	}
	if !statusAfter["pre-push"] {
		t.Errorf("expected pre-push to remain installed")
	}
}

func TestFindRepoRoot(t *testing.T) {
	root, err := FindRepoRoot(".")
	if err != nil {
		t.Fatalf("FindRepoRoot failed: %v", err)
	}
	if !filepath.IsAbs(root) {
		t.Errorf("expected absolute path, got %s", root)
	}
}
