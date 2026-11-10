package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alexandrmotologa/hookcraft/internal/runner"
	"github.com/alexandrmotologa/hookcraft/internal/secrets"
)

func TestOutputFormatting(t *testing.T) {
	var buf bytes.Buffer
	out := NewOutput(&buf)
	out.noColor = true // Disable color for deterministic text assertions

	// Test banner
	out.PrintBanner("pre-commit", 3)
	if !strings.Contains(buf.String(), "hookcraft :: pre-commit [3 staged files]") {
		t.Errorf("unexpected banner output: %s", buf.String())
	}
	buf.Reset()

	// Test passed step
	out.PrintStep(runner.StepResult{
		StepName:   "Go Format",
		Success:    true,
		DurationMs: 14.2,
		ModifiedFiles: []string{
			"main.go",
		},
	})
	if !strings.Contains(buf.String(), "Go Format") || !strings.Contains(buf.String(), "14.2ms") {
		t.Errorf("unexpected step output: %s", buf.String())
	}
	buf.Reset()

	// Test failed step
	out.PrintStep(runner.StepResult{
		StepName:   "Linter",
		Success:    false,
		DurationMs: 25.0,
		Output:     "syntax error at line 10",
	})
	if !strings.Contains(buf.String(), "FAIL") || !strings.Contains(buf.String(), "syntax error at line 10") {
		t.Errorf("unexpected failed step output: %s", buf.String())
	}
	buf.Reset()

	// Test secret alert
	findings := []secrets.Finding{
		{
			FilePath:      "credentials.env",
			LineNumber:    4,
			RuleName:      "AWS Access Key ID",
			RedactedValue: "AKIA****************",
			RawLine:       "AWS_KEY=AKIA1234567890EXAMPLE",
		},
	}
	out.PrintSecretShieldAlert(findings)
	if !strings.Contains(buf.String(), "SECURITY ALERT") || !strings.Contains(buf.String(), "credentials.env:4") {
		t.Errorf("unexpected secret alert: %s", buf.String())
	}
	buf.Reset()

	// Test summary
	out.PrintSummary(&runner.RunSummary{
		Success:    true,
		DurationMs: 45.8,
	})
	if !strings.Contains(buf.String(), "All hooks passed cleanly") || !strings.Contains(buf.String(), "45.8ms") {
		t.Errorf("unexpected summary output: %s", buf.String())
	}
}

func TestColorize(t *testing.T) {
	var buf bytes.Buffer
	out := NewOutput(&buf)

	out.noColor = true
	if out.Colorize(Bold+Green, "test") != "test" {
		t.Errorf("expected plain text when noColor is true")
	}

	out.noColor = false
	colored := out.Colorize(Bold+Green, "test")
	if !strings.HasPrefix(colored, Bold+Green) || !strings.HasSuffix(colored, Reset) {
		t.Errorf("expected ANSI wrapped text, got %q", colored)
	}
}
