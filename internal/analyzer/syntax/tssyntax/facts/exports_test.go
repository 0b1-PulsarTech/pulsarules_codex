package facts

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func exportNameCases() []nameCase {
	return []nameCase{
		{
			name: "namespace members are not exported",
			text: "export namespace Ns {\n  export function inner() {}\n  export const value = " +
				"1;\n}\n",
			want: []syntax.Name{
				{Name: "Ns", Kind: syntax.NameType, Line: 1, Exported: true},
				{Name: "inner", Kind: syntax.NameFunction, Line: 2},
				{Name: "value", Kind: syntax.NameConstant, Line: 3},
			},
		},
		{
			name: "export list marks local names",
			text: "function a() {}\nconst b = 1;\nconst c = 2;\nclass D {}\nexport { a, b as " +
				"renamed };\n",
			want: []syntax.Name{
				{Name: "a", Kind: syntax.NameFunction, Line: 1, Exported: true},
				{Name: "b", Kind: syntax.NameConstant, Line: 2, Exported: true},
				{Name: "c", Kind: syntax.NameConstant, Line: 3},
				{Name: "D", Kind: syntax.NameType, Line: 4},
			},
		},
		{
			name: "props rune is not exported",
			text: "let { label } = $props();\n",
			want: []syntax.Name{
				{Name: "label", Kind: syntax.NameVariable, Line: 1},
			},
		},
		{
			name: "re-export does not mark locals",
			text: "function a() {}\nexport { a } from './other';\n",
			want: []syntax.Name{
				{Name: "a", Kind: syntax.NameFunction, Line: 1},
			},
		},
		{
			name: "exported destructuring and nested local",
			text: "export const { first, second } = pair;\nfunction f() { const inner = 1; }\n" +
				"export { inner };\n",
			want: []syntax.Name{
				{Name: "first", Kind: syntax.NameConstant, Line: 1, Exported: true},
				{Name: "second", Kind: syntax.NameConstant, Line: 1, Exported: true},
				{Name: "f", Kind: syntax.NameFunction, Line: 2},
				{Name: "inner", Kind: syntax.NameConstant, Line: 2},
			},
		},
	}
}

func TestExportedNames(t *testing.T) {
	t.Parallel()

	testCases := exportNameCases()
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
