package secrets

import (
	"path/filepath"
	"strings"
)

// DefaultExcludedExtensions are file extensions that are binary, media, or safe generated files.
var DefaultExcludedExtensions = map[string]bool{
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".gif":   true,
	".svg":   true,
	".ico":   true,
	".pdf":   true,
	".woff":  true,
	".woff2": true,
	".ttf":   true,
	".eot":   true,
	".lock":  true,
}

// DefaultExcludedFilenames are standard files that commonly trigger false positives.
var DefaultExcludedFilenames = map[string]bool{
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":    true,
	"go.sum":            true,
	"Cargo.lock":        true,
	"poetry.lock":       true,
	"composer.lock":     true,
	"Gemfile.lock":      true,
}

// IsPathExcluded checks if the given file path should be exempted from secret scanning.
func IsPathExcluded(filePath string) bool {
	cleanPath := filepath.ToSlash(filePath)
	base := filepath.Base(cleanPath)

	if DefaultExcludedFilenames[base] {
		return true
	}

	ext := strings.ToLower(filepath.Ext(base))
	if DefaultExcludedExtensions[ext] {
		return true
	}

	// Skip test files, mock files, and documentation
	lower := strings.ToLower(cleanPath)
	if strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".test.ts") ||
		strings.HasSuffix(lower, ".test.js") ||
		strings.HasSuffix(lower, ".spec.ts") ||
		strings.HasSuffix(lower, ".spec.js") ||
		strings.HasSuffix(lower, ".md") ||
		strings.Contains(lower, "/testdata/") ||
		strings.Contains(lower, "/fixtures/") ||
		strings.Contains(lower, "/mocks/") ||
		strings.HasPrefix(lower, "docs/") ||
		strings.Contains(lower, "web/dist/") {
		return true
	}

	return false
}

// HasInlineSuppression returns true if line contains a suppression pragma.
func HasInlineSuppression(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(lower, "hookcraft:ignore") ||
		strings.Contains(lower, "hookcraft:allow") ||
		strings.Contains(lower, "gosec:ignore") ||
		strings.Contains(lower, "nosec") ||
		strings.Contains(lower, "pragma: allowlist secret")
}

// RedactSecret replaces the sensitive portion of a secret with asterisks, preserving prefix and length.
func RedactSecret(raw string) string {
	length := len(raw)
	if length <= 6 {
		return strings.Repeat("*", length)
	}

	// Show first 4 characters and last 2 characters
	visiblePrefix := 4
	visibleSuffix := 2
	if length < 10 {
		visiblePrefix = 2
		visibleSuffix = 1
	}

	maskLength := length - visiblePrefix - visibleSuffix
	return raw[:visiblePrefix] + strings.Repeat("*", maskLength) + raw[length-visibleSuffix:]
}
