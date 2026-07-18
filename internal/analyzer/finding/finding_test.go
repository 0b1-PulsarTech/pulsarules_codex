package finding

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
