package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeDiff(t *testing.T) {
	rules := []Rule{
		{Name: "AWS Access Key", Pattern: "AKIA[0-9A-Z]{16}"},
		{Name: "Generic Secret", Pattern: `(?i)secret\s*=\s*['"][^'"]+['"]`},
	}

	mockDiff := `diff --git a/config.env b/config.env
--- a/config.env
+++ b/config.env
@@ -1,3 +1,4 @@
+KEY=normal_value
+AWS_KEY=AKIAIOSFODNN7EXAMPLE
+MY_SECRET="super-secret-token"
-OLD_KEY=old`

	matches, err := analyzeDiff(mockDiff, rules, nil)
	if err != nil {
		t.Fatalf("analyzeDiff failed unexpectedly: %v", err)
	}

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}

	if matches[0].RuleName != "AWS Access Key" {
		t.Errorf("expected first match to be AWS Key, got %s", matches[0].RuleName)
	}

	if matches[1].RuleName != "Generic Secret" {
		t.Errorf("expected second match to be Generic Secret, got %s", matches[1].RuleName)
	}
}

func TestAnalyzeDiffExclude(t *testing.T) {
	rules := []Rule{
		{Name: "AWS Access Key", Pattern: "AKIA[0-9A-Z]{16}"},
	}

	mockDiff := `diff --git a/secrets.png b/secrets.png
--- a/secrets.png
+++ b/secrets.png
@@ -0,0 +1 @@
+AWS_KEY=AKIAIOSFODNN7EXAMPLE
diff --git a/node_modules/lib/index.js b/node_modules/lib/index.js
--- a/node_modules/lib/index.js
+++ b/node_modules/lib/index.js
@@ -0,0 +1 @@
+AWS_KEY=AKIAIOSFODNN7EXAMPLE
diff --git a/config.env b/config.env
--- a/config.env
+++ b/config.env
@@ -0,0 +1 @@
+AWS_KEY=AKIAIOSFODNN7EXAMPLE`

	exclude := []string{"*.png", "node_modules/*"}

	matches, err := analyzeDiff(mockDiff, rules, exclude)
	if err != nil {
		t.Fatalf("analyzeDiff failed unexpectedly: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("expected 1 match (excluded files should be skipped), got %d", len(matches))
	}

	if matches[0].FileName != "config.env" {
		t.Errorf("expected match in config.env, got %s", matches[0].FileName)
	}
}

func TestAnalyzeDiffOnlyAdditions(t *testing.T) {
	rules := []Rule{
		{Name: "AWS Access Key", Pattern: "AKIA[0-9A-Z]{16}"},
	}

	// Deleted lines should never trigger
	mockDiff := `diff --git a/config.env b/config.env
--- a/config.env
+++ b/config.env
@@ -1 +0,0 @@
-AWS_KEY=AKIAIOSFODNN7EXAMPLE`

	matches, err := analyzeDiff(mockDiff, rules, nil)
	if err != nil {
		t.Fatalf("analyzeDiff failed unexpectedly: %v", err)
	}

	if len(matches) != 0 {
		t.Fatalf("expected 0 matches for deleted lines, got %d", len(matches))
	}
}

func TestIsExcluded(t *testing.T) {
	cases := []struct {
		filename string
		patterns []string
		want     bool
	}{
		{"image.png", []string{"*.png"}, true},
		{"node_modules/lodash/index.js", []string{"node_modules/*"}, true},
		{"go.sum", []string{"go.sum"}, true},
		{"config.env", []string{"*.png", "node_modules/*"}, false},
		{"src/image.png", []string{"*.png"}, true},
	}

	for _, tc := range cases {
		got := isExcluded(tc.filename, tc.patterns)
		if got != tc.want {
			t.Errorf("isExcluded(%q, %v) = %v, want %v", tc.filename, tc.patterns, got, tc.want)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "git-good.yaml")

	yaml := `version: "1"
settings:
  fail_on_match: true
  webhook_url: ""
exclude:
  - "*.png"
rules:
  - name: Test Rule
    pattern: "AKIA[0-9A-Z]{16}"
`
	if err := os.WriteFile(configPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	config, err := parseConfig(data)
	if err != nil {
		t.Fatalf("parseConfig failed: %v", err)
	}

	if config.Version != "1" {
		t.Errorf("expected version 1, got %s", config.Version)
	}
	if !config.Settings.FailOnMatch {
		t.Error("expected fail_on_match to be true")
	}
	if len(config.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(config.Rules))
	}
	if config.Rules[0].Name != "Test Rule" {
		t.Errorf("expected rule name 'Test Rule', got %s", config.Rules[0].Name)
	}
	if len(config.Exclude) != 1 || config.Exclude[0] != "*.png" {
		t.Errorf("unexpected exclude list: %v", config.Exclude)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	_, err := parseConfig([]byte("this: is: not: valid: yaml:::::"))
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}
