package packagevar

import (
	"fmt"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/singleuse"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

func buildCtx(t *testing.T, files, unchanged map[string]string) *pass.Context {
	t.Helper()

	cache := astcache.New()
	changed := make([]pass.FileChange, 0, len(files))
	mapFS := make(fstest.MapFS, len(files)+len(unchanged))
	for path, src := range unchanged {
		mapFS[path] = &fstest.MapFile{Data: []byte(src)}
	}
	for path, src := range files {
		mapFS[path] = &fstest.MapFile{Data: []byte(src)}
		if _, err := cache.Parse(path, []byte(src)); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		changed = append(changed, pass.FileChange{Path: path, Extension: ".go"})
	}
	return &pass.Context{
		ASTCache:     cache,
		Sources:      source.NewFSProvider(mapFS, "", source.SkipSet{}),
		ChangedFiles: changed,
	}
}

func located(findings []finding.Finding) []string {
	out := make([]string, 0, len(findings))
	for _, hit := range findings {
		out = append(out, fmt.Sprintf("%s:%d:%s", hit.File, hit.Line, hit.Message))
	}
	slices.Sort(out)
	return out
}

type levelVarCase struct {
	name  string
	files map[string]string
	extra map[string]string
	want  []string
}

func runCases(t *testing.T, testCases []levelVarCase) {
	t.Helper()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := buildCtx(t, testCase.files, testCase.extra)
			got := located(NewAnalyzer().Analyze(ctx))
			want := slices.Clone(testCase.want)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("findings = %q, want %q", got, want)
			}
		})
	}
}

func levelVarCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "mutex flagged",
			files: map[string]string{"a.go": "package p\n\nimport \"sync\"\n\n" +
				"var mu sync.Mutex\n"},
			want: []string{"a.go:5:package-level var mu"},
		},
		{
			name: "regexp flagged",
			files: map[string]string{
				"a.go": "package p\n\nimport \"regexp\"\n\nvar re = regexp.MustCompile(`a`)\n",
			},
			want: []string{"a.go:5:package-level var re"},
		},
		{
			name: "map flagged",
			files: map[string]string{"a.go": "package p\n\n" +
				"var table = map[string]int{\"a\": 1}\n"},
			want: []string{"a.go:3:package-level var table"},
		},
		{
			name:  "test file fixture flagged",
			files: map[string]string{"a_test.go": "package p\n\nvar fixture = []int{1}\n"},
			want:  []string{"a_test.go:3:package-level var fixture"},
		},
		{
			name: "one finding per name",
			files: map[string]string{
				"a.go": "package p\n\nvar (\n\tfirst, second int\n\tthird string\n)\n",
			},
			want: []string{
				"a.go:4:package-level var first",
				"a.go:4:package-level var second",
				"a.go:5:package-level var third",
			},
		},
		{
			name:  "const out of scope",
			files: map[string]string{"a.go": "package p\n\nconst limit = 3\n"},
		},
		{
			name: "local var out of scope",
			files: map[string]string{
				"a.go": "package p\n\nfunc f() { var local int; _ = local }\n",
			},
		},
	}
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	runCases(t, levelVarCases())
}

func nonPackageDirCases() []levelVarCase {
	const src = "package p\n\nvar mu int\n"
	return []levelVarCase{
		{name: "testdata dir skipped", files: map[string]string{"pkg/testdata/a.go": src}},
		{name: "top level testdata dir skipped", files: map[string]string{"testdata/a.go": src}},
		{name: "dot dir skipped", files: map[string]string{".hidden/a.go": src}},
		{name: "underscore dir skipped", files: map[string]string{"a/_scratch/a.go": src}},
		{
			name:  "dir merely containing testdata in its name checked",
			files: map[string]string{"mytestdata/a.go": src},
			want:  []string{"mytestdata/a.go:3:package-level var mu"},
		},
	}
}

func TestAnalyzeNonPackageDirs(t *testing.T) {
	t.Parallel()

	runCases(t, nonPackageDirCases())
}

func TestAnalyzeSkipsGenerated(t *testing.T) {
	t.Parallel()

	files := map[string]string{"a.go": "package p\n\nvar table = map[string]int{}\n"}
	ctx := buildCtx(t, files, nil)
	ctx.ChangedFiles[0].Generated = true
	if got := NewAnalyzer().Analyze(ctx); len(got) != 0 {
		t.Fatalf("findings = %v, want none in a generated file", got)
	}
}

func TestAnalyzeNilCache(t *testing.T) {
	t.Parallel()

	if got := NewAnalyzer().Analyze(&pass.Context{}); len(got) != 0 {
		t.Fatalf("findings = %v, want none", got)
	}
}

func TestDropSingleUseDuplicates(t *testing.T) {
	t.Parallel()

	single := func(file string, line int, symbol string) finding.Finding {
		return finding.Finding{AnalyzerID: singleuse.ID, File: file, Line: line, Symbol: symbol}
	}
	levelVar := func(file string, line int, symbol string) finding.Finding {
		return finding.Finding{AnalyzerID: ID, File: file, Line: line, Symbol: symbol}
	}
	testCases := []struct {
		name  string
		input []finding.Finding
		want  []finding.Finding
	}{
		{
			name:  "dropped at the single-use site",
			input: []finding.Finding{single("a.go", 3, "once"), levelVar("a.go", 3, "once")},
			want:  []finding.Finding{single("a.go", 3, "once")},
		},
		{
			name:  "order of arrival does not matter",
			input: []finding.Finding{levelVar("a.go", 3, "once"), single("a.go", 3, "once")},
			want:  []finding.Finding{single("a.go", 3, "once")},
		},
		{
			name:  "same line other name stays",
			input: []finding.Finding{single("a.go", 3, "a"), levelVar("a.go", 3, "b")},
			want:  []finding.Finding{single("a.go", 3, "a"), levelVar("a.go", 3, "b")},
		},
		{
			name:  "same name on another line stays",
			input: []finding.Finding{single("a.go", 5, "y"), levelVar("a.go", 4, "y")},
			want:  []finding.Finding{single("a.go", 5, "y"), levelVar("a.go", 4, "y")},
		},
		{
			name:  "single-use finding in another file does not drop it",
			input: []finding.Finding{single("b.go", 3, "once"), levelVar("a.go", 3, "once")},
			want:  []finding.Finding{single("b.go", 3, "once"), levelVar("a.go", 3, "once")},
		},
		{
			name: "another analyzer at the same site stays",
			input: []finding.Finding{
				single("a.go", 3, "once"),
				{AnalyzerID: "other", File: "a.go", Line: 3, Symbol: "once"},
			},
			want: []finding.Finding{
				single("a.go", 3, "once"),
				{AnalyzerID: "other", File: "a.go", Line: 3, Symbol: "once"},
			},
		},
		{name: "no findings"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := DropSingleUseDuplicates(slices.Clone(testCase.input))
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("findings = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func TestDropSingleUseDuplicatesOnRealAnalyzers(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"a.go": "package p\n\nvar once = 1\n\nfunc f() int { return once }\n",
	}
	ctx := buildCtx(t, files, nil)
	findings := singleuse.NewAnalyzer().Analyze(ctx)
	findings = append(findings, NewAnalyzer().Analyze(ctx)...)
	if len(findings) != 2 {
		t.Fatalf("both analyzers report the site, got %+v", findings)
	}

	got := DropSingleUseDuplicates(findings)
	if len(got) != 1 || got[0].AnalyzerID != singleuse.ID {
		t.Fatalf("after the transform = %+v, want the single-use finding alone", got)
	}
}
