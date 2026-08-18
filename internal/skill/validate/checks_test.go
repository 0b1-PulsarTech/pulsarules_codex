package validate

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestRouterPresent asserts the mandatory project-router skill is required: the
// embedded index satisfies it, an empty index does not. (SkillExists is backed
// by the loader's by-id map, so the present case uses the real embedded index.)
func TestRouterPresent(t *testing.T) {
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

// TestRuleSummaries asserts a rule with no blockquote summary is reported and
// the embedded index (whose rules all carry one) is silent.
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
	if problems := ruleSummaries(idx); len(problems) != 0 {
		t.Errorf("embedded rules should all carry a summary, got %v", problems)
	}
}

// embeddedIndex loads the real committed knowledge base once per call. It
// backs the "passing" row of every fabricated-failing/passing check test: the
// embedded base is the project's own clean contract, so a check that stays
// silent on it is proven not to false-positive on legitimate data.
func embeddedIndex(tb testing.TB) *knowledge.Index {
	tb.Helper()

	idx, _, err := knowledge.Load("")
	if err != nil {
		tb.Fatalf("Load: %v", err)
	}
	return idx
}

// assertSingleProblem runs check against idx and asserts either exactly one
// problem containing wantProblem (wantProblem set) or no problems at all
// (wantProblem empty) - the shared shape of a fabricated-failing/passing pair.
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

// TestRuleDependencies asserts a rule depending on an unknown rule is reported
// and the embedded index (whose dependencies all resolve) is silent.
func TestRuleDependencies(t *testing.T) {
	t.Parallel()

	embedded := embeddedIndex(t)
	testCases := []struct {
		name        string
		idx         *knowledge.Index
		wantProblem string
	}{
		{
			name: "unknown dependency reported",
			idx: &knowledge.Index{Rules: []knowledge.Rule{
				{ID: "r", Dependencies: []string{"ghost"}},
			}},
			wantProblem: `rule "r" depends on unknown rule "ghost"`,
		},
		{name: "embedded rules resolve cleanly", idx: embedded},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assertSingleProblem(t, ruleDependencies, testCase.idx, testCase.wantProblem)
		})
	}
}

// TestPatternDependencies asserts a pattern depending on an unknown rule is
// reported and the embedded index (whose dependencies all resolve) is silent.
func TestPatternDependencies(t *testing.T) {
	t.Parallel()

	embedded := embeddedIndex(t)
	testCases := []struct {
		name        string
		idx         *knowledge.Index
		wantProblem string
	}{
		{
			name: "unknown dependency reported",
			idx: &knowledge.Index{Patterns: []knowledge.Pattern{
				{ID: "p", Dependencies: []string{"ghost"}},
			}},
			wantProblem: `pattern "p" depends on unknown rule "ghost"`,
		},
		{name: "embedded patterns resolve cleanly", idx: embedded},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assertSingleProblem(t, patternDependencies, testCase.idx, testCase.wantProblem)
		})
	}
}

// TestPatternComposes asserts a pattern composing an unknown pattern is
// reported and the embedded index (whose composes all resolve) is silent.
func TestPatternComposes(t *testing.T) {
	t.Parallel()

	embedded := embeddedIndex(t)
	testCases := []struct {
		name        string
		idx         *knowledge.Index
		wantProblem string
	}{
		{
			name: "unknown composed pattern reported",
			idx: &knowledge.Index{Patterns: []knowledge.Pattern{
				{ID: "p", Composes: []string{"ghost"}},
			}},
			wantProblem: `pattern "p" composes unknown pattern "ghost"`,
		},
		{name: "embedded patterns resolve cleanly", idx: embedded},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assertSingleProblem(t, patternComposes, testCase.idx, testCase.wantProblem)
		})
	}
}

// TestSkillBodies asserts a skill composing a rule with no loaded body (the
// zero value idx.Body falls back to, whether the rule is missing entirely or
// merely empty) is reported, and the embedded index - whose composed rules and
// patterns all carry real bodies - is silent.
func TestSkillBodies(t *testing.T) {
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

// TestSkillCompositions asserts an unresolved composed reference is reported with
// its owner and kind, while a fully-resolving composition is silent.
func TestSkillCompositions(t *testing.T) {
	t.Parallel()

	problems := skillCompositions(&knowledge.Index{Skills: []knowledge.Skill{
		{ID: "x", ComposeRules: []string{"nope"}, ComposeSkills: []string{"ghost"}},
	}})
	if len(problems) != 2 {
		t.Fatalf("expected 2 problems, got %v", problems)
	}
	joined := strings.Join(problems, "\n")
	for _, want := range []string{`skill "x" composes unknown rule "nope"`, `skill "x" composes unknown skill "ghost"`} {
		if !strings.Contains(joined, want) {
			t.Errorf("problems missing %q; got %v", want, problems)
		}
	}
}

// TestReferencesResolve asserts a rule citing an unknown reference is reported and
// the embedded index (whose citations all resolve) is silent.
func TestReferencesResolve(t *testing.T) {
	t.Parallel()

	problems := referencesResolve(&knowledge.Index{Rules: []knowledge.Rule{
		{ID: "r", References: []string{"ghost"}},
	}})
	want := `rule "r" cites unknown reference "ghost"`
	if len(problems) != 1 || !strings.Contains(problems[0], want) {
		t.Fatalf("expected one unknown-reference problem, got %v", problems)
	}

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problems := referencesResolve(idx); len(problems) != 0 {
		t.Errorf("embedded citations should all resolve, got %v", problems)
	}
}
