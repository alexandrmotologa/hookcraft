package secrets

import (
	"math"
	"regexp"
	"strings"
)

var (
	// uuidPattern identifies standard UUIDs which have moderate entropy but are not secrets
	uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	// hexHashPattern identifies git commit SHAs, md5, sha1, sha256 hashes
	hexHashPattern = regexp.MustCompile(`^[0-9a-fA-F]{32,64}$`)
)

// CalculateShannonEntropy calculates the Shannon entropy in bits per byte for a string.
// Formula: H(X) = - sum(P(x) * log2(P(x)))
func CalculateShannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0.0
	}

	freqMap := make(map[rune]int)
	for _, ch := range s {
		freqMap[ch]++
	}

	length := float64(len([]rune(s)))
	var entropy float64

	for _, count := range freqMap {
		prob := float64(count) / length
		if prob > 0 {
			entropy -= prob * math.Log2(prob)
		}
	}

	return entropy
}

// ExtractCandidateTokens finds candidate high-entropy secret tokens from a code line.
func ExtractCandidateTokens(line string) []string {
	// Skip whole line if it's purely a URL or import path
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") || strings.HasPrefix(trimmed, "import ") {
		return nil
	}

	// Split by quotes, whitespace, equal signs, colons, semicolons, brackets, dots, operators, hyphens
	delims := func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\r' || r == '\n' ||
			r == '"' || r == '\'' || r == '`' || r == '=' ||
			r == ':' || r == ';' || r == ',' || r == '(' || r == ')' ||
			r == '{' || r == '}' || r == '[' || r == ']' || r == '<' || r == '>' ||
			r == '/' || r == '\\' || r == '?' || r == '&' || r == '#' || r == '|' ||
			r == '.' || r == '!' || r == '*' || r == '+' || r == '~' || r == '^' ||
			r == '%' || r == '$' || r == '@' || r == '-'
	}

	parts := strings.FieldsFunc(line, delims)
	var candidates []string

	for _, part := range parts {
		token := strings.TrimSpace(part)
		// Minimum length of random keys (AWS, OpenAI, tokens are typically 20+ chars)
		if len(token) < 20 || len(token) > 160 {
			continue
		}

		// Filter out standard UUIDs
		if uuidPattern.MatchString(token) {
			continue
		}

		// Skip URLs and hostnames
		if strings.HasPrefix(token, "http") || strings.Contains(token, ".com") || strings.Contains(token, ".org") || strings.Contains(token, ".io") {
			continue
		}

		// Real random secrets contain digits (hashes, base64, random API tokens)
		// Purely alphabetic strings or words without digits are identifiers
		if !hasDigit(token) {
			continue
		}

		// Filter out pure alphanumeric continuous words with repetitive charset
		if isAllSameChar(token) {
			continue
		}

		candidates = append(candidates, token)
	}

	return candidates
}

func hasDigit(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func isAllSameChar(s string) bool {
	if len(s) == 0 {
		return true
	}
	first := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] != first {
			return false
		}
	}
	return true
}

// IsEntropySecret returns true if token exceeds the entropy threshold and is not a known false positive.
func IsEntropySecret(token string, threshold float64) (bool, float64) {
	// Skip common file hashes (e.g., in lockfiles or integrity checks)
	if hexHashPattern.MatchString(token) {
		return false, 0.0
	}

	// Skip common test values
	upper := strings.ToUpper(token)
	if strings.Contains(upper, "EXAMPLE") ||
		strings.Contains(upper, "TEST") ||
		strings.Contains(upper, "PLACEHOLDER") ||
		strings.Contains(upper, "DUMMY") {
		return false, 0.0
	}

	entropy := CalculateShannonEntropy(token)
	return entropy >= threshold, entropy
}
