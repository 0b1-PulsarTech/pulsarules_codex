package evals

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

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
		name        string
		build       func() Scenario
		wantProblem string
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
			wantProblem: `declares unknown skill "ghost-skill"`,
		},
		{
			name: "empty prompt is reported",
			build: func() Scenario {
				scenario := validScenario()
				scenario.Prompt = "  "
				return scenario
			},
			wantProblem: "has no prompt",
		},
		{
			name: "empty trap is reported",
			build: func() Scenario {
				scenario := validScenario()
				scenario.Trap = ""
				return scenario
			},
			wantProblem: "has no trap",
		},
		{
			name: "no assertions is reported",
			build: func() Scenario {
				scenario := validScenario()
				scenario.Assertions = nil
				return scenario
			},
			wantProblem: "declares no assertions",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			problems := validateScenario(idx, testCase.build(), map[string]bool{})

			if testCase.wantProblem == "" {
				if len(problems) != 0 {
					t.Fatalf("expected no problems, got %v", problems)
				}
				return
			}
			if !containsSubstring(problems, testCase.wantProblem) {
				t.Fatalf("want a problem containing %q, got %v", testCase.wantProblem, problems)
			}
		})
	}
}

func TestValidateScenario_DuplicateSkillID(t *testing.T) {
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
		name        string
		assertion   Assertion
		wantProblem string
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
			name:        "machine with no check is reported",
			assertion:   Assertion{ID: "1", Text: "t", Kind: KindMachine},
			wantProblem: "is machine-graded but declares no check",
		},
		{
			name: "judge with a check is reported",
			assertion: Assertion{
				ID:    "1",
				Text:  "t",
				Kind:  KindJudge,
				Check: &Check{Type: CheckContains, Pattern: "x"},
			},
			wantProblem: "is judge-graded but declares a check",
		},
		{
			name:        "unknown kind is reported",
			assertion:   Assertion{ID: "1", Text: "t", Kind: "vibes"},
			wantProblem: `has unknown kind "vibes"`,
		},
		{
			name:        "empty text is reported",
			assertion:   Assertion{ID: "1", Kind: KindJudge},
			wantProblem: "has no text",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			problems := validateAssertion("scenario", testCase.assertion)

			if testCase.wantProblem == "" {
				if len(problems) != 0 {
					t.Fatalf("expected no problems, got %v", problems)
				}
				return
			}
			if !containsSubstring(problems, testCase.wantProblem) {
				t.Fatalf("want a problem containing %q, got %v", testCase.wantProblem, problems)
			}
		})
	}
}

func TestValidateAssertion_Check(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		check       Check
		wantProblem string
	}{
		{"substring passes", Check{Type: CheckContains, Pattern: "("}, ""},
		{"valid regex passes", Check{Type: CheckRegexMatch, Pattern: "a+"}, ""},
		{"unknown type", Check{Type: "regex_mach", Pattern: "x"}, "unknown check type"},
		{"bad regex_absent", Check{Type: CheckRegexAbsent, Pattern: "(unclosed"}, "invalid regex"},
		{"bad regex_match", Check{Type: CheckRegexMatch, Pattern: "[a-"}, "invalid regex"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assertion := Assertion{ID: "1", Text: "t", Kind: KindMachine, Check: &testCase.check}
			problems := validateAssertion("scenario", assertion)
			if testCase.wantProblem == "" {
				if len(problems) != 0 {
					t.Fatalf("expected no problems, got %v", problems)
				}
				return
			}
			if !containsSubstring(problems, testCase.wantProblem) {
				t.Fatalf("want a problem containing %q, got %v", testCase.wantProblem, problems)
			}
		})
	}
}

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
