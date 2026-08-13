package namedreturns

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

	testCases := append(checkFileDuplicateCases(), checkFileCleanCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := analyzeSource(t, testCase.source)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
		})
	}
}

func checkFileDuplicateCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name:   "two unnamed results of the same type",
			source: "package foo\nfunc f() (string, string) { return \"\", \"\" }\n",
			expect: 1,
		},
		{
			name: "three results with a duplicate among them",
			source: "package foo\n" +
				"func f() (int, int, error) { return 0, 0, nil }\n",
			expect: 1,
		},
		{
			name: "pointer to a qualified generic type duplicated",
			source: "package foo\n" +
				"import \"pkg\"\n" +
				"func f() (*pkg.Lead, *pkg.Lead) { return nil, nil }\n",
			expect: 1,
		},
		{
			name:   "duplicate slice results",
			source: "package foo\nfunc f() ([]byte, []byte) { return nil, nil }\n",
			expect: 1,
		},
		{
			name: "a method with duplicate unnamed results",
			source: "package foo\n" +
				"type T struct{}\n" +
				"func (t T) f() (int, int) { return 0, 0 }\n",
			expect: 1,
		},
		{
			name: "two offending functions in one file",
			source: "package foo\n" +
				"func f() (int, int) { return 0, 0 }\n" +
				"func g() (string, string) { return \"\", \"\" }\n",
			expect: 2,
		},
		{
			name: "interface method with duplicate unnamed results",
			source: "package foo\n" +
				"type T interface {\n" +
				"\tBounds() (int64, int64)\n" +
				"}\n",
			expect: 1,
		},
	}
}

func checkFileCleanCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name:   "different types are never confused",
			source: "package foo\nfunc f() (int, error) { return 0, nil }\n",
			expect: 0,
		},
		{
			name:   "named results are left alone",
			source: "package foo\nfunc f() (name, alias string) { return \"\", \"\" }\n",
			expect: 0,
		},
		{
			name: "named results with distinct types",
			source: "package foo\n" +
				"func f() (out int, err error) { return 0, nil }\n",
			expect: 0,
		},
		{
			name:   "single result has nothing to confuse it with",
			source: "package foo\nfunc f() string { return \"\" }\n",
			expect: 0,
		},
		{
			name:   "no results",
			source: "package foo\nfunc f() {}\n",
			expect: 0,
		},
		{
			name: "named results skip the check even when the type repeats",
			source: "package foo\n" +
				"func f() (a string, b string) { return \"\", \"\" }\n",
			expect: 0,
		},
		{
			name:   "different slice element types are not a duplicate",
			source: "package foo\nfunc f() ([]byte, []string) { return nil, nil }\n",
			expect: 0,
		},
		{
			name: "interface method with named results of the same type",
			source: "package foo\n" +
				"type T interface {\n" +
				"\tBounds() (lo, hi int64)\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "interface method with distinct result types",
			source: "package foo\n" +
				"type T interface {\n" +
				"\tBounds() (int64, error)\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "embedded interface has no method name and must not panic",
			source: "package foo\n" +
				"import \"io\"\n" +
				"type T interface {\n" +
				"\tio.Reader\n" +
				"}\n",
			expect: 0,
		},
		{
			// why: checkFile only inspects *ast.FuncDecl and *ast.InterfaceType,
			// so a func-typed struct field is never visited; pinning that here.
			name: "a func-typed struct field is not reported",
			source: "package foo\n" +
				"type T struct {\n" +
				"\tF func() (int, int)\n" +
				"}\n",
			expect: 0,
		},
	}
}

func TestDuplicateResultTypeMessage(t *testing.T) {
	t.Parallel()

	source := "package foo\nfunc bounds() (int64, int64) { return 0, 0 }\n"
	got := analyzeSource(t, source)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1: %v", len(got), got)
	}

	want := "bounds returns more than one unnamed int64, so only the order tells them apart"
	if got[0].Message != want {
		t.Fatalf("message = %q, want %q", got[0].Message, want)
	}
}

// why: mirrors naming_test.go's analyzeSource - write, read back, parse.
func analyzeSource(t *testing.T, source string) []core.Finding {
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
	return checkFile(cache.FileSet(), fc, f)
}
