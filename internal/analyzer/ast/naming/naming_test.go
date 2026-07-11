package naming

import (
	"go/token"
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

func TestCheckFile(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()

	testCases := []struct {
		name   string
		source string
		expect int
	}{
		{
			name:   "clean file",
			source: "package foo\nvar UserName string\n",
			expect: 0,
		},
		{
			name:   "numbered name",
			source: "package foo\nvar user1 string\n",
			expect: 1,
		},
		{
			name:   "hungarian notation",
			source: "package foo\nvar strName string\n",
			expect: 1,
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
			name:   "func param numbered",
			source: "package foo\nfunc f(x2 int) {}\n",
			expect: 1,
		},
		{
			name:   "short var in assign",
			source: "package foo\nfunc f() { x1 := 1 }\n",
			expect: 1,
		},
		{
			name:   "underscore skipped",
			source: "package foo\nfunc f() { _ = 1 }\n",
			expect: 0,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tmp := t.TempDir()
			path := filepath.Join(tmp, "foo.go")
			if err := os.WriteFile(path, []byte(testCase.source), 0o644); err != nil {
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
			got := a.checkFile(cache.FileSet(), fc, f)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d", len(got), testCase.expect)
			}
		})
	}
}

// TestCheckNumbered pins checkNumbered's rule directly: a trailing digit run
// preceded by an uppercase letter is an acronym/version marker, not a
// sequential-counter smell, even though checkName's Hungarian-notation check
// (a separate, pre-existing rule) would otherwise also fire on a name like
// "UTF8" and confuse a checkName-level assertion.
func TestCheckNumbered(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		ident string
		want  bool
	}{
		{"sequential suffix", "foo1", true},
		{"sequential suffix double digit", "bar22", true},
		{"no digit", "fooBar", false},
		{"markdown heading level", "H1", false},
		{"acronym with trailing digit", "UTF8", false},
		{"version marker", "V2", false},
		{"digit after uppercase mid-word", "bodyWithoutH1", false},
		{"empty", "", false},
		{"all digits", "123", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := checkNumbered(testCase.ident); got != testCase.want {
				t.Fatalf("checkNumbered(%q) = %v, want %v", testCase.ident, got, testCase.want)
			}
		})
	}
}

func TestCheckName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		ident    string
		hasIssue bool
	}{
		{"underscore", "_", false},
		{"empty", "", false},
		{"valid camel", "userName", false},
		{"valid exported", "UserName", false},
		{"single exported", "X", true},
		{"numbered suffix", "foo1", true},
		{"numbered suffix 2", "bar22", true},
		{"no digit", "fooBar", false},
		{"digit after uppercase letter not flagged", "bodyWithoutH1", false},
		{"markdown heading level not flagged", "H1", false},
		{"hungarian str", "strName", true},
		{"hungarian sz", "szBuffer", true},
		{"hungarian n", "nCount", true},
		{"hungarian b", "bFlag", true},
		{"noise word exact", "data", true},
		{"noise word prefix", "dataValue", false},
		{"noise word prefix case", "DataFile", false},
		{"noise word not prefix", "tempFile", false},
		{"noise word not prefix 2", "tmpDir", false},
		{"noise word helper", "helper", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var got int
			checkName(testCase.ident, token.NoPos, func(_ token.Pos, _, _ string) {
				got++
			})
			hasIssue := got > 0
			if hasIssue != testCase.hasIssue {
				t.Fatalf(
					"checkName(%q) hasIssue=%v, want %v",
					testCase.ident,
					hasIssue,
					testCase.hasIssue,
				)
			}
		})
	}
}
