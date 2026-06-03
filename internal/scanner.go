package internal

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type Match struct {
	RuleName string `json:"rule_name"`
	FileName string `json:"file_name"`
	Content  string `json:"content"`
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

	diffOutput := out.String()
	if len(strings.TrimSpace(diffOutput)) == 0 {
		return nil, nil
	}

	return analyzeDiff(diffOutput, config.Rules)
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

func analyzeDiff(diff string, rules []Rule) ([]Match, error) {
	compiledRules, err := compileRules(rules)
	if err != nil {
		return nil, err
	}

	var matches []Match
	lines := strings.Split(diff, "\n")
	currentFile := "unknown file"

	for _, line := range lines {
		if strings.HasPrefix(line, "+++ b/") {
			currentFile = strings.TrimPrefix(line, "+++ b/")
			continue
		}

		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			cleanLine := strings.TrimPrefix(line, "+")

			for _, cr := range compiledRules {
				if cr.re.MatchString(cleanLine) {
					matches = append(matches, Match{
						RuleName: cr.name,
						FileName: currentFile,
						Content:  strings.TrimSpace(cleanLine),
					})
				}
			}
		}
	}

	return matches, nil
}
