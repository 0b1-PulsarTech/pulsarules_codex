package character

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
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
		{"zero width space in dart, a default source extension", "a.dart", "// x\u200By\n", 1, 0},
		{"em dash in dart, a default source extension", "a.dart", "// x\u2014y\n", 0, 1},
		{"em dash in markdown", "a.md", "# x\u2014y\n", 0, 1},
		{"ellipsis in markdown", "a.md", "wait\u2026\n", 0, 1},
		{"nbsp in markdown", "a.md", "a\u00A0b\n", 1, 0},
		{"both classes at once", "a.md", "a\u200Bb\u2014c\n", 1, 1},
		{"clean file", "a.go", "package a\n", 0, 0},
		{"unlisted extension", "a.txt", "x\u200By\n", 0, 0},
		{"fixture under testdata", "testdata/a.md", "x\u200By\u2014z\n", 0, 0},
		{"ai frontmatter key", "a.md", "---\ngenerator: Claude\n---\n\nbody\n", 1, 0},
		{
			"nested key is not provenance",
			"a.md",
			"---\nauthor:\n  claude: no\n---\n\nbody\n",
			0,
			0,
		},
		{
			"prose naming a vendor is not provenance",
			"a.md",
			"We asked Claude about openai today.\n",
			0,
			0,
		},
		{"frontmatter in go is ignored", "a.go", "---\ngenerator: Claude\n---\n", 0, 0},
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
			ctx := &pass.Context{
				ProjectDir: dir,
				Sources:    pass.NewSourceProvider(dir, pass.NewSkipSet(pass.DefaultSkipDirs())),
				ChangedFiles: []pass.FileChange{
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

	if got := NewTextAnalyzer().Analyze(&pass.Context{}); len(got) != 0 {
		t.Errorf("text-markers = %+v, want none", got)
	}
	if got := NewTypographicAnalyzer().Analyze(&pass.Context{}); len(got) != 0 {
		t.Errorf("typographic-markers = %+v, want none", got)
	}
}

func TestCarrierSeverityByClass(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
		want pass.Severity
	}{
		{name: "zero width space blocks", body: "a\u200Bb\n", want: pass.SeverityError},
		{name: "bidi override blocks", body: "a\u202Eb\n", want: pass.SeverityError},
		{name: "no-break space blocks", body: "a\u00A0b\n", want: pass.SeverityError},
		{
			name: "zero width joiner between ascii blocks (no emoji to glue, so it's " +
				"a plain carrier)",
			body: "a\u200Db\n",
			want: pass.SeverityError,
		},
		{
			name: "zero width joiner between emoji advises",
			body: "\U0001F468\u200D\U0001F469\n",
			want: pass.SeverityWarning,
		},
		{name: "byte order mark advises", body: "a\uFEFFb\n", want: pass.SeverityWarning},
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
		want     pass.Severity
	}{
		{name: "default blocks", want: pass.SeverityError},
		{
			name:     "warning keeps the report without the gate",
			severity: "warning",
			want:     pass.SeverityWarning,
		},
		{name: "info reports quietly", severity: "info", want: pass.SeverityInfo},
		{
			name:     "unrecognized value keeps the blocking default",
			severity: "fatal",
			want:     pass.SeverityError,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := markdownContext(t, "a\u2014b\n")
			if testCase.severity != "" {
				ctx.Config = &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{
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

func markdownContext(t *testing.T, body string) *pass.Context {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(body), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return &pass.Context{
		ProjectDir:   dir,
		Sources:      pass.NewSourceProvider(dir, pass.NewSkipSet(pass.DefaultSkipDirs())),
		ChangedFiles: []pass.FileChange{{Path: "a.md", Extension: ".md"}},
	}
}

func TestTypographic_MarkdownCodeSkipped(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ext  string
		body string
		want int
	}{
		{name: "markdown prose is reported", ext: ".md", body: "a \u2014 b\n", want: 1},
		{name: "markdown fence is skipped", ext: ".md", body: "```\na \u2014 b\n```\n"},
		{name: "markdown inline span is skipped", ext: ".md", body: "see `a \u2014 b`\n"},
		{
			name: "prose beside a fence is still reported",
			ext:  ".md",
			body: "a \u2014 b\n```\nc \u2014 d\n```\n",
			want: 1,
		},
		{
			name: "go source keeps reporting",
			ext:  ".go",
			body: "package x\n\nvar s = \"a \u2014 b\"\n",
			want: 1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			name := "a" + testCase.ext
			path := filepath.Join(dir, name)
			if err := os.WriteFile(path, []byte(testCase.body), 0o600); err != nil {
				t.Fatalf("seed: %v", err)
			}
			ctx := &pass.Context{
				ProjectDir:   dir,
				Sources:      pass.NewSourceProvider(dir, pass.NewSkipSet(pass.DefaultSkipDirs())),
				ChangedFiles: []pass.FileChange{{Path: name, Extension: testCase.ext}},
			}
			if got := NewTypographicAnalyzer().Analyze(ctx); len(got) != testCase.want {
				t.Errorf("findings = %+v, want %d", got, testCase.want)
			}
		})
	}
}
