package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateConfig checks the configuration structure for logical errors.
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration is nil")
	}

	if cfg.Settings.EntropyThreshold < 1.0 || cfg.Settings.EntropyThreshold > 8.0 {
		return fmt.Errorf("invalid entropy_threshold: %f (must be between 1.0 and 8.0)", cfg.Settings.EntropyThreshold)
	}

	for hookName, steps := range cfg.Hooks {
		if len(steps) == 0 {
			continue
		}
		for i, step := range steps {
			if strings.TrimSpace(step.Name) == "" {
				return fmt.Errorf("hook %s step #%d is missing a 'name'", hookName, i+1)
			}
			if step.Type == "" || step.Type == "command" {
				if strings.TrimSpace(step.Run) == "" {
					return fmt.Errorf("hook %s step '%s' is missing 'run' command", hookName, step.Name)
				}
			}
		}
	}

	return nil
}

// MatchGlob tests whether a file path matches a glob pattern, supporting {ext1,ext2} expansions.
func MatchGlob(pattern string, targetPath string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}

	// Normalize paths to forward slashes for matching
	cleanTarget := filepath.ToSlash(targetPath)
	cleanPattern := filepath.ToSlash(pattern)

	// Handle curly braces expansion: *.{ts,tsx,js} -> [*.ts, *.tsx, *.js]
	expandedPatterns := expandBraces(cleanPattern)

	baseName := filepath.Base(cleanTarget)

	for _, p := range expandedPatterns {
		// Match against full path
		if match, _ := filepath.Match(p, cleanTarget); match {
			return true
		}
		// Match against filename only if pattern has no slashes
		if !strings.Contains(p, "/") {
			if match, _ := filepath.Match(p, baseName); match {
				return true
			}
		}
		// Suffix match: e.g. *.go
		if strings.HasPrefix(p, "*.") && strings.HasSuffix(cleanTarget, p[1:]) {
			return true
		}
		// Prefix match: e.g. docs/*
		if strings.HasSuffix(p, "/*") && strings.HasPrefix(cleanTarget, strings.TrimSuffix(p, "/*")+"/") {
			return true
		}
	}

	return false
}

// FilterFilesByGlob filters a slice of files by inclusion and optional exclusion globs.
func FilterFilesByGlob(files []string, includeGlob string, excludeGlob string) []string {
	if includeGlob == "" && excludeGlob == "" {
		return files
	}

	var matched []string
	for _, f := range files {
		if includeGlob != "" && !MatchGlob(includeGlob, f) {
			continue
		}
		if excludeGlob != "" && MatchGlob(excludeGlob, f) {
			continue
		}
		matched = append(matched, f)
	}
	return matched
}

// expandBraces expands a single pattern containing {a,b,c} into multiple patterns.
func expandBraces(pattern string) []string {
	start := strings.Index(pattern, "{")
	end := strings.Index(pattern, "}")
	if start == -1 || end == -1 || end <= start {
		return []string{pattern}
	}

	prefix := pattern[:start]
	suffix := pattern[end+1:]
	options := strings.Split(pattern[start+1:end], ",")

	var results []string
	for _, opt := range options {
		subPattern := prefix + strings.TrimSpace(opt) + suffix
		// Recursively expand further braces if any
		results = append(results, expandBraces(subPattern)...)
	}
	return results
}
