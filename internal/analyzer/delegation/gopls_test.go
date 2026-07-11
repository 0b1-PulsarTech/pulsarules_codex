package delegation

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// TestGoplsAnalyzer_Contract pins the identity analysis/specs.go keys on, the
// same way the golangci-lint adapter's contract test does.
func TestGoplsAnalyzer_Contract(t *testing.T) {
	t.Parallel()

	a := NewGoplsAnalyzer()

	testCases := []struct {
		name string
		got  any
		want any
	}{
		{name: "id", got: a.ID(), want: "gopls"},
		{name: "name", got: a.Name(), want: "gopls"},
		{name: "stage", got: a.Stage(), want: core.StageStatic},
		{name: "category", got: a.Category(), want: core.CatSyntax},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if testCase.got != testCase.want {
				t.Errorf("got %v, want %v", testCase.got, testCase.want)
			}
		})
	}

	if req := a.Needs(); req.NeedsAST || req.NeedsGitHistory {
		t.Errorf("Needs() = %+v, want no requirements", req)
	}
}

// TestGoplsAnalyzer_AnalyzeIgnoresContext proves the adapter never reads the
// context it is handed, so a nil Config cannot crash it the way the
// golangci-lint adapter once could.
func TestGoplsAnalyzer_AnalyzeIgnoresContext(t *testing.T) {
	t.Parallel()

	a := NewGoplsAnalyzer()
	for _, finding := range a.Analyze(&core.AnalysisContext{}) {
		if finding.AnalyzerID != "gopls" {
			t.Errorf("AnalyzerID = %q, want %q", finding.AnalyzerID, "gopls")
		}
	}
}
