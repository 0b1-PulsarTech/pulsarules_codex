package shadowing

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

	for _, testCase := range checkFileTestCases() {
		t.Run(testCase.name, func(t *testing.T) {
			runCheckFile(t, a, testCase)
		})
	}
}

type checkFileTestCase struct {
	name   string
	source string
	// expect counts shadowing findings; wantReuse counts short-decl-reuse
	// ones. Both are asserted so a fix that merely relabels a finding cannot
	// pass by keeping the total steady.
	expect    int
	wantReuse int
}

func checkFileTestCases() []checkFileTestCase {
	return append(checkFileCasesBasics(), checkFileCasesExtra()...)
}

func checkFileCasesBasics() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name:   "no shadowing",
			source: "package foo\nfunc f() { x := 1; _ = x }\n",
			expect: 0,
		},
		{
			name: "block shadows outer var",
			source: "package foo\n" +
				"func f() {\n" +
				"	x := 1\n" +
				"	{\n" +
				"		x := 2\n" +
				"		_ = x\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "if shadows outer var",
			source: "package foo\n" +
				"func f() {\n" +
				"	x := 1\n" +
				"	if true {\n" +
				"		x := 2\n" +
				"		_ = x\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "for shadows outer var",
			source: "package foo\n" +
				"func f() {\n" +
				"	x := 1\n" +
				"	for i := 0; i < 10; i++ {\n" +
				"		x := 2\n" +
				"		_ = x\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "param reused in block",
			source: "package foo\n" +
				"func f(x int) {\n" +
				"	{\n" +
				"		x := 2\n" +
				"		_ = x\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
	}
}

func checkFileCasesExtra() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "builtin shadowed",
			source: "package foo\n" +
				"func f() {\n" +
				"	len := 1\n" +
				"	_ = len\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "if init shadowed param",
			source: "package foo\n" +
				"func f(x int) {\n" +
				"	if x := 1; true {\n" +
				"		_ = x\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "range shadowed param",
			source: "package foo\n" +
				"func f(x int) {\n" +
				"	for x := range []int{1, 2} {\n" +
				"		_ = x\n" +
				"	}\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "underscore not flagged",
			source: "package foo\n" +
				"func f() {\n" +
				"	_ := 1\n" +
				"	_ = 2\n" +
				"}\n",
			expect: 0,
		},
	}
}

func TestCheckFileMessage(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()

	testCases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "variable shadow message names outer variable once",
			source: "package foo\n" +
				"func f() {\n" +
				"	x := 1\n" +
				"	{\n" +
				"		x := 2\n" +
				"		_ = x\n" +
				"	}\n" +
				"}\n",
			want: `"x" shadows an outer variable`,
		},
		{
			name: "builtin shadow message names outer builtin",
			source: "package foo\n" +
				"func f() {\n" +
				"	len := 1\n" +
				"	_ = len\n" +
				"}\n",
			want: `"len" shadows an outer builtin`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := analyzeMessage(t, a, testCase.source)
			if got != testCase.want {
				t.Fatalf("message = %q, want %q", got, testCase.want)
			}
		})
	}
}

func analyzeMessage(t *testing.T, a *Analyzer, source string) string {
	t.Helper()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "foo.go")
	if err := os.WriteFile(path, []byte(source), fsperm.File); err != nil {
		t.Fatal(err)
	}

	cache := astcache.New()
	f, err := cache.Parse(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}

	fc := core.FileChange{Path: "foo.go", Extension: ".go"}
	got := a.checkFile(cache.FileSet(), fc, f)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1: %v", len(got), got)
	}
	return got[0].Message
}

func runCheckFile(t *testing.T, a *Analyzer, testCase checkFileTestCase) {
	t.Helper()

	shadows, reuses := countByRule(t, a, testCase.source)
	if shadows != testCase.expect {
		t.Errorf("got %d shadowing findings, want %d", shadows, testCase.expect)
	}
	if reuses != testCase.wantReuse {
		t.Errorf("got %d short-decl-reuse findings, want %d", reuses, testCase.wantReuse)
	}
}

func countByRule(t *testing.T, a *Analyzer, source string) (shadows, reuses int) {
	t.Helper()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "foo.go")
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
	for _, finding := range a.checkFile(cache.FileSet(), fc, f) {
		switch finding.AnalyzerID {
		case "shadowing":
			shadows++
		case "short-decl-reuse":
			reuses++
		default:
			t.Fatalf("unexpected analyzer id %q", finding.AnalyzerID)
		}
	}
	return shadows, reuses
}
