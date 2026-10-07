package syntaxfiles

import (
	"context"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
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
		{name: "nil cache visits nothing", changed: []string{"a.go"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &core.AnalysisContext{}
			var sources []syntax.Source
			for _, path := range testCase.changed {
				ctx.ChangedFiles = append(ctx.ChangedFiles, core.FileChange{Path: path})
				sources = append(sources, syntax.Source{Path: path})
			}
			if testCase.withTree {
				read := func(string) ([]byte, bool) { return nil, true }
				ctx.Syntax = syntax.NewCache(
					context.Background(), []syntax.Provider{stubProvider{}},
					syntax.Corpus{Sources: sources, Read: read},
				)
			}
			var visited []string
			Each(ctx, func(fc core.FileChange, _ *syntax.File) []core.Finding {
				visited = append(visited, fc.Path)
				return nil
			})
			if !slices.Equal(visited, testCase.want) {
				t.Fatalf("visited = %v, want %v", visited, testCase.want)
			}
		})
	}
}
