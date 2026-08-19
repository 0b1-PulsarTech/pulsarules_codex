package naming

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
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
			name: "no cache and no files",
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

type checkFileTestCase struct {
	name   string
	source string
	expect int
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()

	testCases := append(checkFileNumberedCases(), checkFileNameShapeCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := analyzeSource(t, a, testCase.source)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
		})
	}
}

func checkFileNumberedCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name:   "clean file",
			source: "package foo\nvar UserName string\n",
			expect: 0,
		},
		{
			name:   "numbered name with a bare-stem sibling",
			source: "package foo\nvar user string\nvar user1 string\n",
			expect: 1,
		},
		{
			name:   "numbered name with a counter sibling",
			source: "package foo\nvar user1 string\nvar user2 string\n",
			expect: 2,
		},
		{
			name:   "lone numbered name has nothing to count against",
			source: "package foo\nvar user1 string\n",
			expect: 0,
		},
		{
			name:   "semantic number is never a counter",
			source: "package foo\nvar sha256 string\nvar limit32 int\nvar pow10 int\n",
			expect: 0,
		},
		{
			name:   "func param numbered with a sibling param",
			source: "package foo\nfunc f(x1 int, x2 int) {}\n",
			expect: 2,
		},
		{
			name:   "short var in assign with a sibling",
			source: "package foo\nfunc f() { x1 := 1; x2 := 2; _, _ = x1, x2 }\n",
			expect: 2,
		},
	}
}

func checkFileNameShapeCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name:   "hungarian notation",
			source: "package foo\nvar strName string\n",
			expect: 1,
		},
		{
			name:   "acronym-initial name is not hungarian",
			source: "package foo\ntype IDSelector struct{}\ntype HTTPDoer interface{}\n",
			expect: 0,
		},
		{
			name:   "noise word",
			source: "package foo\nvar data string\n",
			expect: 1,
		},
		{
			name:   "noise word not prefix",
			source: "package foo\nvar tempFile string\n",
			expect: 0,
		},
		{
			name:   "single letter exported",
			source: "package foo\nvar X string\n",
			expect: 1,
		},
		{
			name:   "underscore skipped",
			source: "package foo\nfunc f() { _ = 1 }\n",
			expect: 0,
		},
		{
			name: "a value named after the type it is built from is not noise",
			source: "package foo\n" +
				"func f() { manager := pkg.NewManager(); _ = manager }\n",
			expect: 0,
		},
		{
			name: "a noise word with no type behind it still fires",
			source: "package foo\n" +
				"func find() int { return 0 }\n" +
				"func f() { manager := find(); _ = manager }\n",
			expect: 1,
		},
		{
			// The exemption is per binding, so it cannot leak across the file:
			// one earned exemption must not clear an unrelated binding.
			name: "an exemption does not leak to another binding of the same name",
			source: "package foo\n" +
				"func f() { manager := pkg.NewManager(); _ = manager }\n" +
				"func g() { manager := find(); _ = manager }\n",
			expect: 1,
		},
		{
			name:   "a type declaration is judged on its own name",
			source: "package foo\ntype Data struct{}\n",
			expect: 1,
		},
	}
}

func analyzeSource(t *testing.T, a *Analyzer, source string) []core.Finding {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(path, []byte(source), fsperm.File); err != nil {
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
	return a.checkFile(cache.FileSet(), fc, f, namingReporter)
}
