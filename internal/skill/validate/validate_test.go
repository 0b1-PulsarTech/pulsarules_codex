package validate

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

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

func TestValidate_BrokenFolds(t *testing.T) {
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

func TestValidate_ExtraCheckFoldsIntoSameResult(t *testing.T) {
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
