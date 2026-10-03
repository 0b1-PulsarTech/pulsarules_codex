package packagelevelvar

import (
	"fmt"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/sources"
)

func buildCtx(
	t *testing.T, files, unchanged map[string]string, prior []core.Finding,
) *core.AnalysisContext {
	t.Helper()

	cache := astcache.New()
	changed := make([]core.FileChange, 0, len(files))
	mapFS := make(fstest.MapFS, len(files)+len(unchanged))
	for path, src := range unchanged {
		mapFS[path] = &fstest.MapFile{Data: []byte(src)}
	}
	for path, src := range files {
		mapFS[path] = &fstest.MapFile{Data: []byte(src)}
		if _, err := cache.Parse(path, []byte(src)); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		changed = append(changed, core.FileChange{Path: path, Extension: ".go"})
	}
	return &core.AnalysisContext{
		ASTCache:     cache,
		Sources:      sources.NewFSProvider(mapFS, "", sources.SkipSet{}),
		ChangedFiles: changed,
		Findings:     prior,
	}
}

func located(findings []core.Finding) []string {
	out := make([]string, 0, len(findings))
	for _, finding := range findings {
		out = append(out, fmt.Sprintf("%s:%d:%s", finding.File, finding.Line, finding.Message))
	}
	slices.Sort(out)
	return out
}

type levelVarCase struct {
	name  string
	files map[string]string
	extra map[string]string
	prior []core.Finding
	want  []string
}

func runCases(t *testing.T, testCases []levelVarCase) {
	t.Helper()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := buildCtx(t, testCase.files, testCase.extra, testCase.prior)
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
	ctx := buildCtx(t, files, nil, nil)
	ctx.ChangedFiles[0].Generated = true
	if got := NewAnalyzer().Analyze(ctx); len(got) != 0 {
		t.Fatalf("findings = %v, want none in a generated file", got)
	}
}

func TestAnalyzeNilCache(t *testing.T) {
	t.Parallel()

	if got := NewAnalyzer().Analyze(&core.AnalysisContext{}); len(got) != 0 {
		t.Fatalf("findings = %v, want none", got)
	}
}

func TestIdentity(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	if analyzer.ID() != "package-level-var" || analyzer.Stage() != core.StageAST {
		t.Fatalf("id=%q stage=%v", analyzer.ID(), analyzer.Stage())
	}
}
