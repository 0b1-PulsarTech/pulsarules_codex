package vacuousdoc

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
)

type vacuousDocCase struct {
	name   string
	source string
	expect int
}

// The kept cases are not invented: each mirrors a real comment two review passes
// judged worth keeping, so a widened check that starts eating them fails here.
var vacuousDocCases = []vacuousDocCase{
	{
		name:   "doc on a constant-returning accessor",
		source: "package p\n\n// ID returns the analyzer's unique identifier.\nfunc ID() string { return \"commit-lint\" }\n",
		expect: 1,
	},
	{
		name:   "doc on a field accessor",
		source: "package p\n\ntype C struct{ n int }\n\n// Size returns the count.\nfunc (c *C) Size() int { return c.n }\n",
		expect: 1,
	},
	{
		name:   "doc on a len() accessor is out of scope",
		source: "package p\n\ntype C struct{ m map[string]int }\n\n// Size returns the count.\nfunc (c *C) Size() int { return len(c.m) }\n",
		expect: 0,
	},
	{
		name:   "no doc comment at all",
		source: "package p\n\nfunc ID() string { return \"x\" }\n",
		expect: 0,
	},
	{
		name:   "two statements",
		source: "package p\n\n// F does two things.\nfunc F() int {\n\tx := 1\n\treturn x\n}\n",
		expect: 0,
	},
	{
		name:   "a branch the comment may explain",
		source: "package p\n\n// F returns nil when absent, which callers must distinguish.\nfunc F(ok bool) error {\n\tif ok {\n\t\treturn nil\n\t}\n\treturn nil\n}\n",
		expect: 0,
	},
	{
		name:   "composed call carries behaviour",
		source: "package p\n\n// Render delegates to the merge path.\nfunc Render(s string) string { return helper(s) }\n\nfunc helper(s string) string { return s }\n",
		expect: 0,
	},
	{
		name:   "no results at all",
		source: "package p\n\n// F does the thing.\nfunc F() { return }\n",
		expect: 0,
	},
	{
		name:   "constructor returning a composite literal",
		source: "package p\n\ntype A struct{}\n\n// NewAnalyzer creates an A.\nfunc NewAnalyzer() *A { return &A{} }\n",
		expect: 0,
	},
	{
		name:   "a why marker earns its place",
		source: "package p\n\n// why: no portable isatty on this GOOS; failing closed is the safe default.\nfunc stdinIsTerminal() bool { return false }\n",
		expect: 0,
	},
	{
		name:   "a second sentence carrying an invariant earns its place",
		source: "package p\n\ntype C struct{ f int }\n\n// FileSet returns the set. All parsed files share it so positions stay consistent.\nfunc (c *C) FileSet() int { return c.f }\n",
		expect: 0,
	},
	{
		name:   "the canonical BackupMessage shape still fires",
		source: "package p\n\n// BackupMessage formats the note Backup's caller reports, so the phrasing stays\n// identical across every install site.\nfunc BackupMessage() string { return \"x\" }\n",
		expect: 1,
	},
	{
		name:   "an abbreviation is not a second sentence",
		source: "package p\n\n// ID returns the id, e.g. commit-lint.\nfunc ID() string { return \"commit-lint\" }\n",
		expect: 1,
	},
}

// TestCheckFile covers both directions: the accessor shape the rule targets, and
// every shape a kept comment takes, so widening the check cannot pass silently.
func TestCheckFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range vacuousDocCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			file, err := cache.Parse("p.go", []byte(testCase.source))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			fc := core.FileChange{Path: "p.go", Extension: ".go"}
			got := checkFile(cache.FileSet(), fc, file, vacuousDocReporter)
			if len(got) != testCase.expect {
				t.Fatalf("findings = %d, want %d: %+v", len(got), testCase.expect, got)
			}
		})
	}
}

// TestAnalyze_NoASTCache asserts the stage guard, so a context assembled without
// the AST cache degrades to silence instead of panicking mid-pipeline.
func TestAnalyze_NoASTCache(t *testing.T) {
	t.Parallel()

	if got := NewAnalyzer().Analyze(&core.AnalysisContext{}); len(got) != 0 {
		t.Errorf("findings = %+v, want none", got)
	}
}
