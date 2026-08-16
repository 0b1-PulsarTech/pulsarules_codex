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

func TestAnalyzers(t *testing.T) {
	t.Parallel()

	testCases := []markerCase{
		{"zero width space in go", "a.go", "package a // x\u200By\n", 1, 0},
		{"em dash in go", "a.go", "package a // x\u2014y\n", 0, 1},
		{"em dash in markdown", "a.md", "# x\u2014y\n", 0, 1},
		{"ellipsis in markdown", "a.md", "wait\u2026\n", 0, 1},
		{"nbsp in markdown", "a.md", "a\u00A0b\n", 1, 0},
		{"both classes at once", "a.md", "a\u200Bb\u2014c\n", 1, 1},
		{"clean file", "a.go", "package a\n", 0, 0},
		{"unlisted extension", "a.txt", "x\u200By\n", 0, 0},
		{"fixture under testdata", "testdata/a.md", "x\u200By\u2014z\n", 0, 0},
	}

	for _, testCase := range testCases {
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
				t.Errorf(
					"typographic-markers = %d, want %d: %+v", len(got), testCase.wantTypo, got,
				)
			}
		})
	}
}

func TestAnalyze_NoSources(t *testing.T) {
	t.Parallel()

	if got := NewTextAnalyzer().Analyze(&core.AnalysisContext{}); len(got) != 0 {
		t.Errorf("text-markers = %+v, want none", got)
	}
	if got := NewTypographicAnalyzer().Analyze(&core.AnalysisContext{}); len(got) != 0 {
		t.Errorf("typographic-markers = %+v, want none", got)
	}
}

func TestCarrierSeverityByClass(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
		want core.Severity
	}{
		{name: "zero width space blocks", body: "a\u200Bb\n", want: core.SeverityError},
		{name: "bidi override blocks", body: "a\u202Eb\n", want: core.SeverityError},
		{name: "no-break space blocks", body: "a\u00A0b\n", want: core.SeverityError},
		{
			name: "zero width joiner between ascii blocks",
			body: "a\u200Db\n",
			want: core.SeverityError,
		},
		{
			name: "zero width joiner between emoji advises",
			body: "\U0001F468\u200D\U0001F469\n",
			want: core.SeverityWarning,
		},
		{name: "byte order mark advises", body: "a\uFEFFb\n", want: core.SeverityWarning},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := NewTextAnalyzer().Analyze(markdownContext(t, testCase.body))
			if len(got) != 1 {
				t.Fatalf("findings = %+v, want exactly one", got)
			}
			if got[0].Severity != testCase.want {
				t.Errorf("severity = %v, want %v", got[0].Severity, testCase.want)
			}
		})
	}
}

func TestTypographicSeverityIsConfigurable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		severity string
		want     core.Severity
	}{
		{name: "default blocks", want: core.SeverityError},
		{
			name:     "warning keeps the report without the gate",
			severity: "warning",
			want:     core.SeverityWarning,
		},
		{name: "info reports quietly", severity: "info", want: core.SeverityInfo},
		{
			name:     "unrecognized value keeps the blocking default",
			severity: "fatal",
			want:     core.SeverityError,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := markdownContext(t, "a\u2014b\n")
			if testCase.severity != "" {
				ctx.Config = &core.AnalysisConfig{Analyzers: map[string]core.AnalyzerConfig{
					"typographic-markers": {
						Enabled: true,
						Params:  map[string]any{"severity": testCase.severity},
					},
				}}
			}
			got := NewTypographicAnalyzer().Analyze(ctx)
			if len(got) != 1 {
				t.Fatalf("findings = %+v, want exactly one", got)
			}
			if got[0].Severity != testCase.want {
				t.Errorf("severity = %v, want %v", got[0].Severity, testCase.want)
			}
		})
	}
}

func markdownContext(t *testing.T, body string) *core.AnalysisContext {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(body), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return &core.AnalysisContext{
		ProjectDir:   dir,
		Sources:      core.NewSourceProvider(dir),
		ChangedFiles: []core.FileChange{{Path: "a.md", Extension: ".md"}},
	}
}
