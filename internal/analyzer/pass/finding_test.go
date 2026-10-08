package pass

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

func TestReporter_At(t *testing.T) {
	t.Parallel()

	reporter := finding.NewReporter("naming", finding.SeverityWarning, finding.CatSyntax)
	got := reporter.At("main.go", 42, "bad name", "rename it")

	want := finding.Finding{
		AnalyzerID: "naming",
		Severity:   finding.SeverityWarning,
		Category:   finding.CatSyntax,
		File:       "main.go",
		Line:       42,
		Message:    "bad name",
		Suggestion: "rename it",
	}
	if got != want {
		t.Fatalf("At() = %+v, want %+v", got, want)
	}
}

func TestReporter_New(t *testing.T) {
	t.Parallel()

	reporter := finding.NewReporter(
		"commit-desc-required",
		finding.SeverityError,
		finding.CatCommit,
	)
	got := reporter.New("commit message must have a description")

	want := finding.Finding{
		AnalyzerID: "commit-desc-required",
		Severity:   finding.SeverityError,
		Category:   finding.CatCommit,
		Message:    "commit message must have a description",
	}
	if got != want {
		t.Fatalf("New() = %+v, want %+v", got, want)
	}
}

func TestReporter_Resolved(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ctx  *Context
		want finding.Severity
	}{
		{
			name: "no config keeps the reporter's own default",
			ctx:  &Context{},
			want: finding.SeverityWarning,
		},
		{
			name: "a configured severity overrides the default",
			ctx: &Context{Config: &AnalysisConfig{Analyzers: map[string]AnalyzerConfig{
				"naming": {Params: map[string]any{"severity": "error"}},
			}}},
			want: finding.SeverityError,
		},
		{
			name: "a different analyzer's config does not leak in",
			ctx: &Context{Config: &AnalysisConfig{Analyzers: map[string]AnalyzerConfig{
				"other": {Params: map[string]any{"severity": "error"}},
			}}},
			want: finding.SeverityWarning,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			reporter := finding.NewReporter("naming", finding.SeverityWarning, finding.CatSyntax)
			resolved := ResolveReporter(reporter, testCase.ctx)
			if got := resolved.At("main.go", 1, "msg", "").Severity; got != testCase.want {
				t.Fatalf("Resolved().At().Severity = %v, want %v", got, testCase.want)
			}
			if reporter.At("main.go", 1, "msg", "").Severity != finding.SeverityWarning {
				t.Fatal("Resolved() mutated the receiver's own severity")
			}
		})
	}
}

func TestReporter_NewWithSuggestion(t *testing.T) {
	t.Parallel()

	reporter := finding.NewReporter("commit-emoji-repeat", finding.SeverityError, finding.CatCommit)
	got := reporter.NewWithSuggestion("emoji already used", "try one of: :wrench:")

	want := finding.Finding{
		AnalyzerID: "commit-emoji-repeat",
		Severity:   finding.SeverityError,
		Category:   finding.CatCommit,
		Message:    "emoji already used",
		Suggestion: "try one of: :wrench:",
	}
	if got != want {
		t.Fatalf("NewWithSuggestion() = %+v, want %+v", got, want)
	}
}
