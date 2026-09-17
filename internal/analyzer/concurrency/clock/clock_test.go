package clock

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
			name: "no cache and no files",
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

type checkFileTestCase struct {
	name   string
	source string
	expect int
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	testCases := append(checkFileSleepCases(), checkFileClockFieldCases()...)
	testCases = append(testCases, checkFileCleanCases()...)
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

func checkFileSleepCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "a sleep inside synctest.Test drives the fake clock, not the wall clock",
			source: "package foo\n" +
				"import (\n\t\"testing\"\n\t\"testing/synctest\"\n\t\"time\"\n)\n" +
				"func TestX(t *testing.T) {\n" +
				"\tsynctest.Test(t, func(t *testing.T) {\n" +
				"\t\ttime.Sleep(700 * time.Millisecond)\n" +
				"\t})\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "a sleep beside a bubble is still a real sleep",
			source: "package foo\n" +
				"import (\n\t\"testing\"\n\t\"testing/synctest\"\n\t\"time\"\n)\n" +
				"func TestX(t *testing.T) {\n" +
				"\tsynctest.Test(t, func(t *testing.T) {})\n" +
				"\ttime.Sleep(time.Second)\n" +
				"}\n",
			expect: 1,
		},
		{
			name:   "a bare time.Sleep call",
			source: "package foo\nimport \"time\"\nfunc f() { time.Sleep(time.Second) }\n",
			expect: 1,
		},
		{
			name: "two time.Sleep calls in one function",
			source: "package foo\n" +
				"import \"time\"\n" +
				"func f() {\n" +
				"\ttime.Sleep(time.Millisecond)\n" +
				"\ttime.Sleep(time.Second)\n" +
				"}\n",
			expect: 2,
		},
		{
			name: "a differently named Sleep method is not time.Sleep",
			source: "package foo\n" +
				"type limiter struct{}\n" +
				"func (l limiter) Sleep() {}\n" +
				"func f(l limiter) { l.Sleep() }\n",
			expect: 0,
		},
	}
}

func checkFileClockFieldCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "a now func() time.Time clock-injection field",
			source: "package foo\n" +
				"import \"time\"\n" +
				"type Debouncer struct {\n" +
				"\tnow func() time.Time\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "a differently named clock field is still a seam",
			source: "package foo\n" +
				"import \"time\"\n" +
				"type Worker struct {\n" +
				"\tclockFn func() time.Time\n" +
				"}\n",
			expect: 1,
		},
	}
}

func checkFileCleanCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name:   "no results has nothing to inspect",
			source: "package foo\nfunc f() {}\n",
			expect: 0,
		},
		{
			name: "an injected Clock interface is not a seam",
			source: "package foo\n" +
				"import \"time\"\n" +
				"type Clock interface {\n" +
				"\tNow() time.Time\n" +
				"}\n" +
				"type Worker struct {\n" +
				"\tclock Clock\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "a field returning time.Time with a parameter is not a bare seam",
			source: "package foo\n" +
				"import \"time\"\n" +
				"type Worker struct {\n" +
				"\tat func(string) time.Time\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "a field returning a different type is left alone",
			source: "package foo\n" +
				"type Worker struct {\n" +
				"\tid func() string\n" +
				"}\n",
			expect: 0,
		},
	}
}

func analyzeSource(t *testing.T, source string) []pass.Finding {
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
	return checkFile(cache.FileSet(), fc, f, NewAnalyzer().reporter)
}
