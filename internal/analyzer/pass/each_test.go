package pass

import (
	"context"
	"go/ast"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

type fakeProvider struct {
	contents map[string][]byte
}

func (provider fakeProvider) Read(path string) ([]byte, bool) {
	src, ok := provider.contents[path]
	return src, ok
}

func (provider fakeProvider) Walk(func(path string, ext string) bool) {}

var _ source.Provider = fakeProvider{}

type brokenParseProvider struct{}

func (brokenParseProvider) Languages() []language.ID { return []language.ID{language.Go} }

func (brokenParseProvider) Summarize(
	_ context.Context, corpus syntax.Corpus,
) (map[string]*syntax.File, error) {
	files := map[string]*syntax.File{}
	for _, src := range corpus.Sources {
		file := &syntax.File{Path: src.Path}
		if src.Path == "bad.go" {
			file.ParseError = "broken"
		}
		files[src.Path] = file
	}
	return files, nil
}

func isGoFile(fc FileChange) bool { return fc.Extension == ".go" }

func changedGo(paths ...string) []FileChange {
	changed := make([]FileChange, 0, len(paths))
	for _, path := range paths {
		changed = append(changed, FileChange{Path: path, Extension: ".go"})
	}
	return changed
}

func findingFiles(findings []finding.Finding) []string {
	files := make([]string, 0, len(findings))
	for _, hit := range findings {
		files = append(files, hit.File)
	}
	return files
}

func echoSource(fc FileChange, _ []byte) []finding.Finding {
	return []finding.Finding{{File: fc.Path}}
}

func TestContext_EachSource(t *testing.T) {
	t.Parallel()

	skips := source.NewSkipSet([]string{"testdata"})
	readable := fakeProvider{contents: map[string][]byte{
		"a.go":                 []byte("package a"),
		"b.go":                 []byte("package b"),
		"c.md":                 []byte("# c"),
		"pkg/testdata/skip.go": []byte("package skip"),
	}}
	testCases := []struct {
		name string
		ctx  *Context
		want []string
	}{
		{
			name: "nil Sources yields nothing",
			ctx:  &Context{ChangedFiles: changedGo("a.go")},
		},
		{
			name: "skipped path is not read even when eligible",
			ctx: &Context{
				Sources: readable, Skips: skips,
				ChangedFiles: changedGo("pkg/testdata/skip.go", "a.go"),
			},
			want: []string{"a.go"},
		},
		{
			name: "ineligible extension is skipped",
			ctx: &Context{
				Sources:      readable,
				ChangedFiles: append(changedGo("a.go"), FileChange{Path: "c.md", Extension: ".md"}),
			},
			want: []string{"a.go"},
		},
		{
			name: "unreadable source is skipped",
			ctx: &Context{
				Sources: readable, ChangedFiles: changedGo("missing.go", "a.go"),
			},
			want: []string{"a.go"},
		},
		{
			name: "order of ChangedFiles is preserved",
			ctx: &Context{
				Sources: readable, ChangedFiles: changedGo("b.go", "a.go"),
			},
			want: []string{"b.go", "a.go"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := findingFiles(testCase.ctx.EachSource(isGoFile, echoSource))
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("EachSource() files = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestContext_EachGoFile(t *testing.T) {
	t.Parallel()

	newCache := func() *astcache.Cache {
		cache := astcache.New()
		for _, path := range []string{"a.go", "b.go", "testdata/x.go"} {
			if _, err := cache.Parse(path, []byte("package p\n")); err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
		}
		return cache
	}
	skips := source.NewSkipSet([]string{"testdata"})
	testCases := []struct {
		name string
		ctx  *Context
		want []string
	}{
		{
			name: "nil ASTCache yields nothing",
			ctx:  &Context{ChangedFiles: changedGo("a.go")},
		},
		{
			name: "skipped and unparsed paths are not visited",
			ctx: &Context{
				ASTCache: newCache(), Skips: skips,
				ChangedFiles: changedGo("testdata/x.go", "missing.go", "a.go"),
			},
			want: []string{"a.go"},
		},
		{
			name: "non-Go extension is ineligible",
			ctx: &Context{
				ASTCache:     newCache(),
				ChangedFiles: []FileChange{{Path: "a.go", Extension: ".txt"}},
			},
		},
		{
			name: "order of ChangedFiles is preserved",
			ctx: &Context{
				ASTCache: newCache(), ChangedFiles: changedGo("b.go", "a.go"),
			},
			want: []string{"b.go", "a.go"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := testCase.ctx.EachGoFile(func(fc FileChange, file *ast.File) []finding.Finding {
				if file == nil {
					t.Fatalf("nil *ast.File for %s", fc.Path)
				}
				return echoSource(fc, nil)
			})
			if !slices.Equal(findingFiles(got), testCase.want) {
				t.Fatalf("EachGoFile() files = %v, want %v", findingFiles(got), testCase.want)
			}
		})
	}
}

func TestContext_EachSyntaxFile(t *testing.T) {
	t.Parallel()

	skips := source.NewSkipSet([]string{"testdata", "vendor"})
	testCases := []struct {
		name     string
		changed  []string
		withTree bool
		want     []string
	}{
		{"visits every file in order", []string{"b.go", "a.go"}, true, []string{"b.go", "a.go"}},
		{"parse error is skipped", []string{"a.go", "bad.go"}, true, []string{"a.go"}},
		{"file no provider handles is skipped", []string{"a.txt"}, true, nil},
		{
			"skipped dirs are not visited",
			[]string{"a.go", "testdata/x.go", "vendor/m/y.go"},
			true,
			[]string{"a.go"},
		},
		{"nil cache visits nothing", []string{"a.go"}, false, nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &Context{Skips: skips}
			var corpus []syntax.Source
			for _, path := range testCase.changed {
				ctx.ChangedFiles = append(ctx.ChangedFiles, FileChange{Path: path})
				corpus = append(corpus, syntax.Source{Path: path})
			}
			if testCase.withTree {
				read := func(string) ([]byte, bool) { return nil, true }
				ctx.Syntax = syntax.NewCache(
					[]syntax.Provider{brokenParseProvider{}},
					syntax.Corpus{Sources: corpus, Read: read},
				)
			}
			got := ctx.EachSyntaxFile(func(fc FileChange, _ *syntax.File) []finding.Finding {
				return echoSource(fc, nil)
			})
			if !slices.Equal(findingFiles(got), testCase.want) {
				t.Fatalf("EachSyntaxFile() files = %v, want %v", findingFiles(got), testCase.want)
			}
		})
	}
}
