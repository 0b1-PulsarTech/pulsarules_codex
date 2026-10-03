package filesize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/sources"
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
		path     string
		want     int
	}{
		{name: "production keeps the configured max", maxLines: 180, path: "foo.go", want: 180},
		{name: "test file gets 2.6x the max", maxLines: 180, path: "foo_test.go", want: 468},
		{name: "allowance follows a lowered max", maxLines: 100, path: "foo_test.go", want: 260},
		{
			name:     "allowance rounds instead of truncating",
			maxLines: 75,
			path:     "foo_test.go",
			want:     195,
		},
		{name: "a raw zero limit stays zero", maxLines: 0, path: "foo_test.go", want: 0},
		{
			name:     "a non-go test path gets the allowance too",
			maxLines: 180,
			path:     "foo.spec.ts",
			want:     468,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := newLineLimits(testCase.maxLines).forFile(testCase.path)
			if got != testCase.want {
				t.Errorf("limit for maxLines=%d path=%q = %d, want %d",
					testCase.maxLines, testCase.path, got, testCase.want)
			}
		})
	}
}

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
	srcs := sources.NewSourceProvider(dir, sources.NewSkipSet(sources.DefaultSkipDirs()))

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

func TestAnalyze_SkipsTestdata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fixtureDir := filepath.Join(dir, "testdata")
	if err := os.MkdirAll(fixtureDir, fsperm.DirPrivate); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	body := "package fixture\n" + strings.Repeat("// filler\n", 300)
	fixturePath := filepath.Join(fixtureDir, "big.go")
	if err := os.WriteFile(fixturePath, []byte(body), fsperm.File); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	a := NewAnalyzer()
	got := a.Analyze(&core.AnalysisContext{
		ProjectDir: dir,
		Sources:    sources.NewSourceProvider(dir, sources.NewSkipSet(sources.DefaultSkipDirs())),
		ChangedFiles: []core.FileChange{
			{Path: "testdata/big.go", Extension: ".go"},
		},
	})
	if len(got) != 0 {
		t.Fatalf("got %d findings for a testdata fixture, want 0: %+v", len(got), got)
	}
}

func TestAnalyze_ReportsOnADartFileJustLikeGo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	body := "class Foo {\n" + strings.Repeat("  // filler\n", 300) + "}\n"
	if err := os.WriteFile(filepath.Join(dir, "big.dart"), []byte(body), fsperm.File); err != nil {
		t.Fatalf("write big.dart: %v", err)
	}

	a := NewAnalyzer()
	got := a.Analyze(&core.AnalysisContext{
		ProjectDir: dir,
		Sources:    sources.NewSourceProvider(dir, sources.NewSkipSet(sources.DefaultSkipDirs())),
		ChangedFiles: []core.FileChange{
			{Path: "big.dart", Extension: ".dart"},
		},
	})
	if len(got) != 1 {
		t.Fatalf("got %d findings for an oversized .dart file, want 1: %+v", len(got), got)
	}
}

func TestCountLines(t *testing.T) {
	t.Parallel()

	count := countLines([]byte("line1\nline2\nline3\n"))
	if count != 3 {
		t.Fatalf("expected 3 lines, got %d", count)
	}
	count = countLines([]byte(""))
	if count != 0 {
		t.Fatalf("expected 0 for empty content, got %d", count)
	}
}
