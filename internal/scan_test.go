package internal

import "testing"

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

	matches, err := analyzeDiff(mockDiff, rules)
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
