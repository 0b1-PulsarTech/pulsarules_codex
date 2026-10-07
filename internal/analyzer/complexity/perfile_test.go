package complexity

import (
	"context"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

const oneLineSource = "package foo\n\n" +
	"func A(a, b, c, d, e, f bool) { _ = 7 }; func B(a, b, c, d, e, f bool) { _ = 8 }\n"

func flagMessages(name string) []string {
	params := []string{"a", "b", "c", "d", "e", "f"}
	messages := make([]string, 0, len(params))
	for _, param := range params {
		messages = append(messages, "flag argument \""+param+"\" in "+name)
	}
	return messages
}

func goOnlyMessages(name, literal string) []string {
	return append(flagMessages(name), "magic number "+literal+" in "+name)
}

func perFunctionMessages(name, literal string) []string {
	head := name + " has 6 parameters, max 5"
	return append([]string{head}, goOnlyMessages(name, literal)...)
}

type checkChangedCase struct {
	name    string
	source  string
	summary string
	want    []string
}

func checkChangedCases() []checkChangedCase {
	const stubbedSource = "package foo\n\n" + "func stub(int) int; " +
		"func A(a, b, c, d, e, f bool) { _ = 7 }; func B(a, b, c, d, e, f bool) { _ = 8 }\n"
	both := append(perFunctionMessages("A", "7"), perFunctionMessages("B", "8")...)
	return slices.Concat([]checkChangedCase{
		{
			name:    "functions on one line report each function whole",
			source:  oneLineSource,
			summary: oneLineSource,
			want:    both,
		},
		{
			name:    "bodyless declaration on the same line is not paired",
			source:  stubbedSource,
			summary: oneLineSource,
			want:    both,
		},
	}, fallbackCases())
}

func fallbackCases() []checkChangedCase {
	const firstOnlySource = "package foo\n\n" + "func A(a, b, c, d, e, f bool) { _ = 7 }\n"
	const splitLineSource = "package foo\n\n" +
		"func A(a, b, c, d, e, f bool) { _ = 7 }\n" +
		"func B(a, b, c, d, e, f bool) { _ = 8 }\n"
	goOnly := append(goOnlyMessages("A", "7"), goOnlyMessages("B", "8")...)
	return []checkChangedCase{
		{
			name:   "no summary falls back to the go-only findings",
			source: oneLineSource,
			want:   goOnly,
		},
		{
			name:    "fewer summaries than declarations report neutral then go",
			source:  oneLineSource,
			summary: firstOnlySource,
			want:    append([]string{"A has 6 parameters, max 5"}, goOnly...),
		},
		{
			name:    "summary lines that differ from the declarations fall back",
			source:  oneLineSource,
			summary: splitLineSource,
			want: append(
				[]string{"A has 6 parameters, max 5", "B has 6 parameters, max 5"}, goOnly...,
			),
		},
	}
}

func TestCheckChanged(t *testing.T) {
	t.Parallel()

	for _, testCase := range checkChangedCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := parsedContext(t, testCase.source)
			if testCase.summary != "" {
				ctx.Syntax = summaries(parsedContext(t, testCase.summary).ASTCache, "foo.go")
			}
			messages := analyzeMessages(ctx)
			if !slices.Equal(messages, testCase.want) {
				t.Fatalf("messages = %q, want %q", messages, testCase.want)
			}
		})
	}
}

func parsedContext(t *testing.T, source string) *pass.Context {
	t.Helper()

	cache := astcache.New()
	if _, err := cache.Parse("foo.go", []byte(source)); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return &pass.Context{
		ASTCache:     cache,
		Syntax:       emptySyntax(),
		ChangedFiles: []pass.FileChange{{Path: "foo.go", Extension: ".go"}},
	}
}

func analyzeMessages(ctx *pass.Context) []string {
	got := NewAnalyzer().Analyze(ctx)
	messages := make([]string, 0, len(got))
	for _, finding := range got {
		messages = append(messages, finding.Message)
	}
	return messages
}

func emptySyntax() *syntax.Cache {
	read := func(string) ([]byte, bool) { return nil, true }
	return syntax.NewCache(context.Background(), nil, syntax.Corpus{Read: read})
}
