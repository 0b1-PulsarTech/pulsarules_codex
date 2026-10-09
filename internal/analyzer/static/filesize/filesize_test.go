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

	analyzer := NewAnalyzer()

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
			change := core.FileChange{Path: "foo.go", IsTest: testCase.isTest}
			got := newLineLimits(testCase.maxLines, 0).forFile(change)
			if got != testCase.want {
				t.Errorf("limit for maxLines=%d isTest=%v = %d, want %d",
					testCase.maxLines, testCase.isTest, got, testCase.want)
			}
		})
	}
}

type allowanceCase struct {
	name   string
	change core.FileChange
	expect int
}

func runAllowanceCases(t *testing.T, files map[string]int, testCases []allowanceCase) {
	t.Helper()

	dir := t.TempDir()
	for name, lines := range files {
		body := "package foo\n" + strings.Repeat("// filler\n", lines-1)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), fsperm.File); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	analyzer := NewAnalyzer()
	srcs := sources.NewProvider(dir, sources.NewSkipSet(sources.DefaultSkipDirs()))
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := analyzer.Analyze(&core.AnalysisContext{
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

func TestAnalyze_TestFilesGetTheWiderAllowance(t *testing.T) {
	t.Parallel()

	runAllowanceCases(t, map[string]int{"big.go": 300, "big_test.go": 300, "huge_test.go": 500},
		[]allowanceCase{
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
		})
}

func TestAnalyze_MjsTestFlagDecidesTheCap(t *testing.T) {
	t.Parallel()

	runAllowanceCases(t, map[string]int{"x.mjs": 300}, []allowanceCase{
		{
			name:   "a 300-line mjs flagged as a test is under the test cap",
			change: core.FileChange{Path: "x.mjs", Extension: ".mjs", IsTest: true},
			expect: 0,
		},
		{
			name:   "the same mjs not flagged as a test breaks the 180 limit",
			change: core.FileChange{Path: "x.mjs", Extension: ".mjs"},
			expect: 1,
		},
	})
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

	analyzer := NewAnalyzer()
	got := analyzer.Analyze(&core.AnalysisContext{
		ProjectDir: dir,
		Sources:    sources.NewProvider(dir, sources.NewSkipSet(sources.DefaultSkipDirs())),
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

	analyzer := NewAnalyzer()
	got := analyzer.Analyze(&core.AnalysisContext{
		ProjectDir: dir,
		Sources:    sources.NewProvider(dir, sources.NewSkipSet(sources.DefaultSkipDirs())),
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

func analyzeHardCap(
	t *testing.T, name string, lines int, config *core.AnalysisConfig,
) []core.Finding {
	t.Helper()

	dir := t.TempDir()
	body := strings.Repeat("// filler\n", lines)
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), fsperm.File); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return NewAnalyzer().Analyze(&core.AnalysisContext{
		ProjectDir:   dir,
		Config:       config,
		Sources:      sources.NewProvider(dir, sources.NewSkipSet(sources.DefaultSkipDirs())),
		ChangedFiles: []core.FileChange{{Path: name, Extension: filepath.Ext(name)}},
	})
}

type hardCapCase struct {
	name     string
	file     string
	lines    int
	config   *core.AnalysisConfig
	severity core.Severity
	count    int
	message  string
	hint     string
}

func runHardCapCases(t *testing.T, testCases []hardCapCase) {
	t.Helper()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := analyzeHardCap(t, testCase.file, testCase.lines, testCase.config)
			if len(got) != testCase.count {
				t.Fatalf("got %d findings, want %d: %+v", len(got), testCase.count, got)
			}
			if testCase.count == 0 {
				return
			}
			if got[0].Severity != testCase.severity {
				t.Errorf("severity = %v, want %v: %+v", got[0].Severity, testCase.severity, got[0])
			}
			if testCase.message != "" && got[0].Message != testCase.message {
				t.Errorf("message = %q, want %q", got[0].Message, testCase.message)
			}
			if testCase.hint != "" && got[0].Suggestion != testCase.hint {
				t.Errorf("hint = %q, want %q", got[0].Suggestion, testCase.hint)
			}
		})
	}
}

func TestAnalyze_HardCap(t *testing.T) {
	t.Parallel()

	goCapped := paramsConfig(map[string]any{"max_lines": 100, "error_lines": 250})
	runHardCapCases(t, []hardCapCase{
		{
			name: "no cap without error_lines", file: "a.go", lines: 600,
			severity: core.SeverityWarning, count: 1,
		},
		{
			name: "the cap errors for go", file: "a.go", lines: 251, config: goCapped,
			severity: core.SeverityError, count: 1,
		},
		{
			name: "the cap keeps the warning below it", file: "a.go", lines: 250,
			config: goCapped, severity: core.SeverityWarning, count: 1,
		},
	})
}

func TestAnalyze_HardCapBelowTheSoftLimit(t *testing.T) {
	t.Parallel()

	lowCap := paramsConfig(map[string]any{"error_lines": 150})
	projectSoft := paramsConfig(map[string]any{"max_lines": 200, "error_lines": 190})
	runHardCapCases(t, []hardCapCase{
		{
			name: "cap under the inherited default max still errors", file: "a.svelte",
			lines: 170, config: lowCap, severity: core.SeverityError, count: 1,
			message: "file is 170 lines, hard cap 150",
		},
		{
			name: "cap under the project max still errors", file: "a.svelte", lines: 195,
			config: projectSoft, severity: core.SeverityError, count: 1,
			message: "file is 195 lines, hard cap 190",
		},
		{name: "a file under both caps is clean", file: "a.svelte", lines: 140, config: lowCap},
	})
}

func TestAnalyze_CapMessages(t *testing.T) {
	t.Parallel()

	runHardCapCases(t, []hardCapCase{
		{
			name: "hard cap names itself", file: "a.go", lines: 301,
			config:   paramsConfig(map[string]any{"error_lines": 300}),
			severity: core.SeverityError, count: 1,
			message: "file is 301 lines, hard cap 300",
			hint:    "split into smaller files (hard cap 300 lines per file)",
		},
		{
			name: "soft limit names itself", file: "a.go", lines: 200,
			severity: core.SeverityWarning, count: 1,
			message: "file is 200 lines, max 180",
			hint:    "split into smaller files (max 180 lines per file)",
		},
	})
}

func paramsConfig(params map[string]any) *core.AnalysisConfig {
	return &core.AnalysisConfig{Analyzers: map[string]core.AnalyzerConfig{
		"file-size": {Enabled: true, Params: params},
	}}
}
