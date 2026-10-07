package controlflow

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
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
			name: "no cache",
			ctx:  &pass.Context{},
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

func goContext(t *testing.T, source string) *pass.Context {
	t.Helper()

	cache := astcache.New()
	if _, err := cache.Parse("foo.go", []byte(source)); err != nil {
		t.Fatalf("parse: %v", err)
	}
	read := func(string) ([]byte, bool) { return []byte(source), true }
	return &pass.Context{
		ASTCache:     cache,
		ChangedFiles: []pass.FileChange{{Path: "foo.go", Extension: ".go"}},
		Syntax: syntax.NewCache(
			context.Background(), []syntax.Provider{gosyntax.New(cache)},
			syntax.Corpus{Sources: []syntax.Source{{Path: "foo.go"}}, Read: read},
		),
	}
}

func TestAnalyze_OrdersFindingsFromTheSyntaxCache(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name: "nesting comes from the syntax cache",
			source: "package foo\nfunc f(ok bool) {\n" +
				"\tif ok {\n\t\tif ok {\n\t\t\tif ok {\n\t\t\t\tif ok {\n\t\t\t\t\tif ok {\n" +
				"\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t}\n}\n",
			want: []string{"7:nesting depth 5 exceeds max 4"},
		},
		{
			name: "elses interleave with nesting in walk order",
			source: "package foo\nfunc f(ok bool) {\n" +
				"\tif ok {\n\t\tif ok {\n\t\t\tif ok {\n\t\t\t\tif ok {\n" +
				"\t\t\t\t\tif ok {\n\t\t\t\t\t} else { return }\n" +
				"\t\t\t\t}\n\t\t\t}\n\t\t}\n\t}\n}\n",
			want: []string{
				"7:nesting depth 5 exceeds max 4",
				"8:else block contains return; use early return instead",
				"8:nesting depth 5 exceeds max 4",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			findings := NewAnalyzer().Analyze(goContext(t, testCase.source))
			got := make([]string, 0, len(findings))
			for _, finding := range findings {
				got = append(got, fmt.Sprintf("%d:%s", finding.Line, finding.Message))
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("findings = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestCheckNesting(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		blocks []syntax.Block
		want   []int
	}{
		{name: "at the limit", blocks: []syntax.Block{{Line: 1, Depth: 4}}},
		{
			name:   "beyond the limit",
			blocks: []syntax.Block{{Line: 1, Depth: 5}, {Line: 9, Depth: 6}, {Line: 12, Depth: 2}},
			want:   []int{1, 9},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			analyzer := NewAnalyzer()
			got := analyzer.checkBlocks(
				pass.FileChange{Path: "a.dart"}, &syntax.File{Blocks: testCase.blocks},
				nil, analyzer.reporter,
			)
			var lines []int
			for _, finding := range got {
				lines = append(lines, finding.Line)
			}
			if !slices.Equal(lines, testCase.want) {
				t.Fatalf("lines = %v, want %v", lines, testCase.want)
			}
		})
	}
}
