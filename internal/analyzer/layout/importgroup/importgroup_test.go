package importgroup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
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
			name: "no changed files",
			ctx:  &pass.Context{ASTCache: astcache.New()},
		},
		{
			name: "non-go file skipped",
			ctx: &pass.Context{
				ASTCache:     astcache.New(),
				ChangedFiles: []pass.FileChange{{Path: "foo.md", Extension: ".md"}},
			},
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

type checkFileTestCase struct {
	name   string
	source string
	expect int
}

func checkFileCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name:   "no imports",
			source: "package foo\n",
			expect: 0,
		},
		{
			name: "correct order std ext mod",
			source: "package foo\n" +
				"import (\n" +
				`	"fmt"` + "\n" +
				`	"os"` + "\n" +
				"\n" +
				`	"gopkg.in/yaml.v3"` + "\n" +
				"\n" +
				`	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"` + "\n" +
				")\n",
			expect: 0,
		},
		{
			name: "module import before external",
			source: "package foo\n" +
				"import (\n" +
				`	"fmt"` + "\n" +
				"\n" +
				`	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"` + "\n" +
				"\n" +
				`	"gopkg.in/yaml.v3"` + "\n" +
				")\n",
			expect: 1,
		},
		{
			name: "external before stdlib",
			source: "package foo\n" +
				"import (\n" +
				`	"gopkg.in/yaml.v3"` + "\n" +
				"\n" +
				`	"fmt"` + "\n" +
				")\n",
			expect: 1,
		},
	}
}

func TestCheckFile(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	const modPath = "github.com/0b1-PulsarTech/pulsarules_codex"

	for _, testCase := range checkFileCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			projectDir := t.TempDir()
			writeSource(t, filepath.Join(projectDir, "go.mod"), "module "+modPath+"\n")
			path := filepath.Join(projectDir, "foo.go")
			writeSource(t, path, testCase.source)
			cache := astcache.New()
			if _, err := cache.Parse(path, []byte(testCase.source)); err != nil {
				t.Fatal(err)
			}
			summary, ok := summaries(cache, projectDir, path).Get(path)
			if !ok {
				t.Fatal("no summary")
			}

			fc := pass.FileChange{Path: "foo.go", Extension: ".go"}
			got := checkFile(fc, summary, analyzer.reporter)
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d", len(got), testCase.expect)
			}
		})
	}
}

func TestAnalyze_ForeignModule(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	const modPath = "example.com/other"
	writeSource(t, filepath.Join(projectDir, "go.mod"), "module "+modPath+"\n\ngo 1.26\n")

	src := "package foo\n\nimport (\n\t\"fmt\"\n\n\t\"" + modPath +
		"/bar\"\n\n\t\"github.com/x/y\"\n)\n\nvar _ = fmt.Sprint\n"
	path := filepath.Join(projectDir, "foo.go")
	writeSource(t, path, src)

	cache := astcache.New()
	if _, err := cache.Parse(path, []byte(src)); err != nil {
		t.Fatalf("parse: %v", err)
	}

	got := NewAnalyzer().Analyze(&pass.Context{
		ProjectDir:   projectDir,
		ASTCache:     cache,
		Syntax:       summaries(cache, projectDir, path),
		ChangedFiles: []pass.FileChange{{Path: path, Extension: ".go"}},
	})
	if len(got) == 0 {
		t.Fatal("a local import grouped before the external one went unreported: the analyzer " +
			"did not recognise " + modPath + " as this project's own module")
	}
}

func summaries(cache *astcache.Cache, projectDir, path string) *syntax.Cache {
	read := func(string) ([]byte, bool) { return nil, true }
	module := func() string {
		modulePath, _ := pass.ModulePath(projectDir)
		return modulePath
	}
	return syntax.NewCache(
		[]syntax.Provider{gosyntax.New(cache)},
		syntax.Corpus{
			Root: projectDir, Sources: []syntax.Source{{Path: path}}, Read: read,
			Module: module,
		},
	)
}

func writeSource(tb testing.TB, path, body string) {
	tb.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		tb.Fatalf("write %s: %v", path, err)
	}
}

type languageCase struct {
	name           string
	file           syntax.File
	wantMessage    string
	wantSuggestion string
}

func languageCases() []languageCase {
	groups := []string{"stdlib", "external", "example.com/app"}
	return []languageCase{
		{
			name: "go out of order import",
			file: syntax.File{
				Language: language.Go, ImportGroups: groups,
				Imports: []syntax.Import{
					{Literal: `"a.com/x"`, Line: 3, Group: 1},
					{Literal: `"fmt"`, Line: 4, Group: 0},
				},
			},
			wantMessage:    `import "fmt" out of order (stdlib → external → example.com/app)`,
			wantSuggestion: "reorder imports to follow: stdlib, external, this module",
		},
		{
			name: "dart keeps its own group names",
			file: syntax.File{
				Language:     language.Dart,
				ImportGroups: []string{"dart:", "package:", "relative"},
				Imports: []syntax.Import{
					{Literal: `'package:a/a.dart'`, Line: 3, Group: 1},
					{Literal: `'dart:io'`, Line: 4, Group: 0},
				},
			},
			wantMessage:    `import 'dart:io' out of order (dart: → package: → relative)`,
			wantSuggestion: "reorder imports to follow: dart:, package:, relative",
		},
		{
			name: "typescript keeps its own group names",
			file: syntax.File{
				Language:     language.TypeScript,
				ImportGroups: []string{"node:", "external", "local"},
				Imports: []syntax.Import{
					{Literal: `'./a'`, Line: 3, Group: 2},
					{Literal: `'pkg'`, Line: 4, Group: 1},
				},
			},
			wantMessage:    `import 'pkg' out of order (node: → external → local)`,
			wantSuggestion: "reorder imports to follow: node:, external, local",
		},
		{
			name: "ordered imports",
			file: syntax.File{ImportGroups: groups, Imports: []syntax.Import{
				{Literal: `"fmt"`, Line: 3, Group: 0},
				{Literal: `"a.com/x"`, Line: 4, Group: 1},
			}},
		},
		{
			name: "no group names stays silent",
			file: syntax.File{Imports: []syntax.Import{
				{Literal: `"a"`, Group: 1}, {Literal: `"b"`, Group: 0},
			}},
		},
		{name: "no imports", file: syntax.File{ImportGroups: groups}},
	}
}

func TestCheckFile_SummaryFromAnyLanguage(t *testing.T) {
	t.Parallel()

	testCases := languageCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := checkFile(
				pass.FileChange{Path: "a.dart"}, &testCase.file, NewAnalyzer().reporter,
			)
			if testCase.wantMessage == "" {
				if len(got) != 0 {
					t.Fatalf("findings = %+v, want none", got)
				}
				return
			}
			if len(got) != 1 || got[0].Message != testCase.wantMessage ||
				got[0].Suggestion != testCase.wantSuggestion {
				t.Fatalf("findings = %+v", got)
			}
		})
	}
}

func TestCheckPaths(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		imports        []syntax.Import
		wantMessage    string
		wantSuggestion string
	}{
		{
			name:           "alias into the own directory",
			imports:        []syntax.Import{{Literal: `'#lib/x'`, Line: 2, Prefer: "./x"}},
			wantMessage:    "import '#lib/x' has a better spelling",
			wantSuggestion: "write it as ./x",
		},
		{
			name:           "parent import covered by an alias",
			imports:        []syntax.Import{{Literal: `'../x'`, Line: 2, Prefer: "#lib/x"}},
			wantMessage:    "import '../x' has a better spelling",
			wantSuggestion: "write it as #lib/x",
		},
		{
			name:           "parent import with no alias",
			imports:        []syntax.Import{{Literal: `'../x'`, Line: 2, Upward: true}},
			wantMessage:    "import '../x' climbs out of its directory",
			wantSuggestion: aliasHint,
		},
		{
			name:           "parent import out of its package",
			imports:        []syntax.Import{{Literal: `'../../x'`, Line: 2, Foreign: true}},
			wantMessage:    "import '../../x' climbs out of its package",
			wantSuggestion: "import it through its package name",
		},
		{
			name:    "plain imports stay silent",
			imports: []syntax.Import{{Literal: `'./x'`}, {Literal: `'pkg'`}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file := syntax.File{Imports: testCase.imports}
			got := checkPaths(pass.FileChange{Path: "a.ts"}, &file, NewAnalyzer().pathReporter)
			if testCase.wantMessage == "" {
				if len(got) != 0 {
					t.Fatalf("findings = %+v, want none", got)
				}
				return
			}
			if len(got) != 1 || got[0].Message != testCase.wantMessage ||
				got[0].Suggestion != testCase.wantSuggestion ||
				got[0].AnalyzerID != PathID ||
				got[0].Severity != finding.SeverityWarning || got[0].Line != 2 {
				t.Fatalf("findings = %+v", got)
			}
		})
	}
}

type pathProvider struct{}

func (pathProvider) Languages() []language.ID { return []language.ID{language.TypeScript} }

func (pathProvider) Summarize(
	_ context.Context, _ syntax.Corpus,
) (map[string]*syntax.File, error) {
	file := &syntax.File{
		Language: language.TypeScript,
		Imports:  []syntax.Import{{Literal: `'#lib/x'`, Line: 2, Prefer: "./x"}},
	}
	return map[string]*syntax.File{"a.ts": file}, nil
}

func TestAnalyze_PathSeverityFromConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		config *pass.AnalysisConfig
		want   finding.Severity
	}{
		{name: "default severity is warning", want: finding.SeverityWarning},
		{
			name: "configured severity overrides the default",
			config: &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{
				PathID: {Params: map[string]any{"severity": "error"}},
			}},
			want: finding.SeverityError,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			read := func(string) ([]byte, bool) { return nil, true }
			corpus := syntax.Corpus{Sources: []syntax.Source{{Path: "a.ts"}}, Read: read}
			providers := []syntax.Provider{pathProvider{}}
			got := NewAnalyzer().Analyze(&pass.Context{
				Syntax:       syntax.NewCache(providers, corpus),
				ChangedFiles: []pass.FileChange{{Path: "a.ts", Extension: ".ts"}},
				Config:       testCase.config,
			})
			if len(got) != 1 || got[0].Severity != testCase.want {
				t.Fatalf("findings = %+v, want one at severity %v", got, testCase.want)
			}
		})
	}
}
