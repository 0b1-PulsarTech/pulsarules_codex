package shadowing

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()

	testCases := []struct {
		name   string
		ctx    *pass.Context
		expect int
	}{
		{
			name: "no cache",
			ctx:  &pass.Context{},
		},
		{
			name: "non-go file skipped",
			ctx: &pass.Context{
				ASTCache:     astcache.New(),
				ChangedFiles: []pass.FileChange{{Path: "foo.md", Extension: ".md"}},
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
			t.Parallel()
			runCheckFile(t, a, testCase)
		})
	}
}

type checkFileTestCase struct {
	name      string
	source    string
	expect    int
	wantReuse int
}

func checkFileTestCases() []checkFileTestCase {
	cases := append(checkFileCasesBasics(), checkFileCasesExtra()...)
	return append(cases, checkFileCasesPinned()...)
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

func checkFileCasesPinned() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "shadowing exempts ok",
			source: "package foo\n" +
				"func f(ok bool) {\n" +
				"	if ok {\n" +
				"		ok := 1\n" +
				"		_ = ok\n" +
				"	}\n" +
				"}\n",
		},
		{
			name: "short-decl-reuse still reports ok",
			source: "package foo\n" +
				"func f(ok bool) {\n" +
				"	ok, count := true, 1\n" +
				"	_, _ = ok, count\n" +
				"}\n",
			wantReuse: 1,
		},
		{
			name: "each function reports its own err reuse",
			source: "package foo\n" +
				"func first() (err error) {\n" +
				"	count, err := 1, nil\n" +
				"	_ = count\n" +
				"	return err\n" +
				"}\n" +
				"func second() (err error) {\n" +
				"	count, err := 1, nil\n" +
				"	_ = count\n" +
				"	return err\n" +
				"}\n",
			wantReuse: 2,
		},
		{
			name: "a labeled loop body is walked",
			source: "package foo\n" +
				"func f(v int) {\n" +
				"loop:\n" +
				"	for {\n" +
				"		v := 1\n" +
				"		_ = v\n" +
				"		break loop\n" +
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

	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(path, []byte(source), disk.File); err != nil {
		t.Fatal(err)
	}

	cache := astcache.New()
	f, err := cache.Parse(path, []byte(source))
	if err != nil {
		t.Fatal(err)
	}

	fc := pass.FileChange{Path: "foo.go", Extension: ".go"}
	got := a.checkFile(
		cache.FileSet(), fc, f,
		fileReporters{shadow: a.shadowReporter, reuse: a.reuseReporter},
	)
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

	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(path, []byte(source), disk.File); err != nil {
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

	fc := pass.FileChange{Path: "foo.go", Extension: ".go"}
	findings := a.checkFile(
		cache.FileSet(), fc, f,
		fileReporters{shadow: a.shadowReporter, reuse: a.reuseReporter},
	)
	for _, finding := range findings {
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
