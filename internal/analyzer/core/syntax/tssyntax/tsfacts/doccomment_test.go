package tsfacts

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

type docCommentCase struct {
	name string
	text string
	want []syntax.DocSite
}

func docCommentCases() []docCommentCase {
	return slices.Concat(docCommentCasesHead(), docCommentCasesTail())
}

func docCommentCasesHead() []docCommentCase {
	return []docCommentCase{
		{
			name: "blank line detaches the comment",
			text: "/** a */\n\nfunction f() {}\n",
		},
		{
			name: "no blank line attaches it",
			text: "/** a\n * b\n */\nfunction f() {}\n",
			want: []syntax.DocSite{{Symbol: "f", Kind: syntax.DocFunction, Line: 1}},
		},
		{
			name: "last doc comment wins",
			text: "/** one */\n/** two */\nfunction f() {}\n",
			want: []syntax.DocSite{{Symbol: "f", Kind: syntax.DocFunction, Line: 2}},
		},
		{
			name: "file header before a blank line is not the doc",
			text: "/** header */\n\n/** real */\nfunction f() {}\n",
			want: []syntax.DocSite{{Symbol: "f", Kind: syntax.DocFunction, Line: 3}},
		},
		{
			name: "empty block comment is not a doc comment",
			text: "/**/\nfunction f() {}\n",
		},
		{
			name: "directive line comment next to prose is mixed",
			text: "// eslint-disable-next-line\n/** Prose. */\nfunction f() {}\n",
			want: []syntax.DocSite{
				{Symbol: "f", Kind: syntax.DocFunction, Line: 2, MixedDirective: true},
			},
		},
	}
}

func docCommentCasesTail() []docCommentCase {
	return []docCommentCase{
		{
			name: "directive inside the doc next to prose is mixed",
			text: "/**\n * Prose.\n * @ts-ignore\n */\nfunction f() {}\n",
			want: []syntax.DocSite{
				{Symbol: "f", Kind: syntax.DocFunction, Line: 1, MixedDirective: true},
			},
		},
		{
			name: "directive only doc is exempt not mixed",
			text: "/** eslint-disable */\nfunction f() {}\n",
			want: []syntax.DocSite{{Symbol: "f", Kind: syntax.DocFunction, Line: 1, Exempt: true}},
		},
		{
			name: "empty doc is exempt",
			text: "/** */\nfunction f() {}\n",
			want: []syntax.DocSite{{Symbol: "f", Kind: syntax.DocFunction, Line: 1, Exempt: true}},
		},
		{
			name: "short deprecation is exempt",
			text: "/**\n * @deprecated use g\n */\nfunction f() {}\n",
			want: []syntax.DocSite{{Symbol: "f", Kind: syntax.DocFunction, Line: 1, Exempt: true}},
		},
		{
			name: "long deprecation is not",
			text: "/**\n * @deprecated use g\n *\n * because reasons\n */\nfunction f() {}\n",
			want: []syntax.DocSite{{Symbol: "f", Kind: syntax.DocFunction, Line: 1}},
		},
		{
			name: "simplification marker is exempt",
			text: "/** simplification: drops the cache */\nfunction f() {}\n",
			want: []syntax.DocSite{{Symbol: "f", Kind: syntax.DocFunction, Line: 1, Exempt: true}},
		},
		{
			name: "comment on the same line as the previous token is not leading",
			text: "interface I { /** a */ p: number }\n",
		},
	}
}

func TestDocComment(t *testing.T) {
	t.Parallel()

	testCases := docCommentCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := collectOf(t, testCase.text, Options{}).DocSites
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("doc sites\n got %+v\nwant %+v", got, testCase.want)
			}
		})
	}
}
