package imports

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/gosyntax"
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
			name: "no changed files",
			ctx:  &core.AnalysisContext{ASTCache: astcache.New()},
		},
		{
			name: "non-go file skipped",
			ctx: &core.AnalysisContext{
				ASTCache:     astcache.New(),
				ChangedFiles: []core.FileChange{{Path: "foo.md", Extension: ".md"}},
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

			fc := core.FileChange{Path: "foo.go", Extension: ".go"}
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

	got := NewAnalyzer().Analyze(&core.AnalysisContext{
		ProjectDir:   projectDir,
		ASTCache:     cache,
		Syntax:       summaries(cache, projectDir, path),
		ChangedFiles: []core.FileChange{{Path: path, Extension: ".go"}},
	})
	if len(got) == 0 {
		t.Fatal("a local import grouped before the external one went unreported: the analyzer " +
			"did not recognise " + modPath + " as this project's own module")
	}
}

func summaries(cache *astcache.Cache, projectDir, path string) *syntax.Cache {
	read := func(string) ([]byte, bool) { return nil, true }
	return syntax.NewCache(
		context.Background(),
		[]syntax.Provider{gosyntax.New(cache)},
		syntax.Corpus{Root: projectDir, Sources: []syntax.Source{{Path: path}}, Read: read},
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
				Language: syntax.LangGo, ImportGroups: groups,
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
				Language:     syntax.LangDart,
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
				Language:     syntax.LangTypeScript,
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
				core.FileChange{Path: "a.dart"}, &testCase.file, NewAnalyzer().reporter,
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
