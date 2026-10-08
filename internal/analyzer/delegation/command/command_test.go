//go:build unix

package command

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestAnalyzerIdentity(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(nil)
	if analyzer.ID() != ID || analyzer.Stage() != finding.StageStatic {
		t.Fatalf("ID %q Stage %v", analyzer.ID(), analyzer.Stage())
	}
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	missing := pass.DelegatedCommand{
		ID: "first", Format: "dart-analyze-json", Run: []string{"no-such-tool-xyz"},
	}
	broken := pass.DelegatedCommand{ID: "second", Format: "nope", Run: []string{"x"}}

	testCases := []struct {
		name   string
		config *pass.AnalysisConfig
		want   []string
	}{
		{"nil config", nil, nil},
		{"nothing configured", &pass.AnalysisConfig{}, nil},
		{
			"each command is reported on its own",
			&pass.AnalysisConfig{Delegated: []pass.DelegatedCommand{missing, broken}},
			[]string{"first/unavailable", "second/config"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &pass.Context{ProjectDir: root, Config: testCase.config}
			findings := NewAnalyzer(nil).Analyze(ctx)
			if len(findings) != len(testCase.want) {
				t.Fatalf("findings = %+v, want ids %v", findings, testCase.want)
			}
			for idx, finding := range findings {
				if finding.AnalyzerID != testCase.want[idx] {
					t.Errorf(
						"finding %d = %q, want %q",
						idx,
						finding.AnalyzerID,
						testCase.want[idx],
					)
				}
			}
		})
	}
}
