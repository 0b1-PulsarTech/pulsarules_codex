package contextdiscipline

import (
	"os"
	"path/filepath"
	"strings"
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
	name       string
	source     string
	expect     int
	wantSubstr string
	wantLine   int
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	testCases := append(fireAndForgetCases(), checkFileCleanCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := analyzeSource(t, testCase.source)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
			if testCase.wantSubstr != "" &&
				!strings.Contains(got[0].Message, testCase.wantSubstr) {
				t.Fatalf("message = %q, want substring %q", got[0].Message, testCase.wantSubstr)
			}
			if testCase.wantLine != 0 && got[0].Line != testCase.wantLine {
				t.Fatalf("line = %d, want %d", got[0].Line, testCase.wantLine)
			}
		})
	}
}

func fireAndForgetCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "goroutine body starts with ctx.Done receive",
			source: "package foo\n" +
				"import \"context\"\n" +
				"func run(ctx context.Context) {\n" +
				"\tgo func() {\n" +
				"\t\t<-ctx.Done()\n" +
				"\t\tcleanup()\n" +
				"\t}()\n" +
				"}\n" +
				"func cleanup() {}\n",
			expect:     1,
			wantSubstr: "context.AfterFunc(ctx, cleanup)",
		},
		{
			name: "goroutine doing unrelated work is not fire-and-forget",
			source: "package foo\n" +
				"func run() {\n" +
				"\tgo func() {\n" +
				"\t\tdoWork()\n" +
				"\t}()\n" +
				"}\n" +
				"func doWork() {}\n",
			expect: 0,
		},
		{
			name: "ctx.Done receive after the first statement does not count",
			source: "package foo\n" +
				"import \"context\"\n" +
				"func run(ctx context.Context) {\n" +
				"\tgo func() {\n" +
				"\t\tdoWork()\n" +
				"\t\t<-ctx.Done()\n" +
				"\t}()\n" +
				"}\n" +
				"func doWork() {}\n",
			expect: 0,
		},
	}
}

func checkFileCleanCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name:   "no results has nothing to inspect",
			source: "package foo\nfunc run() {}\n",
			expect: 0,
		},
	}
}

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
	file, err := cache.Parse(path, src)
	if err != nil {
		t.Fatal(err)
	}

	fc := core.FileChange{Path: "foo.go", Extension: ".go"}
	return checkFile(cache.FileSet(), fc, file, NewAnalyzer().reporter)
}
