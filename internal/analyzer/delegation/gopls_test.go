package delegation

import (
	"strings"
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

// TestGoplsAnalyzer_AnalyzeIgnoresContext proves the adapter never reads
// its context, so a nil Config can't crash it like golangci-lint's adapter
// once did. It also asserts every finding stays SeverityInfo: this is an
// availability probe, not a diagnostics source - a higher severity would
// mean it started claiming diagnostics it can't actually produce.
func TestGoplsAnalyzer_AnalyzeIgnoresContext(t *testing.T) {
	t.Parallel()

	a := NewGoplsAnalyzer()
	for _, finding := range a.Analyze(&core.AnalysisContext{}) {
		if finding.AnalyzerID != "gopls" {
			t.Errorf("AnalyzerID = %q, want %q", finding.AnalyzerID, "gopls")
		}
		if finding.Severity != core.SeverityInfo {
			t.Errorf("Severity = %v, want SeverityInfo (probe, not diagnostics)", finding.Severity)
		}
	}
}

// TestGoplsAnalyzer_DescriptionStatesAProbe pins Description() to describing
// an availability probe (not "delegates ... for diagnostics", the overclaim
// this analyzer used to make while Run() only ever shelled "gopls version").
func TestGoplsAnalyzer_DescriptionStatesAProbe(t *testing.T) {
	t.Parallel()

	a := NewGoplsAnalyzer()
	desc := strings.ToLower(a.Description())
	if strings.Contains(desc, "delegates") {
		t.Errorf(
			"Description() = %q, must not claim to delegate diagnostics it cannot produce",
			desc,
		)
	}
	if !strings.Contains(desc, "probe") && !strings.Contains(desc, "availab") {
		t.Errorf("Description() = %q, want it to describe an availability probe", desc)
	}
}
