//go:build unix

package command

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func TestAnalyzerIdentity(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer(nil)
	if analyzer.ID() != AnalyzerID || analyzer.Stage() != core.StageStatic {
		t.Fatalf("ID %q Stage %v", analyzer.ID(), analyzer.Stage())
	}
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	missing := core.DelegatedCommand{
		ID: "first", Format: "dart-analyze-json", Run: []string{"no-such-tool-xyz"},
	}
	broken := core.DelegatedCommand{ID: "second", Format: "nope", Run: []string{"x"}}

	testCases := []struct {
		name   string
		config *core.AnalysisConfig
		want   []string
	}{
		{"nil config", nil, nil},
		{"nothing configured", &core.AnalysisConfig{}, nil},
		{
			"each command is reported on its own",
			&core.AnalysisConfig{Delegated: []core.DelegatedCommand{missing, broken}},
			[]string{"first/unavailable", "second/config"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &core.AnalysisContext{ProjectDir: root, Config: testCase.config}
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
