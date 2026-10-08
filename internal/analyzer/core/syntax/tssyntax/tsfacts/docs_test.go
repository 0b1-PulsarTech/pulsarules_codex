package tsfacts

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

type docSiteCase struct {
	name string
	text string
	want []syntax.DocSite
}

func docSiteCases() []docSiteCase {
	return slices.Concat(docSiteCasesHead(), docSiteCasesTail())
}

func docSiteCasesHead() []docSiteCase {
	return []docSiteCase{
		{
			name: "undocumented declarations",
			text: "function f() {}\nclass A { m() {} }\nconst v = 1;\n",
		},
		{
			name: "top level kinds",
			text: "/** a */\nfunction f() {}\n/** b */\nclass A {}\n/** c */\ninterface I {}\n" +
				"/** d */\ntype T = 1;\n/** e */\nenum E { X }\n/** f */\nconst v = 1;\n/** g " +
				"*/\nconst fn = () => {};\n/** h */\nconst wrapped = (function () {}) as X;\n",
			want: []syntax.DocSite{
				{Symbol: "f", Kind: syntax.DocFunction, Line: 1},
				{Symbol: "A", Kind: syntax.DocType, Line: 3},
				{Symbol: "I", Kind: syntax.DocType, Line: 5},
				{Symbol: "T", Kind: syntax.DocType, Line: 7},
				{Symbol: "E", Kind: syntax.DocType, Line: 9},
				{Symbol: "v", Kind: syntax.DocVariable, Line: 11},
				{Symbol: "fn", Kind: syntax.DocFunction, Line: 13},
				{Symbol: "wrapped", Kind: syntax.DocFunction, Line: 15},
			},
		},
		{
			name: "members carry their owner",
			text: "class A {\n  /** a */\n  x = 1;\n  /** b */\n  constructor() {}\n  /** c " +
				"*/\n  m() {}\n  /** d */\n  get g() { return 1 }\n}\ninterface I {\n  /** e " +
				"*/\n  p: number;\n  /** f */\n  q(): void;\n}\nenum E {\n  /** g */\n  V,\n}\n",
			want: []syntax.DocSite{
				{Symbol: "A.x", Kind: syntax.DocField, Line: 2},
				{Symbol: "A.constructor", Kind: syntax.DocConstructor, Line: 4},
				{Symbol: "A.m", Kind: syntax.DocMethod, Line: 6},
				{Symbol: "A.g", Kind: syntax.DocMethod, Line: 8},
				{Symbol: "I.p", Kind: syntax.DocField, Line: 12},
				{Symbol: "I.q", Kind: syntax.DocMethod, Line: 14},
				{Symbol: "E.V", Kind: syntax.DocField, Line: 18},
			},
		},
	}
}

func docSiteCasesTail() []docSiteCase {
	return []docSiteCase{
		{
			name: "namespaces nest their owners",
			text: "/** a */\nnamespace Outer {\n  /** b */\n  export function f() {}\n}\n/** c " +
				"*/\nnamespace One.Two {\n  /** d */\n  export class C {}\n}\n",
			want: []syntax.DocSite{
				{Symbol: "Outer", Kind: syntax.DocType, Line: 1},
				{Symbol: "Outer.f", Kind: syntax.DocFunction, Line: 3},
				{Symbol: "One", Kind: syntax.DocType, Line: 6},
				{Symbol: "One.Two.C", Kind: syntax.DocType, Line: 8},
			},
		},
		{
			name: "exported declarations and decorators",
			text: "/** a */\nexport function f() {}\n/** b */\nexport default class {}\n",
			want: []syntax.DocSite{
				{Symbol: "f", Kind: syntax.DocFunction, Line: 1},
				{Symbol: "", Kind: syntax.DocType, Line: 3},
			},
		},
		{
			name: "declarations inside bodies are not sites",
			text: "function f() {\n  /** a */\n  function g() {}\n}\n",
		},
		{
			name: "plain comments are not doc comments",
			text: "// note\nfunction f() {}\n/* block */\nfunction g() {}\n",
		},
	}
}

func TestDocSites(t *testing.T) {
	t.Parallel()

	testCases := docSiteCases()
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
