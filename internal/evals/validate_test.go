package evals

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// embeddedIndex loads the real committed knowledge base once per call, so a
// test exercising SkillExists (backed by the loader's by-id map, not by a
// fabricated Index literal) resolves against real skill ids. Mirrors
// internal/skill/validate's own embeddedIndex helper.
func embeddedIndex(tb testing.TB) *knowledge.Index {
	tb.Helper()

	idx, _, err := knowledge.Load("")
	if err != nil {
		tb.Fatalf("Load: %v", err)
	}
	return idx
}

func validScenario() Scenario {
	return Scenario{
		Skill:  "commits",
		ID:     "s1",
		Prompt: "prompt",
		Trap:   "trap",
		Assertions: []Assertion{
			{
				ID:    "1.1",
				Text:  "text",
				Kind:  KindMachine,
				Check: &Check{Type: CheckContains, Pattern: "x"},
			},
		},
	}
}

func TestValidateScenario(t *testing.T) {
	t.Parallel()

	idx := embeddedIndex(t)

	testCases := []struct {
		name    string
		build   func() Scenario
		wantErr string
	}{
		{
			name:  "valid scenario passes",
			build: validScenario,
		},
		{
			name: "unknown skill is reported",
			build: func() Scenario {
				scenario := validScenario()
				scenario.Skill = "ghost-skill"
				return scenario
			},
			wantErr: `declares unknown skill "ghost-skill"`,
		},
		{
			name: "empty prompt is reported",
			build: func() Scenario {
				scenario := validScenario()
				scenario.Prompt = "  "
				return scenario
			},
			wantErr: "has no prompt",
		},
		{
			name: "empty trap is reported",
			build: func() Scenario {
				scenario := validScenario()
				scenario.Trap = ""
				return scenario
			},
			wantErr: "has no trap",
		},
		{
			name: "no assertions is reported",
			build: func() Scenario {
				scenario := validScenario()
				scenario.Assertions = nil
				return scenario
			},
			wantErr: "declares no assertions",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			problems := validateScenario(idx, testCase.build(), map[string]bool{})

			if testCase.wantErr == "" {
				if len(problems) != 0 {
					t.Fatalf("expected no problems, got %v", problems)
				}
				return
			}
			if !containsSubstring(problems, testCase.wantErr) {
				t.Fatalf("expected a problem containing %q, got %v", testCase.wantErr, problems)
			}
		})
	}
}

// TestValidateScenario_Duplicate asserts the second scenario sharing a
// skill+id pair is reported, but a different id under the same skill is not.
func TestValidateScenario_Duplicate(t *testing.T) {
	t.Parallel()

	idx := embeddedIndex(t)
	seen := map[string]bool{}

	if problems := validateScenario(idx, validScenario(), seen); len(problems) != 0 {
		t.Fatalf("first occurrence should pass, got %v", problems)
	}
	problems := validateScenario(idx, validScenario(), seen)
	if !containsSubstring(problems, "duplicate scenario id") {
		t.Fatalf("expected a duplicate-id problem, got %v", problems)
	}

	other := validScenario()
	other.ID = "s2"
	if problems := validateScenario(idx, other, seen); len(problems) != 0 {
		t.Fatalf("a different id under the same skill should pass, got %v", problems)
	}
}

func TestValidateAssertion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		assertion Assertion
		wantErr   string
	}{
		{
			name: "machine with check passes",
			assertion: Assertion{
				ID:    "1",
				Text:  "t",
				Kind:  KindMachine,
				Check: &Check{Type: CheckContains, Pattern: "x"},
			},
		},
		{
			name:      "judge with no check passes",
			assertion: Assertion{ID: "1", Text: "t", Kind: KindJudge},
		},
		{
			name:      "machine with no check is reported",
			assertion: Assertion{ID: "1", Text: "t", Kind: KindMachine},
			wantErr:   "is machine-graded but declares no check",
		},
		{
			name: "judge with a check is reported",
			assertion: Assertion{
				ID:    "1",
				Text:  "t",
				Kind:  KindJudge,
				Check: &Check{Type: CheckContains, Pattern: "x"},
			},
			wantErr: "is judge-graded but declares a check",
		},
		{
			name:      "unknown kind is reported",
			assertion: Assertion{ID: "1", Text: "t", Kind: "vibes"},
			wantErr:   `has unknown kind "vibes"`,
		},
		{
			name:      "empty text is reported",
			assertion: Assertion{ID: "1", Kind: KindJudge},
			wantErr:   "has no text",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			problems := validateAssertion("scenario", testCase.assertion)

			if testCase.wantErr == "" {
				if len(problems) != 0 {
					t.Fatalf("expected no problems, got %v", problems)
				}
				return
			}
			if !containsSubstring(problems, testCase.wantErr) {
				t.Fatalf("expected a problem containing %q, got %v", testCase.wantErr, problems)
			}
		})
	}
}

// TestValidateCheck_Embedded is the regression guard: every committed eval
// scenario must resolve against the real skills.yaml and carry a trap plus at
// least one well-formed assertion, proving the check stays silent on today's
// truthful data.
func TestValidateCheck_Embedded(t *testing.T) {
	t.Parallel()

	idx := embeddedIndex(t)
	if problems := ValidateCheck(idx); len(problems) != 0 {
		t.Errorf("committed eval scenarios should all validate clean, got %v", problems)
	}
}

func containsSubstring(problems []string, want string) bool {
	for _, problem := range problems {
		if strings.Contains(problem, want) {
			return true
		}
	}
	return false
}
