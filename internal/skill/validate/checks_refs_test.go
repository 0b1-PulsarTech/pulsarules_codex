package validate

import (
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
