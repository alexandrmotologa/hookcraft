package secrets

import (
	"strings"

	"github.com/alexandrmotologa/hookcraft/internal/git"
)

// Finding represents a flagged secret leak with line reference.
type Finding struct {
	FilePath      string  `json:"file_path"`
	LineNumber    int     `json:"line_number"`
	RuleID        string  `json:"rule_id"`
	RuleName      string  `json:"rule_name"`
	RedactedValue string  `json:"redacted_value"`
	RawLine       string  `json:"raw_line"`
	Entropy       float64 `json:"entropy"`
	Description   string  `json:"description"`
}

// Scanner scans diff hunks and raw files for secrets.
type Scanner struct {
	EntropyThreshold float64
	Rules            []Rule
	IgnorePatterns   []string
}

// NewScanner creates a new Scanner with given entropy threshold.
func NewScanner(threshold float64) *Scanner {
	if threshold <= 0 {
		threshold = 3.8
	}
	return &Scanner{
		EntropyThreshold: threshold,
		Rules:            SecretRules,
		IgnorePatterns:   nil,
	}
}

// SetIgnorePatterns configures custom ignore rules from .hookcraftignore.
func (s *Scanner) SetIgnorePatterns(patterns []string) {
	s.IgnorePatterns = patterns
}

// ScanHunks analyzes staged diff hunks, checking only added lines for secrets.
func (s *Scanner) ScanHunks(hunks []git.FileHunk) []Finding {
	var findings []Finding

	for _, hunk := range hunks {
		if IsPathExcludedWithIgnore(hunk.FilePath, s.IgnorePatterns) {
			continue
		}

		for _, line := range hunk.AddedLines {
			if HasInlineSuppression(line.Content) {
				continue
			}

			// Check against predefined rules
			lineFindings := s.scanLine(hunk.FilePath, line.LineNumber, line.Content)
			findings = append(findings, lineFindings...)
		}
	}

	return findings
}

// ScanText analyzes raw text content line by line (used for whole file scanning).
func (s *Scanner) ScanText(content string, filePath string) []Finding {
	if IsPathExcludedWithIgnore(filePath, s.IgnorePatterns) {
		return nil
	}

	var findings []Finding
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		lineNum := i + 1
		if HasInlineSuppression(line) {
			continue
		}

		lineFindings := s.scanLine(filePath, lineNum, line)
		findings = append(findings, lineFindings...)
	}

	return findings
}

// scanLine inspects a single line against regex rules and entropy heuristics.
func (s *Scanner) scanLine(filePath string, lineNum int, lineContent string) []Finding {
	var findings []Finding
	foundRuleIDs := make(map[string]bool)

	// 1. Signature Rule Check
	for _, rule := range s.Rules {
		// Fast keyword check
		if len(rule.Keywords) > 0 {
			matched := false
			for _, kw := range rule.Keywords {
				if strings.Contains(lineContent, kw) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		matches := rule.Pattern.FindAllString(lineContent, -1)
		for _, m := range matches {
			// If rule has entropy requirement, verify entropy
			if rule.MinEntropy > 0 {
				ent := CalculateShannonEntropy(m)
				if ent < rule.MinEntropy {
					continue
				}
			}

			foundRuleIDs[rule.ID] = true
			findings = append(findings, Finding{
				FilePath:      filePath,
				LineNumber:    lineNum,
				RuleID:        rule.ID,
				RuleName:      rule.Name,
				RedactedValue: RedactSecret(m),
				RawLine:       strings.TrimSpace(lineContent),
				Description:   rule.Description,
			})
		}
	}

	// 2. High Shannon Entropy Token Check (if no specific signature matched on this line)
	if len(findings) == 0 {
		candidates := ExtractCandidateTokens(lineContent)
		for _, candidate := range candidates {
			isSecret, entropy := IsEntropySecret(candidate, s.EntropyThreshold)
			if isSecret {
				findings = append(findings, Finding{
					FilePath:      filePath,
					LineNumber:    lineNum,
					RuleID:        "shannon-high-entropy",
					RuleName:      "High Shannon Entropy Secret",
					RedactedValue: RedactSecret(candidate),
					RawLine:       strings.TrimSpace(lineContent),
					Entropy:       entropy,
					Description:   "Random token exceeding Shannon entropy safety threshold",
				})
				break
			}
		}
	}

	return findings
}
