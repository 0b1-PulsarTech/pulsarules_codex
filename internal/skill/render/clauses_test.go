package render

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestIsLoadBearingClause(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		line string
		want bool
	}{
		{name: "NEVER anywhere", line: "An inner layer NEVER names an outer one.", want: true},
		{name: "MUST anywhere", line: "Every worker MUST gate on ctx.Done().", want: true},
		{name: "MANDATORY anywhere", line: "This step is MANDATORY before commit.", want: true},
		{name: "unbulleted leading No", line: "No new top-level dirs.", want: true},
		{
			name: "dash-bulleted leading No",
			line: "- No dot imports in production code.",
			want: true,
		},
		{
			name: "star-bulleted leading No",
			line: "* No naked returns in long functions.",
			want: true,
		},
		{name: "numbered leading No", line: "3. No unused imports.", want: true},
		{name: "double-digit numbered leading No", line: "12. No em dash anywhere.", want: true},
		{
			name: "arabic-indic numbered marker is not a marker",
			line: "٣. No dot imports.",
			want: false,
		},
		{
			name: "plain lowercase must is not a marker",
			line: "you must read the docs first",
			want: false,
		},
		{
			name: "No mid-sentence is not leading",
			line: "There is No exception to this rule.",
			want: false,
		},
		{
			name: "No-prefixed word is not a marker",
			line: "Notes on the migration follow.",
			want: false,
		},
		{name: "ordinary prose", line: "Use RegisterConstructor* only for wiring.", want: false},
		{name: "empty line", line: "", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := isLoadBearingClause(testCase.line); got != testCase.want {
				t.Errorf("isLoadBearingClause(%q) = %v, want %v", testCase.line, got, testCase.want)
			}
		})
	}
}

func TestStripListMarker(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		line string
		want string
	}{
		{name: "dash marker", line: "- No dot imports.", want: "No dot imports."},
		{name: "star marker", line: "* No dot imports.", want: "No dot imports."},
		{name: "numbered marker", line: "3. No unused imports.", want: "No unused imports."},
		{
			name: "double-digit numbered marker",
			line: "12. No em dash anywhere.",
			want: "No em dash anywhere.",
		},
		{
			name: "tab-separated numbered marker",
			line: "3.\tNo unused imports.",
			want: "No unused imports.",
		},
		{
			name: "numbered marker with multiple spaces",
			line: "3.   No unused imports.",
			want: "No unused imports.",
		},
		{name: "no marker", line: "No new top-level dirs.", want: "No new top-level dirs."},
		{
			name: "digits with no dot are not a marker",
			line: "3 No unused imports.",
			want: "3 No unused imports.",
		},
		{
			name: "digits and dot with no following space are not a marker",
			line: "3.No unused imports.",
			want: "3.No unused imports.",
		},
		{
			name: "arabic-indic digit is not an ASCII marker",
			line: "٣. No dot imports.",
			want: "٣. No dot imports.",
		},
		{
			name: "fullwidth digit is not an ASCII marker",
			line: "３. No dot imports.",
			want: "３. No dot imports.",
		},
		{
			name: "nbsp separator is not ASCII whitespace",
			line: "3. No unused imports.",
			want: "3. No unused imports.",
		},
		{name: "empty line has no marker", line: "", want: ""},
		{name: "lone dot has no marker", line: ".", want: "."},
		{name: "lone digit has no marker", line: "3", want: "3"},
		{
			name: "nested numbering is not a single marker",
			line: "1.2. No unused imports.",
			want: "1.2. No unused imports.",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := stripListMarker(testCase.line); got != testCase.want {
				t.Errorf("stripListMarker(%q) = %q, want %q", testCase.line, got, testCase.want)
			}
		})
	}
}

func TestClausesFromSections(t *testing.T) {
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
		{
			name:     "empty sections map",
			sections: map[string]string{},
			want:     nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := clausesFromSections(testCase.sections)
			if len(got) != len(testCase.want) {
				t.Fatalf("clausesFromSections = %v, want %v", got, testCase.want)
			}
			for i, line := range got {
				if line != testCase.want[i] {
					t.Errorf("clause[%d] = %q, want %q", i, line, testCase.want[i])
				}
			}
		})
	}
}

// TestClausesSurviveComposition proves a MARKED clause (NEVER/MUST/MANDATORY
// or a leading "No ") survives transclusion into every skill composing it.
// Scope is the marked clauses only: the marker is the author's own stop-sign
// signal, and widening it would mean writing MUST into prose to please a test.
func TestClausesSurviveComposition(t *testing.T) {
	t.Parallel()

	idx, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rnd, err := NewRenderer(templates)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	// why: named compositionCase, not testCase - this function's own row
	// variable is named testCase per the table-test convention, and a
	// same-named local type would shadow it at every loop site.
	type compositionCase struct {
		name  string
		skill knowledge.Skill
		want  []string
	}
	testCases := make([]compositionCase, 0, len(idx.Skills))
	totalClauses := 0
	for _, skill := range idx.Skills {
		sources, mergeErr := mergeSources(idx, skill)
		if mergeErr != nil {
			t.Fatalf("mergeSources %q: %v", skill.ID, mergeErr)
		}
		var want []string
		for _, src := range sources {
			want = append(want, clausesFromSections(src.sections)...)
		}
		totalClauses += len(want)
		testCases = append(testCases, compositionCase{name: skill.ID, skill: skill, want: want})
	}
	if totalClauses == 0 {
		t.Fatal(
			"no load-bearing clauses found across the composed corpus; the marker scan may be broken",
		)
	}
	t.Logf("checking %d load-bearing clauses across %d skills", totalClauses, len(testCases))

	for _, testCase := range testCases {
		if len(testCase.want) == 0 {
			continue // nothing load-bearing composed by this skill
		}
		t.Run(testCase.name, func(t *testing.T) {
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
