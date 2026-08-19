package controlflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()

	testCases := []struct {
		name   string
		ctx    *core.AnalysisContext
		expect int
	}{
		{
			name: "no cache",
			ctx:  &core.AnalysisContext{},
		},
		{
			name: "non-go file skipped",
			ctx: &core.AnalysisContext{
				ASTCache:     astcache.New(),
				ChangedFiles: []core.FileChange{{Path: "foo.md", Extension: ".md"}},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := a.Analyze(testCase.ctx)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d", len(got), testCase.expect)
			}
		})
	}
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()

	testCases := []struct {
		name   string
		source string
		expect int
	}{
		{
			name:   "clean if",
			source: "package foo\nfunc f() { if true { return }; x := 1; _ = x }\n",
			expect: 0,
		},
		{
			name: "else return",
			source: "package foo\n" +
				"func f() {\n" +
				"	if true {\n" +
				"	} else {\n" +
				"		return\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "else panic",
			source: "package foo\n" +
				"func f() {\n" +
				"	if true {\n" +
				"	} else {\n" +
				"		panic(\"x\")\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "else for loop",
			source: "package foo\n" +
				"func f() {\n" +
				"	if true {\n" +
				"	} else {\n" +
				"		for i := 0; i < 10; i++ {}\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "else break",
			source: "package foo\n" +
				"func f() {\n" +
				"	for i := 0; i < 10; i++ {\n" +
				"		if true {\n" +
				"		} else {\n" +
				"			break\n" +
				"		}\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "else if no finding",
			source: "package foo\n" +
				"func f() {\n" +
				"	if true {\n" +
				"	} else if false {\n" +
				"		return\n" +
				"	}\n" +
				"}\n",
			expect: 0,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runCheckFile(t, a, testCase.source, testCase.expect)
		})
	}
}

func runCheckFile(t *testing.T, a *Analyzer, source string, expect int) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	cache := astcache.New()
	f, err := cache.Parse(path, src)
	if err != nil {
		t.Fatal(err)
	}

	fc := core.FileChange{Path: "foo.go", Extension: ".go"}
	got := a.checkFile(cache.FileSet(), fc, f, controlFlowReporter)
	if len(got) != expect {
		t.Fatalf("got %d findings, want %d: %v", len(got), expect, got)
	}
}
