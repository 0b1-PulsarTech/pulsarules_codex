package render

import (
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func loadRenderer(t *testing.T) (*knowledge.Index, *Renderer) {
	t.Helper()

	idx, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rnd, err := NewRenderer(templates)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	return idx, rnd
}

func assertSkillBody(t *testing.T, idx *knowledge.Index, skill knowledge.Skill, body string) {
	t.Helper()

	if !strings.HasPrefix(body, "---\nname: "+skill.ID+"\n") {
		t.Errorf("%q: missing frontmatter name", skill.ID)
	}
	if skill.ID == knowledge.RouterSkillID {
		return
	}
	has, normErr := HasNormativeSection(idx, skill)
	if normErr != nil {
		t.Errorf("%q: HasNormativeSection: %v", skill.ID, normErr)
	} else if !has {
		t.Errorf(
			"%q: rendered skill carries no composed must/forbidden/validation section",
			skill.ID,
		)
	}
	if strings.Contains(body, "Apply every applicable rule") {
		t.Errorf("%q: rendered skill still carries the generic checklist footer", skill.ID)
	}
}

func TestRenderSkill_NormativeSection(t *testing.T) {
	t.Parallel()

	idx, rnd := loadRenderer(t)
	for _, skill := range idx.Skills {
		body, renderErr := rnd.RenderSkill(idx, skill, nil)
		if renderErr != nil {
			t.Errorf("render %q: %v", skill.ID, renderErr)
			continue
		}
		assertSkillBody(t, idx, skill, body)
	}

	router, err := rnd.renderRouterDoc(idx, nil)
	if err != nil {
		t.Fatalf("renderRouterDoc: %v", err)
	}
	for _, want := range []string{
		"MANDATORY orchestrator", "Dispatch table", "Tests are not optional routing",
	} {
		if !strings.Contains(router, want) {
			t.Errorf("router missing %q", want)
		}
	}
}

func TestComposedLinters_NoDuplicates(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for _, skill := range idx.Skills {
		t.Run(skill.ID, func(t *testing.T) {
			t.Parallel()
			seen := map[string]bool{}
			for _, linter := range composedLinters(idx, skill) {
				if seen[linter] {
					t.Errorf(
						"linter %q appears more than once in %q's composed linters",
						linter,
						skill.ID,
					)
				}
				seen[linter] = true
			}
		})
	}
}

func TestRender_Workflow(t *testing.T) {
	t.Parallel()

	idx, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rnd, err := NewRenderer(templates)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	wf, ok := idx.Workflow("refactoring")
	if !ok {
		t.Fatal("refactoring workflow not found in index")
	}
	body, renderErr := rnd.RenderWorkflow(idx, wf)
	if renderErr != nil {
		t.Fatalf("RenderWorkflow: %v", renderErr)
	}
	if !strings.HasPrefix(body, "---\nname: refactoring\n") {
		t.Errorf("RenderWorkflow: missing frontmatter, got prefix: %q", body[:min(len(body), 80)])
	}
	for _, want := range []string{"# Refactoring", "## Steps", "put on one hat"} {
		if !strings.Contains(body, want) {
			t.Errorf("RenderWorkflow output missing %q", want)
		}
	}
}

func TestRender_UnknownComposition(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	idx := &knowledge.Index{}
	rnd, err := NewRenderer(templates)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	if _, err = rnd.renderBody(
		idx,
		knowledge.Skill{ID: "x", ComposeRules: []string{"nope"}},
	); err == nil {
		t.Fatal("expected error for unknown composed rule, got nil")
	}
}

func assertCursorBody(t *testing.T, skill knowledge.Skill, body string) {
	t.Helper()

	if !strings.HasPrefix(body, "---\ndescription: ") {
		t.Errorf("%q: missing Cursor frontmatter description", skill.ID)
	}
	if !strings.Contains(body, "alwaysApply: false") {
		t.Errorf("%q: expected alwaysApply: false, got:\n%s", skill.ID, body)
	}
	if strings.HasPrefix(body, "---\nname: ") {
		t.Errorf("%q: cursor rule still carries SKILL.md's name field", skill.ID)
	}
}

func TestRenderSkillCursor_AllSkills(t *testing.T) {
	t.Parallel()

	idx, rnd := loadRenderer(t)
	for _, skill := range idx.Skills {
		body, renderErr := rnd.RenderSkillCursor(idx, skill, nil)
		if renderErr != nil {
			t.Errorf("render cursor %q: %v", skill.ID, renderErr)
			continue
		}
		assertCursorBody(t, skill, body)
	}

	router, err := rnd.renderCursorRouter(idx, nil)
	if err != nil {
		t.Fatalf("renderCursorRouter: %v", err)
	}
	for _, want := range []string{
		"MANDATORY orchestrator", "Dispatch table", "alwaysApply: false",
	} {
		if !strings.Contains(router, want) {
			t.Errorf("cursor router missing %q", want)
		}
	}
}

func loadBearingClause(line string) bool {
	if strings.Contains(line, "NEVER") ||
		strings.Contains(line, "MUST") ||
		strings.Contains(line, "MANDATORY") {
		return true
	}
	trimmed := strings.TrimSpace(line)
	var rest string
	switch {
	case strings.HasPrefix(trimmed, "- "), strings.HasPrefix(trimmed, "* "):
		rest = trimmed[2:]
	default:
		rest = stripNumberedMarker(trimmed)
	}
	return strings.HasPrefix(rest, "No ")
}

func stripNumberedMarker(trimmed string) string {
	afterDigits := strings.TrimLeftFunc(trimmed, func(char rune) bool {
		return char >= '0' && char <= '9'
	})
	if afterDigits == trimmed || !strings.HasPrefix(afterDigits, ".") {
		return trimmed
	}
	afterDot := afterDigits[1:]
	rest := strings.TrimLeftFunc(afterDot, func(char rune) bool {
		return char == '\t' || char == '\n' || char == '\f' || char == '\r' || char == ' '
	})
	if rest == afterDot {
		return trimmed
	}
	return rest
}

func sourceClauses(sections map[string]string) []string {
	var clauses []string
	for _, key := range []string{"must", "forbidden"} {
		for line := range strings.SplitSeq(sections[key], "\n") {
			if loadBearingClause(line) {
				clauses = append(clauses, line)
			}
		}
	}
	return clauses
}

func TestSourceClauses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		sections map[string]string
		want     []string
	}{
		{
			name:     "no must or forbidden section",
			sections: map[string]string{"when": "NEVER skip this."},
			want:     nil,
		},
		{
			name: "must section mixes marked and unmarked lines",
			sections: map[string]string{
				"must": "1. Use the injector.\n2. MUST gate every worker on ctx.Done().",
			},
			want: []string{"2. MUST gate every worker on ctx.Done()."},
		},
		{
			name: "forbidden section only",
			sections: map[string]string{
				"forbidden": "- Global var state.\n- NEVER store the ctx on a struct.",
			},
			want: []string{"- NEVER store the ctx on a struct."},
		},
		{
			name: "both sections contribute in must-then-forbidden order",
			sections: map[string]string{
				"must":      "- No output arguments.",
				"forbidden": "- MANDATORY review before merge.",
			},
			want: []string{"- No output arguments.", "- MANDATORY review before merge."},
		},
		{name: "empty sections map", sections: map[string]string{}, want: nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := sourceClauses(testCase.sections); !slices.Equal(got, testCase.want) {
				t.Errorf("sourceClauses = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestLoadBearingClause(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		line string
		want bool
	}{
		{name: "NEVER anywhere", line: "An inner layer NEVER names an outer one.", want: true},
		{name: "MUST anywhere", line: "Every worker MUST gate on ctx.Done().", want: true},
		{name: "MANDATORY anywhere", line: "This step is MANDATORY.", want: true},
		{name: "unbulleted leading No", line: "No new top-level dirs.", want: true},
		{name: "dash-bulleted leading No", line: "- No dot imports.", want: true},
		{name: "star-bulleted leading No", line: "* No naked returns.", want: true},
		{name: "numbered leading No", line: "12. No unused imports.", want: true},
		{name: "arabic-indic digit is no marker", line: "\u0663. No dot imports.", want: false},
		{name: "tab-separated numbered No", line: "3.\tNo unused imports.", want: true},
		{name: "multi-space numbered No", line: "3.   No unused imports.", want: true},
		{name: "fullwidth digit is no marker", line: "\uFF13. No dot imports.", want: false},
		{name: "nbsp separator is no marker", line: "3.\u00A0No unused imports.", want: false},
		{name: "digits and dot without space", line: "3.No unused imports.", want: false},
		{name: "digits without dot", line: "3 No unused imports.", want: false},
		{name: "nested numbering", line: "1.2. No unused imports.", want: false},
		{name: "lone dot", line: ".", want: false},
		{name: "lone digit", line: "3", want: false},
		{name: "lowercase must", line: "you must read the docs", want: false},
		{name: "No mid-sentence", line: "There is No exception.", want: false},
		{name: "No-prefixed word", line: "Notes on the migration.", want: false},
		{name: "ordinary prose", line: "Use RegisterConstructor* only for wiring.", want: false},
		{name: "empty line", line: "", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := loadBearingClause(testCase.line); got != testCase.want {
				t.Errorf("loadBearingClause(%q) = %v, want %v", testCase.line, got, testCase.want)
			}
		})
	}
}

type compositionCase struct {
	skill knowledge.Skill
	want  []string
}

func compositionCases(t *testing.T, idx *knowledge.Index) []compositionCase {
	t.Helper()

	testCases := make([]compositionCase, 0, len(idx.Skills))
	for _, skill := range idx.Skills {
		sources, mergeErr := mergeSources(idx, skill)
		if mergeErr != nil {
			t.Fatalf("mergeSources %q: %v", skill.ID, mergeErr)
		}
		perSource := make([][]string, 0, len(sources))
		for _, src := range sources {
			perSource = append(perSource, sourceClauses(src.sections))
		}
		want := slices.Concat(perSource...)
		testCases = append(testCases, compositionCase{skill: skill, want: want})
	}
	return testCases
}

func TestClausesSurviveComposition(t *testing.T) {
	t.Parallel()

	idx, rnd := loadRenderer(t)
	testCases := compositionCases(t, idx)
	total := 0
	for _, testCase := range testCases {
		total += len(testCase.want)
	}
	const guardedFloor = 47
	if total < guardedFloor {
		t.Fatalf("guarded clauses = %d, want at least %d (MUST, NEVER, MANDATORY or a leading No)",
			total, guardedFloor)
	}
	t.Logf("checking %d load-bearing clauses across %d skills", total, len(testCases))

	for _, testCase := range testCases {
		t.Run(testCase.skill.ID, func(t *testing.T) {
			t.Parallel()

			body, renderErr := rnd.RenderSkill(idx, testCase.skill, nil)
			if renderErr != nil {
				t.Fatalf("render %q: %v", testCase.skill.ID, renderErr)
			}
			for _, clause := range testCase.want {
				if !strings.Contains(body, clause) {
					t.Errorf(
						"skill %q dropped a load-bearing clause: %q",
						testCase.skill.ID,
						clause,
					)
				}
			}
		})
	}
}
