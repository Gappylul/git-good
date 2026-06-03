package internal

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Match struct {
	RuleName string
	FileName string
	Content  string
}

type compiledRule struct {
	name string
	re   *regexp.Regexp
}

func ScanStagedChanges(config *Config) ([]Match, error) {
	cmd := exec.Command("git", "diff", "--cached", "-U0")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git diff failed: %s", stderr.String())
	}

	if out.Len() == 0 {
		return nil, nil
	}

	return analyzeDiff(out.String(), config.Rules, config.Exclude)
}

func compileRules(rules []Rule) ([]compiledRule, error) {
	compiledRules := make([]compiledRule, 0, len(rules))
	for _, r := range rules {
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern for rule %q: %w", r.Name, err)
		}
		compiledRules = append(compiledRules, compiledRule{name: r.Name, re: re})
	}
	return compiledRules, nil
}

func isExcluded(filename string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		if matched, err := filepath.Match(pattern, filename); err == nil && matched {
			return true
		}

		if matched, err := filepath.Match(pattern, filepath.Base(filename)); err == nil && matched {
			return true
		}

		cleanPattern := strings.TrimSuffix(pattern, "*")
		cleanPattern = strings.TrimSuffix(cleanPattern, "/")

		if strings.HasPrefix(filename, cleanPattern+"/") || filename == cleanPattern {
			return true
		}
	}
	return false
}

func analyzeDiff(diff string, rules []Rule, exclude []string) ([]Match, error) {
	compiledRules, err := compileRules(rules)
	if err != nil {
		return nil, err
	}

	var matches []Match
	lines := strings.Split(diff, "\n")
	currentFile := ""
	skipCurrentFile := false

	for _, line := range lines {
		if strings.HasPrefix(line, "+++ b/") {
			currentFile = strings.TrimPrefix(line, "+++ b/")
			skipCurrentFile = isExcluded(currentFile, exclude)
			continue
		}

		if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "diff --git") {
			continue
		}

		if skipCurrentFile || currentFile == "" {
			continue
		}

		if isLineAddition(line) {
			cleanLine := strings.TrimPrefix(line, "+")
			matches = append(matches, checkRulesOnLine(cleanLine, currentFile, compiledRules)...)
		}
	}

	return matches, nil
}

func isLineAddition(line string) bool {
	return strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++")
}

func checkRulesOnLine(line string, fileName string, compiledRules []compiledRule) []Match {
	var lineMatches []Match
	for _, cr := range compiledRules {
		if cr.re.MatchString(line) {
			lineMatches = append(lineMatches, Match{
				RuleName: cr.name,
				FileName: fileName,
				Content:  strings.TrimSpace(line),
			})
		}
	}
	return lineMatches
}
