package facts

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

type nameCase struct {
	name string
	text string
	want []syntax.Name
}

func nameCases() []nameCase {
	return slices.Concat(declarationNameCases(), bindingNameCases(), overloadNameCases())
}

func declarationNameCases() []nameCase {
	return slices.Concat(declarationNameCasesHead(), declarationNameCasesTail())
}

func declarationNameCasesHead() []nameCase {
	return []nameCase{
		{
			name: "exported declarations",
			text: "export function run() {}\nexport class Box {}\nexport interface Shape {}\n" +
				"export type Id = string;\nexport enum Color { Red }\nexport const limit = 1;\n" +
				"export let count = 2;\nnamespace Hidden {}\n",
			want: []syntax.Name{
				{Name: "run", Kind: syntax.NameFunction, Line: 1, Exported: true},
				{Name: "Box", Kind: syntax.NameType, Line: 2, Exported: true},
				{Name: "Shape", Kind: syntax.NameType, Line: 3, Exported: true},
				{Name: "Id", Kind: syntax.NameType, Line: 4, Exported: true},
				{Name: "Color", Kind: syntax.NameType, Line: 5, Exported: true},
				{Name: "Red", Kind: syntax.NameEnumValue, Line: 5},
				{Name: "limit", Kind: syntax.NameConstant, Line: 6, Exported: true},
				{Name: "count", Kind: syntax.NameVariable, Line: 7, Exported: true},
				{Name: "Hidden", Kind: syntax.NameType, Line: 8},
			},
		},
		{
			name: "members are never exported",
			text: "export class Box {\n  size = 1;\n  grow() {}\n  get w() { return 1 }\n  #p " +
				"= 2;\n}\n",
			want: []syntax.Name{
				{Name: "Box", Kind: syntax.NameType, Line: 1, Exported: true},
				{Name: "size", Kind: syntax.NameField, Line: 2},
				{Name: "grow", Kind: syntax.NameMethod, Line: 3},
				{Name: "w", Kind: syntax.NameMethod, Line: 4},
				{Name: "#p", Kind: syntax.NameField, Line: 5},
			},
		},
	}
}

func declarationNameCasesTail() []nameCase {
	return []nameCase{
		{
			name: "interface members and type parameters",
			text: "interface Box<T> {\n  size: T;\n  grow(by: number): void;\n}\n",
			want: []syntax.Name{
				{Name: "Box", Kind: syntax.NameType, Line: 1},
				{Name: "T", Kind: syntax.NameTypeParam, Line: 1},
				{Name: "size", Kind: syntax.NameField, Line: 2},
				{Name: "grow", Kind: syntax.NameMethod, Line: 3},
				{Name: "by", Kind: syntax.NameParameter, Line: 3},
			},
		},
		{
			name: "parameters with patterns skip this",
			text: "function f(this: X, a, { b, c: d }, [e, , ...g], h = 1) {}\n",
			want: []syntax.Name{
				{Name: "f", Kind: syntax.NameFunction, Line: 1},
				{Name: "a", Kind: syntax.NameParameter, Line: 1},
				{Name: "b", Kind: syntax.NameParameter, Line: 1},
				{Name: "d", Kind: syntax.NameParameter, Line: 1},
				{Name: "e", Kind: syntax.NameParameter, Line: 1},
				{Name: "g", Kind: syntax.NameParameter, Line: 1},
				{Name: "h", Kind: syntax.NameParameter, Line: 1},
			},
		},
		{
			name: "loop and catch variables",
			text: "for (let i = 0;;) {}\nfor (const k in o) {}\nfor (const [x, y] of o) {}\n" +
				"try {} catch (err) {}\ntry {} catch {}\n",
			want: []syntax.Name{
				{Name: "i", Kind: syntax.NameLoopVariable, Line: 1},
				{Name: "k", Kind: syntax.NameLoopVariable, Line: 2},
				{Name: "x", Kind: syntax.NameLoopVariable, Line: 3},
				{Name: "y", Kind: syntax.NameLoopVariable, Line: 3},
				{Name: "err", Kind: syntax.NameCatchParameter, Line: 4},
			},
		},
	}
}

func bindingNameCases() []nameCase {
	return []nameCase{
		{
			name: "destructured declarations take the list kind",
			text: "const { a, b: c } = o;\nlet [d] = o;\n",
			want: []syntax.Name{
				{Name: "a", Kind: syntax.NameConstant, Line: 1},
				{Name: "c", Kind: syntax.NameConstant, Line: 1},
				{Name: "d", Kind: syntax.NameVariable, Line: 2},
			},
		},
		{
			name: "labels",
			text: "outer: for (;;) { break outer; }\n",
			want: []syntax.Name{
				{Name: "outer", Kind: syntax.NameLabel, Line: 1},
			},
		},
		{
			name: "import aliases only",
			text: "import def from 'a';\nimport * as ns from 'b';\nimport { x, y as z } from " +
				"'c';\nimport q = require('d');\nimport 'e';\n",
			want: []syntax.Name{
				{Name: "def", Kind: syntax.NameImportAlias, Line: 1},
				{Name: "ns", Kind: syntax.NameImportAlias, Line: 2},
				{Name: "z", Kind: syntax.NameImportAlias, Line: 3},
				{Name: "q", Kind: syntax.NameImportAlias, Line: 4},
			},
		},
		{
			name: "function expressions and class expressions",
			text: "const f = function inner() {};\nconst c = class Made {};\n",
			want: []syntax.Name{
				{Name: "f", Kind: syntax.NameConstant, Line: 1},
				{Name: "inner", Kind: syntax.NameFunction, Line: 1},
				{Name: "c", Kind: syntax.NameConstant, Line: 2},
				{Name: "Made", Kind: syntax.NameType, Line: 2},
			},
		},
		{
			name: "missing identifiers from broken code are skipped",
			text: "const = 1;\nfunction () {}\n",
		},
	}
}

func TestNames(t *testing.T) {
	t.Parallel()

	testCases := nameCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := collectOf(t, testCase.text, Options{}).Names
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("names\n got %+v\nwant %+v", got, testCase.want)
			}
		})
	}
}

func overloadNameCases() []nameCase {
	return []nameCase{
		{
			name: "a bodyless declaration before another name keeps its names",
			text: "declare function f(a: string): void;\nfunction g() {}\n",
			want: []syntax.Name{
				{Name: "f", Kind: syntax.NameFunction, Line: 1},
				{Name: "a", Kind: syntax.NameParameter, Line: 1},
				{Name: "g", Kind: syntax.NameFunction, Line: 2},
			},
		},
		{
			name: "constructor overload signatures add no names",
			text: "class B {\n  constructor(a: string);\n  constructor(b: any) {}\n}\n",
			want: []syntax.Name{
				{Name: "B", Kind: syntax.NameType, Line: 1},
				{Name: "b", Kind: syntax.NameParameter, Line: 3},
			},
		},
		{
			name: "a signature before another member kind is not an overload",
			text: "class A {\n  m(a: string): void;\n  get m() { return 1 }\n}\n",
			want: []syntax.Name{
				{Name: "A", Kind: syntax.NameType, Line: 1},
				{Name: "m", Kind: syntax.NameMethod, Line: 2},
				{Name: "a", Kind: syntax.NameParameter, Line: 2},
				{Name: "m", Kind: syntax.NameMethod, Line: 3},
			},
		},
		{
			name: "overload signatures add no names",
			text: "function f(a: string): void;\nfunction f(b: any) {}\n",
			want: []syntax.Name{
				{Name: "f", Kind: syntax.NameFunction, Line: 2},
				{Name: "b", Kind: syntax.NameParameter, Line: 2},
			},
		},
	}
}
