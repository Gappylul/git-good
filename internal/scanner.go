package internal

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
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

type fileDiffJob struct {
	fileName string
	lines    []string
}

type scanResult struct {
	matches []Match
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

	return analyzeDiffParallel(out.String(), config.Rules, config.Exclude)
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

func analyzeDiffParallel(diff string, rules []Rule, exclude []string) ([]Match, error) {
	compiledRules, err := compileRules(rules)
	if err != nil {
		return nil, err
	}

	jobsList := parseDiffIntoJobs(diff, exclude)
	if len(jobsList) == 0 {
		return nil, nil
	}

	numWorkers := runtime.NumCPU()
	if numWorkers > len(jobsList) {
		numWorkers = len(jobsList)
	}

	jobsChan := make(chan fileDiffJob, len(jobsList))
	resultsChan := make(chan scanResult, len(jobsList))
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go worker(jobsChan, resultsChan, compiledRules, &wg)
	}

	for _, job := range jobsList {
		jobsChan <- job
	}
	close(jobsChan)

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var allMatches []Match
	for res := range resultsChan {
		if len(res.matches) > 0 {
			allMatches = append(allMatches, res.matches...)
		}
	}

	return allMatches, nil
}

func parseDiffIntoJobs(diff string, exclude []string) []fileDiffJob {
	var jobs []fileDiffJob

	normalizedDiff := strings.ReplaceAll(diff, "\r\n", "\n")
	lines := strings.Split(normalizedDiff, "\n")

	var currentJob fileDiffJob
	var active bool
	skipCurrentFile := false

	for _, line := range lines {
		if strings.HasPrefix(line, "+++ b/") {
			currentJob, active, skipCurrentFile = processHeaderLine(line, exclude, currentJob, active, skipCurrentFile, &jobs)
			continue
		}

		if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "diff --git") {
			continue
		}

		if skipCurrentFile || !active {
			continue
		}

		currentJob.lines = append(currentJob.lines, line)
	}

	if active && !skipCurrentFile {
		jobs = append(jobs, currentJob)
	}

	return jobs
}

func processHeaderLine(line string, exclude []string, currentJob fileDiffJob, active bool, skipCurrentFile bool, jobs *[]fileDiffJob) (fileDiffJob, bool, bool) {
	if active && !skipCurrentFile {
		*jobs = append(*jobs, currentJob)
	}

	fileName := strings.TrimPrefix(line, "+++ b/")
	nextSkip := isExcluded(fileName, exclude)

	if nextSkip {
		return fileDiffJob{}, false, true
	}

	nextJob := fileDiffJob{
		fileName: fileName,
		lines:    make([]string, 0, 16),
	}

	return nextJob, true, false
}

func worker(jobs <-chan fileDiffJob, results chan<- scanResult, compiledRules []compiledRule, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		var workerMatches []Match
		for _, line := range job.lines {
			if isLineAddition(line) {
				cleanLine := strings.TrimPrefix(line, "+")
				workerMatches = append(workerMatches, checkRulesOnLine(cleanLine, job.fileName, compiledRules)...)
			}
		}
		results <- scanResult{matches: workerMatches}
	}
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
