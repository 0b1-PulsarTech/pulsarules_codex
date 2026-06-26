package validate

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestValidate_Embedded asserts the committed knowledge base validates cleanly.
func TestValidate_Embedded(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if result := Validate(idx); !result.OK() {
		t.Fatalf("validation failed:\n  - %v", result.Errors)
	}
}

// TestValidate_Broken asserts a malformed index fails: it has no router and a
// skill that composes an unknown rule, so Validate folds both problems in.
func TestValidate_Broken(t *testing.T) {
	t.Parallel()

	idx := &knowledge.Index{Skills: []knowledge.Skill{
		{ID: "x", ComposeRules: []string{"missing-rule"}},
	}}
	result := Validate(idx)
	if result.OK() {
		t.Fatal("expected validation to fail, got OK")
	}
	if len(result.Errors) < 2 {
		t.Errorf(
			"expected at least the missing-rule and missing-router problems, got %v",
			result.Errors,
		)
	}
}

// TestValidate_ExtraCheck asserts an extra check runs alongside the built-in
// pipeline and its problems fold into the same result, so a caller (e.g. the
// CLI folding in render.LintSections) never needs a second call site.
func TestValidate_ExtraCheck(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	extra := func(*knowledge.Index) []string { return []string{"extra problem"} }
	result := Validate(idx, extra)
	if result.OK() {
		t.Fatal("expected validation to fail, got OK")
	}
	if len(result.Errors) != 1 || result.Errors[0] != "extra problem" {
		t.Fatalf("expected exactly [\"extra problem\"], got %v", result.Errors)
	}
}
