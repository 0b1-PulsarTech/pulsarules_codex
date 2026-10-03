package singleuse

import (
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

func buildCtx(t *testing.T, files map[string]string, changed []string) *pass.Context {
	t.Helper()

	mapFS := make(fstest.MapFS, len(files))
	for path, src := range files {
		mapFS[path] = &fstest.MapFile{Data: []byte(src)}
	}

	cache := astcache.New()
	changedFiles := make([]pass.FileChange, 0, len(changed))
	for _, path := range changed {
		src, ok := files[path]
		if !ok {
			t.Fatalf("changed path %q has no source", path)
		}
		if _, err := cache.Parse(path, []byte(src)); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		changedFiles = append(changedFiles, pass.FileChange{Path: path, Extension: ".go"})
	}

	return &pass.Context{
		ASTCache:     cache,
		Sources:      source.NewFSSourceProvider(mapFS, "", source.SkipSet{}),
		ChangedFiles: changedFiles,
	}
}

type singleUseCase struct {
	name    string
	files   map[string]string
	changed []string
	expect  int
}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	for _, testCase := range singleUseCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := buildCtx(t, testCase.files, testCase.changed)
			got := NewAnalyzer().Analyze(ctx)
			if len(got) != testCase.expect {
				t.Fatalf("findings = %d, want %d: %+v", len(got), testCase.expect, got)
			}
		})
	}
}

func singleUseCases() []singleUseCase {
	return append(
		append(singleUseImmovableCases(), singleUseBasicCases()...),
		singleUseCrossFileCases()...)
}

func singleUseImmovableCases() []singleUseCase {
	return []singleUseCase{
		{
			name: "go:embed var cannot move, the directive is package-level only",
			files: map[string]string{
				"p.go": "package p\n\nimport \"embed\"\n\n" +
					"//go:embed assets\nvar assets embed.FS\n\n" +
					"func Load() embed.FS {\n\treturn assets\n}\n",
			},
			changed: []string{"p.go"},
			expect:  0,
		},
		{
			name: "atomic counter must keep its identity across calls",
			files: map[string]string{
				"p.go": "package p\n\nimport \"sync/atomic\"\n\n" +
					"var counter atomic.Int64\n\n" +
					"func Next() int64 {\n\treturn counter.Add(1)\n}\n",
			},
			changed: []string{"p.go"},
			expect:  0,
		},
		{
			name: "sync primitive must keep its identity across calls",
			files: map[string]string{
				"p.go": "package p\n\nimport \"sync\"\n\n" +
					"var once sync.Once\n\n" +
					"func Init() {\n\tonce.Do(func() {})\n}\n",
			},
			changed: []string{"p.go"},
			expect:  0,
		},
	}
}

func singleUseBasicCases() []singleUseCase {
	return []singleUseCase{
		{
			name: "single-use const flagged",
			files: map[string]string{
				"p.go": "package p\n\nconst magicNumber = 42\n\n" +
					"func UseIt() int {\n\treturn magicNumber * 2\n}\n",
			},
			changed: []string{"p.go"},
			expect:  1,
		},
		{
			name: "const used by two functions not flagged",
			files: map[string]string{
				"p.go": "package p\n\nconst shared = 42\n\n" +
					"func A() int { return shared }\n\nfunc B() int { return shared }\n",
			},
			changed: []string{"p.go"},
			expect:  0,
		},
		{
			name: "exported const not flagged",
			files: map[string]string{
				"p.go": "package p\n\nconst Exported = 42\n\n" +
					"func UseIt() int { return Exported }\n",
			},
			changed: []string{"p.go"},
			expect:  0,
		},
		{
			name: "var initialised by a call is exempt even when used once",
			files: map[string]string{
				"p.go": "package p\n\nimport \"regexp\"\n\n" +
					"var pattern = regexp.MustCompile(`^a+$`)\n\n" +
					"func Match(s string) bool { return pattern.MatchString(s) }\n",
			},
			changed: []string{"p.go"},
			expect:  0,
		},
		{
			name: "var initialised by a composite literal is exempt even when used once",
			files: map[string]string{
				"p.go": "package p\n\nvar names = []string{\"a\", \"b\"}\n\n" +
					"func First() string { return names[0] }\n",
			},
			changed: []string{"p.go"},
			expect:  0,
		},
		{
			name: "var _ interface assertion is never a candidate",
			files: map[string]string{
				"p.go": "package p\n\ntype Iface interface{ M() }\n\ntype impl struct{}\n\n" +
					"func (impl) M() {}\n\nvar _ Iface = (*impl)(nil)\n",
			},
			changed: []string{"p.go"},
			expect:  0,
		},
		{
			name: "never-referenced declaration is not flagged (unused, not single-use)",
			files: map[string]string{
				"p.go": "package p\n\nconst unreferenced = 1\n\nfunc F() int { return 0 }\n",
			},
			changed: []string{"p.go"},
			expect:  0,
		},
	}
}

func singleUseCrossFileCases() []singleUseCase {
	return []singleUseCase{
		{
			name: "single-use var flagged when its only use lives in an untouched sibling file",
			files: map[string]string{
				"a.go": "package p\n\nvar helperCount = 0\n",
				"b.go": "package p\n\nfunc Track() {\n\thelperCount++\n}\n",
			},
			changed: []string{"a.go"},
			expect:  1,
		},
		{
			name: "var used from two different files is not flagged",
			files: map[string]string{
				"a.go": "package p\n\nvar shared = 1\n",
				"b.go": "package p\n\nfunc A() int { return shared }\n",
				"c.go": "package p\n\nfunc B() int { return shared }\n",
			},
			changed: []string{"a.go"},
			expect:  0,
		},
		{
			name: "a top-level reference disqualifies base but derived is still single-use " +
				"and flagged",
			files: map[string]string{
				"p.go": "package p\n\nvar base = 1\nvar derived = base + 1\n\n" +
					"func UseDerived() int { return derived }\n",
			},
			changed: []string{"p.go"},
			expect:  1,
		},
	}
}

func TestAnalyze_ParamsDisableEachKind(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"p.go": "package p\n\nconst c = 1\nvar v = 1\n\n" +
			"func UseBoth() int { return c + v }\n",
	}

	testCases := []struct {
		name      string
		params    pass.ParamSet
		wantConst bool
		wantVar   bool
	}{
		{name: "both enabled by default", params: nil, wantConst: true, wantVar: true},
		{
			name:      "const disabled",
			params:    pass.ParamSet{"check_const": false},
			wantConst: false, wantVar: true,
		},
		{
			name:      "var disabled",
			params:    pass.ParamSet{"check_var": false},
			wantConst: true, wantVar: false,
		},
		{
			name:      "both disabled",
			params:    pass.ParamSet{"check_const": false, "check_var": false},
			wantConst: false, wantVar: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := buildCtx(t, files, []string{"p.go"})
			if testCase.params != nil {
				ctx.Config = &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{
					"single-use-declaration": {Params: testCase.params},
				}}
			}
			got := NewAnalyzer().Analyze(ctx)
			gotConst, gotVar := false, false
			for _, finding := range got {
				switch finding.Message {
				case "c is referenced only inside UseBoth":
					gotConst = true
				case "v is referenced only inside UseBoth":
					gotVar = true
				}
			}
			if gotConst != testCase.wantConst || gotVar != testCase.wantVar {
				t.Fatalf(
					"const flagged = %v (want %v), var flagged = %v (want %v): %+v",
					gotConst, testCase.wantConst, gotVar, testCase.wantVar, got,
				)
			}
		})
	}
}

func TestAnalyze_NoContext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ctx  *pass.Context
	}{
		{
			name: "no AST cache",
			ctx: &pass.Context{
				Sources: source.NewFSSourceProvider(fstest.MapFS{}, "", source.SkipSet{}),
			},
		},
		{name: "no sources", ctx: &pass.Context{ASTCache: astcache.New()}},
		{name: "zero value", ctx: &pass.Context{}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := NewAnalyzer().Analyze(testCase.ctx); len(got) != 0 {
				t.Errorf("findings = %+v, want none", got)
			}
		})
	}
}
