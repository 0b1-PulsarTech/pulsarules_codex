package complexity

import (
	"go/ast"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestCheckFlagArguments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name:   "bool parameter is a flag argument",
			source: "package foo\n\nfunc run(verbose bool) {}\n",
			want:   []string{"flag argument \"verbose\" in run"},
		},
		{
			name:   "value-like bool name is exempt",
			source: "package foo\n\nfunc run(fallback bool) {}\n",
		},
		{
			name:   "no parameters",
			source: "package foo\n\nfunc run() {}\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cache, file := parseSourceFile(t, testCase.source)
			fn, isFunc := file.Decls[0].(*ast.FuncDecl)
			if !isFunc {
				t.Fatal("first declaration is not a function")
			}
			reporter := finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST)
			fileChange := pass.FileChange{Path: "foo.go"}
			findings := checkFlagArguments(cache.FileSet(), fileChange, fn, reporter)
			if len(findings) != len(testCase.want) {
				t.Fatalf("got %d findings, want %d", len(findings), len(testCase.want))
			}
			for index, finding := range findings {
				if finding.Message != testCase.want[index] {
					t.Errorf("message = %q, want %q", finding.Message, testCase.want[index])
				}
				if finding.RuleID != "flag-arguments" {
					t.Errorf("RuleID = %q, want flag-arguments", finding.RuleID)
				}
			}
		})
	}
}
