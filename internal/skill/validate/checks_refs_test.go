package validate

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestCrossReferences(t *testing.T) {
	t.Parallel()

	embedded := embeddedIndex(t)
	testCases := []struct {
		name        string
		idx         *knowledge.Index
		wantProblem string
	}{
		{
			name: "rule depends on unknown rule",
			idx: &knowledge.Index{Rules: []knowledge.Rule{
				{ID: "r", Dependencies: []string{"ghost"}},
			}},
			wantProblem: `rule "r" depends on unknown rule "ghost"`,
		},
		{
			name: "pattern depends on unknown rule",
			idx: &knowledge.Index{Patterns: []knowledge.Pattern{
				{ID: "p", Dependencies: []string{"ghost"}},
			}},
			wantProblem: `pattern "p" depends on unknown rule "ghost"`,
		},
		{
			name: "pattern composes unknown pattern",
			idx: &knowledge.Index{Patterns: []knowledge.Pattern{
				{ID: "p", Composes: []string{"ghost"}},
			}},
			wantProblem: `pattern "p" composes unknown pattern "ghost"`,
		},
		{
			name: "rule cites unknown reference",
			idx: &knowledge.Index{Rules: []knowledge.Rule{
				{ID: "r", References: []string{"ghost"}},
			}},
			wantProblem: `rule "r" cites unknown reference "ghost"`,
		},
		{
			name: "pattern cites unknown reference",
			idx: &knowledge.Index{Patterns: []knowledge.Pattern{
				{ID: "p", References: []string{"ghost"}},
			}},
			wantProblem: `pattern "p" cites unknown reference "ghost"`,
		},
		{name: "embedded index resolves cleanly", idx: embedded},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assertSingleProblem(t, crossReferences, testCase.idx, testCase.wantProblem)
		})
	}
}

func TestCrossReferences_AllKindsOrdered(t *testing.T) {
	t.Parallel()

	idx := &knowledge.Index{
		Rules: []knowledge.Rule{
			{ID: "r1", Dependencies: []string{"g1"}, References: []string{"g4"}},
			{ID: "r2", Dependencies: []string{"g2"}, References: []string{"g5"}},
		},
		Patterns: []knowledge.Pattern{
			{
				ID: "p1", Dependencies: []string{"g3"}, Composes: []string{"g6"},
				References: []string{"g7"},
			},
		},
	}
	want := []string{
		`rule "r1" depends on unknown rule "g1"`,
		`rule "r2" depends on unknown rule "g2"`,
		`pattern "p1" depends on unknown rule "g3"`,
		`pattern "p1" composes unknown pattern "g6"`,
		`rule "r1" cites unknown reference "g4"`,
		`rule "r2" cites unknown reference "g5"`,
		`pattern "p1" cites unknown reference "g7"`,
	}
	if got := crossReferences(idx); !slices.Equal(got, want) {
		t.Errorf("crossReferences = %q, want %q", got, want)
	}
}
