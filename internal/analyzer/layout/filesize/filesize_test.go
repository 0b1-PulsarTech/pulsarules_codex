package filesize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()

	testCases := []struct {
		name   string
		ctx    *pass.Context
		expect int
	}{
		{
			name:   "no changed files",
			ctx:    &pass.Context{ProjectDir: ".", ChangedFiles: nil},
			expect: 0,
		},
		{
			name: "non-go file skipped",
			ctx: &pass.Context{
				ProjectDir:   ".",
				ChangedFiles: []pass.FileChange{{Path: "foo.md", Extension: ".md"}},
			},
			expect: 0,
		},
		{
			name: "nonexistent file skipped",
			ctx: &pass.Context{
				ProjectDir:   ".",
				ChangedFiles: []pass.FileChange{{Path: "nonexistent.go", Extension: ".go"}},
			},
			expect: 0,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := analyzer.Analyze(testCase.ctx)
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
		{name: "production keeps the configured max", maxLines: 180, want: 180},
		{name: "test file gets 2.6x the max", maxLines: 180, isTest: true, want: 468},
		{name: "allowance follows a lowered max", maxLines: 100, isTest: true, want: 260},
		{name: "allowance rounds instead of truncating", maxLines: 75, isTest: true, want: 195},
		{name: "a raw zero limit stays zero", maxLines: 0, isTest: true, want: 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			change := pass.FileChange{Path: "foo.go", IsTest: testCase.isTest}
			got := newLineLimits(testCase.maxLines).forFile(change)
			if got != testCase.want {
				t.Errorf("limit for maxLines=%d isTest=%v = %d, want %d",
					testCase.maxLines, testCase.isTest, got, testCase.want)
			}
		})
	}
}

type allowanceCase struct {
	name   string
	change pass.FileChange
	expect int
}

func runAllowanceCases(t *testing.T, files map[string]int, testCases []allowanceCase) {
	t.Helper()

	dir := t.TempDir()
	for name, lines := range files {
		body := "package foo\n" + strings.Repeat("// filler\n", lines-1)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), disk.File); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	analyzer := NewAnalyzer()
	srcs := source.NewProvider(dir, source.NewSkipSet(source.DefaultSkipDirs()))
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := analyzer.Analyze(&pass.Context{
				ProjectDir:   dir,
				Sources:      srcs,
				ChangedFiles: []pass.FileChange{testCase.change},
			})
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %+v", len(got), testCase.expect, got)
			}
		})
	}
}

func TestAnalyze_TestFilesGetTheWiderAllowance(t *testing.T) {
	t.Parallel()

	runAllowanceCases(t, map[string]int{"big.go": 300, "big_test.go": 300, "huge_test.go": 500},
		[]allowanceCase{
			{
				name:   "300-line production file breaks the 180 limit",
				change: pass.FileChange{Path: "big.go", Extension: ".go"},
				expect: 1,
			},
			{
				name:   "the same 300 lines are fine in a test file",
				change: pass.FileChange{Path: "big_test.go", Extension: ".go", IsTest: true},
				expect: 0,
			},
			{
				name:   "a test file still fails past 468 lines",
				change: pass.FileChange{Path: "huge_test.go", Extension: ".go", IsTest: true},
				expect: 1,
			},
		})
}

func TestAnalyze_MjsTestFlagDecidesTheCap(t *testing.T) {
	t.Parallel()

	runAllowanceCases(t, map[string]int{"x.mjs": 300}, []allowanceCase{
		{
			name:   "a 300-line mjs flagged as a test is under the test cap",
			change: pass.FileChange{Path: "x.mjs", Extension: ".mjs", IsTest: true},
			expect: 0,
		},
		{
			name:   "the same mjs not flagged as a test breaks the 180 limit",
			change: pass.FileChange{Path: "x.mjs", Extension: ".mjs"},
			expect: 1,
		},
	})
}

func TestAnalyze_SkipsTestdata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fixtureDir := filepath.Join(dir, "testdata")
	if err := os.MkdirAll(fixtureDir, disk.DirPrivate); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	body := "package fixture\n" + strings.Repeat("// filler\n", 300)
	fixturePath := filepath.Join(fixtureDir, "big.go")
	if err := os.WriteFile(fixturePath, []byte(body), disk.File); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	analyzer := NewAnalyzer()
	got := analyzer.Analyze(&pass.Context{
		ProjectDir: dir,
		Sources:    source.NewProvider(dir, source.NewSkipSet(source.DefaultSkipDirs())),
		ChangedFiles: []pass.FileChange{
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
	if err := os.WriteFile(filepath.Join(dir, "big.dart"), []byte(body), disk.File); err != nil {
		t.Fatalf("write big.dart: %v", err)
	}

	analyzer := NewAnalyzer()
	got := analyzer.Analyze(&pass.Context{
		ProjectDir: dir,
		Sources:    source.NewProvider(dir, source.NewSkipSet(source.DefaultSkipDirs())),
		ChangedFiles: []pass.FileChange{
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
