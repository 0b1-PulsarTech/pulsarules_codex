package syntaxfiles

import (
	"context"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

type stubProvider struct{}

func (stubProvider) Language() syntax.Language { return syntax.LangGo }

func (stubProvider) Handles(ext string) bool { return ext == ".go" }

func (stubProvider) Summarize(
	_ context.Context, corpus syntax.Corpus,
) (map[string]*syntax.File, error) {
	files := map[string]*syntax.File{}
	for _, source := range corpus.Sources {
		file := &syntax.File{Path: source.Path}
		if source.Path == "bad.go" {
			file.ParseError = "broken"
		}
		files[source.Path] = file
	}
	return files, nil
}

func TestEach(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		changed  []string
		withTree bool
		want     []string
	}{
		{
			name:     "visits every summarised file",
			changed:  []string{"a.go", "b.go"},
			withTree: true,
			want:     []string{"a.go", "b.go"},
		},
		{
			name:     "skips a parse error",
			changed:  []string{"a.go", "bad.go"},
			withTree: true,
			want:     []string{"a.go"},
		},
		{name: "skips a file no provider handles", changed: []string{"a.txt"}, withTree: true},
		{
			name:     "skips a file under a skipped dir",
			changed:  []string{"a.go", "testdata/x.go", "vendor/m/y.go"},
			withTree: true,
			want:     []string{"a.go"},
		},
		{name: "nil cache visits nothing", changed: []string{"a.go"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &pass.Context{
				Skips: source.NewSkipSet([]string{"testdata", "vendor"}),
			}
			var corpus []syntax.Source
			for _, path := range testCase.changed {
				ctx.ChangedFiles = append(ctx.ChangedFiles, pass.FileChange{Path: path})
				corpus = append(corpus, syntax.Source{Path: path})
			}
			if testCase.withTree {
				read := func(string) ([]byte, bool) { return nil, true }
				ctx.Syntax = syntax.NewCache(
					context.Background(), []syntax.Provider{stubProvider{}},
					syntax.Corpus{Sources: corpus, Read: read},
				)
			}
			var visited []string
			Each(ctx, func(fc pass.FileChange, _ *syntax.File) []pass.Finding {
				visited = append(visited, fc.Path)
				return nil
			})
			if !slices.Equal(visited, testCase.want) {
				t.Fatalf("visited = %v, want %v", visited, testCase.want)
			}
		})
	}
}
