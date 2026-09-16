package singleusedecl

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type fakeProvider struct {
	owner string
	ok    bool
}

var _ core.DeclarationProvider = fakeProvider{}

func (fakeProvider) Declarations(_ *core.AnalysisContext) []core.Declaration { return nil }

func (provider fakeProvider) SoleUser(_ *core.AnalysisContext, _ core.Declaration) (string, bool) {
	return provider.owner, provider.ok
}

func TestCheckCandidate(t *testing.T) {
	t.Parallel()

	decl := core.Declaration{Name: "helperCount", File: "a.go", Line: 3}
	reporter := core.NewReporter("single-use-declaration", core.SeverityWarning, core.CatAST)

	testCases := []struct {
		name     string
		provider fakeProvider
		want     []core.Finding
	}{
		{
			name:     "no sole user yields no finding",
			provider: fakeProvider{ok: false},
			want:     nil,
		},
		{
			name:     "a sole user yields a finding naming it",
			provider: fakeProvider{owner: "Track", ok: true},
			want: []core.Finding{{
				AnalyzerID: "single-use-declaration",
				Severity:   core.SeverityWarning,
				Category:   core.CatAST,
				File:       "a.go",
				Line:       3,
				Message:    "helperCount is referenced only inside Track",
				Suggestion: "move the declaration inside Track",
			}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := checkCandidate(&core.AnalysisContext{}, testCase.provider, decl, reporter)
			if len(got) != len(testCase.want) {
				t.Fatalf("findings = %+v, want %+v", got, testCase.want)
			}
			for idx, finding := range got {
				if finding != testCase.want[idx] {
					t.Fatalf("finding[%d] = %+v, want %+v", idx, finding, testCase.want[idx])
				}
			}
		})
	}
}
