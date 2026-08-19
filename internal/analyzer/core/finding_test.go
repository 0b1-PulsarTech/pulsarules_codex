package core

import "testing"

func TestReporter_At(t *testing.T) {
	t.Parallel()

	reporter := NewReporter("naming", SeverityWarning, CatSyntax)
	got := reporter.At("main.go", 42, "bad name", "rename it")

	want := Finding{
		AnalyzerID: "naming",
		Severity:   SeverityWarning,
		Category:   CatSyntax,
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

	reporter := NewReporter("commit-desc-required", SeverityError, CatCommit)
	got := reporter.New("commit message must have a description")

	want := Finding{
		AnalyzerID: "commit-desc-required",
		Severity:   SeverityError,
		Category:   CatCommit,
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
		ctx  *AnalysisContext
		want Severity
	}{
		{
			name: "no config keeps the reporter's own default",
			ctx:  &AnalysisContext{},
			want: SeverityWarning,
		},
		{
			name: "a configured severity overrides the default",
			ctx: &AnalysisContext{Config: &AnalysisConfig{Analyzers: map[string]AnalyzerConfig{
				"naming": {Params: map[string]any{"severity": "error"}},
			}}},
			want: SeverityError,
		},
		{
			name: "a different analyzer's config does not leak in",
			ctx: &AnalysisContext{Config: &AnalysisConfig{Analyzers: map[string]AnalyzerConfig{
				"other": {Params: map[string]any{"severity": "error"}},
			}}},
			want: SeverityWarning,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			reporter := NewReporter("naming", SeverityWarning, CatSyntax)
			resolved := reporter.Resolved(testCase.ctx)
			if got := resolved.At("main.go", 1, "msg", "").Severity; got != testCase.want {
				t.Fatalf("Resolved().At().Severity = %v, want %v", got, testCase.want)
			}
			if reporter.At("main.go", 1, "msg", "").Severity != SeverityWarning {
				t.Fatal("Resolved() mutated the receiver's own severity")
			}
		})
	}
}

func TestReporter_NewWithSuggestion(t *testing.T) {
	t.Parallel()

	reporter := NewReporter("commit-emoji-repeat", SeverityError, CatCommit)
	got := reporter.NewWithSuggestion("emoji already used", "try one of: :wrench:")

	want := Finding{
		AnalyzerID: "commit-emoji-repeat",
		Severity:   SeverityError,
		Category:   CatCommit,
		Message:    "emoji already used",
		Suggestion: "try one of: :wrench:",
	}
	if got != want {
		t.Fatalf("NewWithSuggestion() = %+v, want %+v", got, want)
	}
}
