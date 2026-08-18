package filesize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
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
			name:   "no changed files",
			ctx:    &core.AnalysisContext{ProjectDir: ".", ChangedFiles: nil},
			expect: 0,
		},
		{
			name: "non-go file skipped",
			ctx: &core.AnalysisContext{
				ProjectDir:   ".",
				ChangedFiles: []core.FileChange{{Path: "foo.md", Extension: ".md"}},
			},
			expect: 0,
		},
		{
			name: "nonexistent file skipped",
			ctx: &core.AnalysisContext{
				ProjectDir:   ".",
				ChangedFiles: []core.FileChange{{Path: "nonexistent.go", Extension: ".go"}},
			},
			expect: 0,
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

func TestLineLimits(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		maxLines int
		isTest   bool
		want     int
	}{
		{name: "production keeps the configured max", maxLines: 180, isTest: false, want: 180},
		{name: "test file gets 2.6x the max", maxLines: 180, isTest: true, want: 468},
		{name: "allowance follows a lowered max", maxLines: 100, isTest: true, want: 260},
		{name: "allowance rounds instead of truncating", maxLines: 75, isTest: true, want: 195},
		{name: "a disabling zero stays zero", maxLines: 0, isTest: true, want: 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := newLineLimits(testCase.maxLines).
				forFile(core.FileChange{IsTest: testCase.isTest})
			if got != testCase.want {
				t.Errorf("limit for maxLines=%d isTest=%v = %d, want %d",
					testCase.maxLines, testCase.isTest, got, testCase.want)
			}
		})
	}
}

// TestAnalyze_TestFilesGetTheWiderAllowance pins the pair that matters: the
// same line count that fails production code passes in a _test.go, and the
// test file still fails once it crosses its own larger limit.
func TestAnalyze_TestFilesGetTheWiderAllowance(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write := func(name string, lines int) {
		t.Helper()
		body := "package foo\n" + strings.Repeat("// filler\n", lines-1)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), fsperm.File); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("big.go", 300)
	write("big_test.go", 300)
	write("huge_test.go", 500)

	a := NewAnalyzer()
	srcs := core.NewSourceProvider(dir)

	testCases := []struct {
		name   string
		change core.FileChange
		expect int
	}{
		{
			name:   "300-line production file breaks the 180 limit",
			change: core.FileChange{Path: "big.go", Extension: ".go"},
			expect: 1,
		},
		{
			name:   "the same 300 lines are fine in a test file",
			change: core.FileChange{Path: "big_test.go", Extension: ".go", IsTest: true},
			expect: 0,
		},
		{
			name:   "a test file still fails past 468 lines",
			change: core.FileChange{Path: "huge_test.go", Extension: ".go", IsTest: true},
			expect: 1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := a.Analyze(&core.AnalysisContext{
				ProjectDir:   dir,
				Sources:      srcs,
				ChangedFiles: []core.FileChange{testCase.change},
			})
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %+v", len(got), testCase.expect, got)
			}
		})
	}
}

func TestCountLines(t *testing.T) {
	t.Parallel()

	n := countLines([]byte("line1\nline2\nline3\n"))
	if n != 3 {
		t.Fatalf("expected 3 lines, got %d", n)
	}
	n = countLines([]byte(""))
	if n != 0 {
		t.Fatalf("expected 0 for empty content, got %d", n)
	}
}
