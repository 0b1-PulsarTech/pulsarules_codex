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
