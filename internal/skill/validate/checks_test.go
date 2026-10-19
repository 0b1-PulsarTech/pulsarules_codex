package validate

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestRouterPresent_Required(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problems := routerPresent(idx); len(problems) != 0 {
		t.Errorf("embedded index should have the router, got %v", problems)
	}
	if problems := routerPresent(&knowledge.Index{}); len(problems) == 0 {
		t.Error("empty index should report the missing router")
	}
}

func TestRuleSummaries(t *testing.T) {
	t.Parallel()

	problems := ruleSummaries(&knowledge.Index{Rules: []knowledge.Rule{{ID: "no-summary"}}})
	want := `rule "no-summary" has no blockquote summary`
	if len(problems) != 1 || !strings.Contains(problems[0], want) {
		t.Fatalf("expected one missing-summary problem, got %v", problems)
	}

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problems = ruleSummaries(idx); len(problems) != 0 {
		t.Errorf("embedded rules should all carry a summary, got %v", problems)
	}
}

func embeddedIndex(tb testing.TB) *knowledge.Index {
	tb.Helper()

	idx, _, err := knowledge.Load("")
	if err != nil {
		tb.Fatalf("Load: %v", err)
	}
	return idx
}

func assertSingleProblem(t *testing.T, check Check, idx *knowledge.Index, wantProblem string) {
	t.Helper()

	problems := check(idx)
	if wantProblem == "" {
		if len(problems) != 0 {
			t.Errorf("expected no problems, got %v", problems)
		}
		return
	}
	if len(problems) != 1 || !strings.Contains(problems[0], wantProblem) {
		t.Fatalf("expected one problem containing %q, got %v", wantProblem, problems)
	}
}

func TestSkillBodies_EmptyOrMissing(t *testing.T) {
	t.Parallel()

	embedded := embeddedIndex(t)
	testCases := []struct {
		name        string
		idx         *knowledge.Index
		wantProblem string
	}{
		{
			name: "empty composed rule body reported",
			idx: &knowledge.Index{Skills: []knowledge.Skill{
				{ID: "s", ComposeRules: []string{"ghost"}},
			}},
			wantProblem: `skill "s" composes rule "ghost" with empty body`,
		},
		{name: "embedded skill bodies are all non-empty", idx: embedded},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assertSingleProblem(t, skillBodies, testCase.idx, testCase.wantProblem)
		})
	}
}

func TestSkillCompositions_Unresolved(t *testing.T) {
	t.Parallel()

	problems := skillCompositions(&knowledge.Index{Skills: []knowledge.Skill{
		{ID: "x", ComposeRules: []string{"nope"}, ComposeSkills: []string{"ghost"}},
	}})
	if len(problems) != 2 {
		t.Fatalf("expected 2 problems, got %v", problems)
	}
	joined := strings.Join(problems, "\n")
	for _, want := range []string{
		`skill "x" composes unknown rule "nope"`,
		`skill "x" composes unknown skill "ghost"`,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("problems missing %q; got %v", want, problems)
		}
	}
}
