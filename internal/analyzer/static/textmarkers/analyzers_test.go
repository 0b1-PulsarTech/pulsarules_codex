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

// TestCarrierSeverityByClass pins the split the design rests on: a carrier no
// context can justify blocks, while one that MAY be load-bearing only advises.
// Every marker here is written as a Go escape so the fixture does not trip the
// very check it exercises.
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
		// A joiner flanked by ASCII cannot be gluing an emoji, so mark demotes
		// it to a plain carrier - which must block like any other.
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

// TestTypographicSeverityIsConfigurable asserts the default blocks and that a
// project can keep the report while dropping the gate, without the fallback
// ever collapsing to Info (Severity's zero value) on an unknown value.
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

// markdownContext writes body to a markdown file in a fresh temp dir and
// returns the context the analyzers read it through.
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

// TestTypographicSkipsMarkdownCode pins the exception through the analyzer: a
// character inside a markdown fence is content being shown, while the same
// character in prose - or anywhere in a .go file, where nothing can tell a
// string literal from prose - is still reported.
func TestTypographicSkipsMarkdownCode(t *testing.T) {
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
			// A Go string literal is indistinguishable from prose, so the
			// exception deliberately stops at markdown.
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
			ctx := &core.AnalysisContext{
				ProjectDir:   dir,
				Sources:      core.NewSourceProvider(dir),
				ChangedFiles: []core.FileChange{{Path: name, Extension: testCase.ext}},
			}
			if got := NewTypographicAnalyzer().Analyze(ctx); len(got) != testCase.want {
				t.Errorf("findings = %+v, want %d", got, testCase.want)
			}
		})
	}
}
