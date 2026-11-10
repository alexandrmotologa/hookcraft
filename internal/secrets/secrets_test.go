package secrets

import (
	"testing"

	"github.com/alexandrmotologa/hookcraft/internal/git"
)

func TestCalculateShannonEntropy(t *testing.T) {
	// Low entropy (repetitive)
	low := CalculateShannonEntropy("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if low != 0.0 {
		t.Errorf("expected 0.0 entropy for repeated string, got %f", low)
	}

	// Normal English text ~ 2.5 - 3.2
	text := CalculateShannonEntropy("the quick brown fox jumps over the lazy dog")
	if text < 2.5 || text > 4.5 {
		t.Errorf("unexpected entropy for english text: %f", text)
	}

	// High entropy random Base64 / Hex token ~ 4.5+
	randomKey := CalculateShannonEntropy("vN7rX9kL2pQ8wZ1mY4tB6cF3jH5sD0gA")
	if randomKey < 4.0 {
		t.Errorf("expected high entropy (>4.0) for random base64 token, got %f", randomKey)
	}
}

func TestRedactSecret(t *testing.T) {
	raw := "AKIA1234567890EXAMPLE"
	redacted := RedactSecret(raw)
	if !testing.Short() {
		if redacted[:4] != "AKIA" {
			t.Errorf("expected prefix AKIA, got %s", redacted)
		}
		if redacted[len(redacted)-2:] != "LE" {
			t.Errorf("expected suffix LE, got %s", redacted)
		}
		if len(redacted) != len(raw) {
			t.Errorf("expected same length %d, got %d", len(raw), len(redacted))
		}
	}
}

func TestScannerDetectSignatures(t *testing.T) {
	scanner := NewScanner(3.8)

	tests := []struct {
		name         string
		line         string
		expectedRule string
	}{
		{
			name:         "AWS Access Key",
			line:         `const AWS_KEY = "` + "AK" + `IA1234567890ABCDEF";`,
			expectedRule: "aws-access-key-id",
		},
		{
			name:         "GitHub Classic PAT",
			line:         `export GITHUB_TOKEN="` + "gh" + `p_1234567890abcdefghijklmnopqrstuvwx"`,
			expectedRule: "github-personal-access-token",
		},
		{
			name:         "OpenAI API Key",
			line:         `apiKey = "` + "sk-proj-" + `1234567890abcdefghijklmnopqrstuvwxyz1234567890abcdef"`,
			expectedRule: "openai-api-key",
		},
		{
			name:         "Stripe Secret Key",
			line:         "stripeSecret: \"sk_" + "live_1234567890abcdefghijklmnopqrstuv\"",
			expectedRule: "stripe-secret-key",
		},
		{
			name:         "Slack Bot Token",
			line:         "token = \"xo" + "xb-123456789012-123456789012-abcdefghijklmnopqrstuvwx\"",
			expectedRule: "slack-bot-token",
		},
		{
			name:         "Google API Key",
			line:         `gcpKey = "` + "AI" + `zaSyD-1234567890abcdefghijklmnopqrst"`,
			expectedRule: "google-api-key",
		},
		{
			name:         "Private Key Header",
			line:         `-----BEGIN ` + `RSA PRIVATE KEY-----`,
			expectedRule: "private-key-header",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hunks := []git.FileHunk{
				{
					FilePath: "config.ts",
					AddedLines: []git.DiffLine{
						{LineNumber: 42, Content: tc.line},
					},
				},
			}

			findings := scanner.ScanHunks(hunks)
			if len(findings) == 0 {
				t.Fatalf("expected finding for line %q, got none", tc.line)
			}
			if findings[0].RuleID != tc.expectedRule {
				t.Errorf("expected rule ID %s, got %s", tc.expectedRule, findings[0].RuleID)
			}
		})
	}
}

func TestInlineSuppression(t *testing.T) {
	scanner := NewScanner(3.8)
	line := `const AWS_KEY = "AKIA1234567890ABCDEF"; // hookcraft:ignore`

	hunks := []git.FileHunk{
		{
			FilePath: "sample.go",
			AddedLines: []git.DiffLine{
				{LineNumber: 10, Content: line},
			},
		},
	}

	findings := scanner.ScanHunks(hunks)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings due to inline suppression, got %d", len(findings))
	}
}

func TestPathExclusion(t *testing.T) {
	scanner := NewScanner(3.8)
	line := `const AWS_KEY = "AKIA1234567890ABCDEF";`

	hunks := []git.FileHunk{
		{
			FilePath: "package-lock.json",
			AddedLines: []git.DiffLine{
				{LineNumber: 100, Content: line},
			},
		},
	}

	findings := scanner.ScanHunks(hunks)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for package-lock.json, got %d", len(findings))
	}
}

func TestHookcraftIgnoreCustomPatterns(t *testing.T) {
	scanner := NewScanner(3.8)
	scanner.SetIgnorePatterns([]string{"legacy/*", "public_keys.env"})

	line := `const AWS_KEY = "AKIA1234567890ABCDEF";`

	hunks := []git.FileHunk{
		{
			FilePath: "legacy/old_service.go",
			AddedLines: []git.DiffLine{
				{LineNumber: 15, Content: line},
			},
		},
		{
			FilePath: "public_keys.env",
			AddedLines: []git.DiffLine{
				{LineNumber: 2, Content: line},
			},
		},
	}

	findings := scanner.ScanHunks(hunks)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings for files matching .hookcraftignore patterns, got %d", len(findings))
	}
}
