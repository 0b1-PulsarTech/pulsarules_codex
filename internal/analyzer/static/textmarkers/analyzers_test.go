package textmarkers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type markerCase struct {
	name     string
	rel      string
	body     string
	wantText int
	wantTypo int
}

var markerCases = []markerCase{
	{"zero width space in go", "a.go", "package a // x\u200By\n", 1, 0},
	{"em dash in go", "a.go", "package a // x\u2014y\n", 0, 1},
	{"em dash in markdown", "a.md", "# x\u2014y\n", 0, 1},
	{"ellipsis in markdown", "a.md", "wait\u2026\n", 0, 1},
	{"nbsp in markdown", "a.md", "a\u00A0b\n", 1, 0},
	{"both classes at once", "a.md", "a\u200Bb\u2014c\n", 1, 1},
	{"clean file", "a.go", "package a\n", 0, 0},
	{"unlisted extension", "a.txt", "x\u200By\n", 0, 0},
	{"fixture under testdata", "testdata/a.md", "x\u200By\u2014z\n", 0, 0},
	{"ai frontmatter key", "a.md", "---\ngenerator: Claude\n---\n\nbody\n", 1, 0},
	{"nested key is not provenance", "a.md", "---\nauthor:\n  claude: no\n---\n\nbody\n", 0, 0},
	{
		"prose naming a vendor is not provenance",
		"a.md",
		"We asked Claude about openai today.\n",
		0,
		0,
	},
	{"frontmatter in go is ignored", "a.go", "---\ngenerator: Claude\n---\n", 0, 0},
}

func TestAnalyzers(t *testing.T) {
	t.Parallel()

	for _, testCase := range markerCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, testCase.rel)
			if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if err := os.WriteFile(path, []byte(testCase.body), 0o600); err != nil {
				t.Fatalf("seed: %v", err)
			}
			ctx := &core.AnalysisContext{
				ProjectDir: dir,
				Sources:    core.NewSourceProvider(dir),
				ChangedFiles: []core.FileChange{
					{Path: testCase.rel, Extension: filepath.Ext(testCase.rel)},
				},
			}

			if got := NewTextAnalyzer().Analyze(ctx); len(got) != testCase.wantText {
				t.Errorf("text-markers = %d, want %d: %+v", len(got), testCase.wantText, got)
			}
			if got := NewTypographicAnalyzer().Analyze(ctx); len(got) != testCase.wantTypo {
				t.Errorf("typographic-markers = %d, want %d: %+v", len(got), testCase.wantTypo, got)
			}
		})
	}
}

// TestAnalyze_NoSources pins the guard: a context assembled without a source
// provider degrades to silence instead of panicking mid-pipeline.
func TestAnalyze_NoSources(t *testing.T) {
	t.Parallel()

	if got := NewTextAnalyzer().Analyze(&core.AnalysisContext{}); len(got) != 0 {
		t.Errorf("text-markers = %+v, want none", got)
	}
	if got := NewTypographicAnalyzer().Analyze(&core.AnalysisContext{}); len(got) != 0 {
		t.Errorf("typographic-markers = %+v, want none", got)
	}
}

// TestSeverities pins the split the whole design rests on: typographic blocks,
// carriers only warn.
func TestSeverities(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "a.md"),
		[]byte("a\u200Bb\u2014c\n"),
		0o600,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}
	ctx := &core.AnalysisContext{
		ProjectDir:   dir,
		Sources:      core.NewSourceProvider(dir),
		ChangedFiles: []core.FileChange{{Path: "a.md", Extension: ".md"}},
	}
	if got := NewTextAnalyzer().Analyze(ctx); got[0].Severity != core.SeverityWarning {
		t.Errorf("carrier severity = %v, want warning", got[0].Severity)
	}
	if got := NewTypographicAnalyzer().Analyze(ctx); got[0].Severity != core.SeverityError {
		t.Errorf("typographic severity = %v, want error", got[0].Severity)
	}
}
