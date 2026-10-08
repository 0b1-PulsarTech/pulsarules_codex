package singleuse

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type fakeProvider struct {
	owner string
	ok    bool
}

var _ DeclarationProvider = fakeProvider{}

func (fakeProvider) Declarations(_ *pass.Context) []Declaration { return nil }

func (provider fakeProvider) SoleUser(_ *pass.Context, _ Declaration) (string, bool) {
	return provider.owner, provider.ok
}

func TestCheckCandidate(t *testing.T) {
	t.Parallel()

	decl := Declaration{Name: "helperCount", File: "a.go", Line: 3}
	reporter := finding.NewReporter(
		"single-use-declaration",
		finding.SeverityWarning,
		finding.CatAST,
	)

	testCases := []struct {
		name     string
		provider fakeProvider
		want     []finding.Finding
	}{
		{
			name:     "no sole user yields no finding",
			provider: fakeProvider{ok: false},
			want:     nil,
		},
		{
			name:     "a sole user yields a finding naming it",
			provider: fakeProvider{owner: "Track", ok: true},
			want: []finding.Finding{{
				AnalyzerID: "single-use-declaration",
				Severity:   finding.SeverityWarning,
				Category:   finding.CatAST,
				File:       "a.go",
				Line:       3,
				Message:    "helperCount is referenced only inside Track",
				Suggestion: "move the declaration inside Track",
				Symbol:     "helperCount",
			}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := checkCandidate(&pass.Context{}, testCase.provider, decl, reporter)
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
