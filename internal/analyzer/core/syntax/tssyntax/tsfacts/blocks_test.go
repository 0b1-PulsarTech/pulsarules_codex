package tsfacts

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

type blockCase struct {
	name   string
	text   string
	isTest bool
	want   []syntax.Block
}

func runBlocks(t *testing.T, testCases []blockCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := collectOf(t, testCase.text, Options{IsTest: testCase.isTest}).Blocks
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("blocks %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestBlocksStatements(t *testing.T) {
	t.Parallel()

	runBlocks(t, []blockCase{
		{name: "body block itself is omitted", text: "function f() {\n}\n"},
		{
			name: "if else chain keeps one depth",
			text: "function f() {\nif (a) {\n} else if (b) {\n} else {\n}\n}\n",
			want: []syntax.Block{{Line: 2, Depth: 1}, {Line: 3, Depth: 1}, {Line: 4, Depth: 1}},
		},
		{
			name: "loops nest",
			text: "function f() {\nfor (;;) {\nwhile (a) {\ndo {\n} while (b);\n}\n}\n}\n",
			want: []syntax.Block{{Line: 2, Depth: 1}, {Line: 3, Depth: 2}, {Line: 4, Depth: 3}},
		},
		{
			name: "single statement body deepens its children",
			text: "function f() {\nif (a) for (;;) {\n}\n}\n",
			want: []syntax.Block{{Line: 2, Depth: 2}},
		},
		{
			name: "switch counts the case block and its clauses",
			text: "function f() {\nswitch (a) {\ncase 1: {\nbreak;\n}\ndefault:\nif (b) {\n}\n" +
				"}\n}\n",
			want: []syntax.Block{{Line: 2, Depth: 1}, {Line: 3, Depth: 2}, {Line: 7, Depth: 2}},
		},
		{
			name: "try catch finally",
			text: "function f() {\ntry {\n} catch (e) {\n} finally {\n}\n}\n",
			want: []syntax.Block{{Line: 2, Depth: 1}, {Line: 3, Depth: 1}, {Line: 4, Depth: 1}},
		},
		{
			name: "bare and labelled blocks",
			text: "function f() {\n{\n}\nouter: for (;;) {\n}\n}\n",
			want: []syntax.Block{{Line: 2, Depth: 1}, {Line: 4, Depth: 1}},
		},
	})
}

func TestBlocksFunctions(t *testing.T) {
	t.Parallel()

	runBlocks(t, []blockCase{
		{
			name: "expression body has no blocks",
			text: "const f = () => a ? { b: 1 } : 2;\n",
		},
		{
			name: "anonymous closure contributes none",
			text: "function f() {\nlist.map((x) => {\nif (x) {\n}\n});\n}\n",
		},
		{
			name: "nested declaration restarts at depth zero",
			text: "function f() {\nfunction g() {\nif (a) {\n}\n}\n}\n",
			want: []syntax.Block{{Line: 3, Depth: 1}},
		},
		{
			name: "bound arrow contributes",
			text: "const f = () => {\nif (a) {\n}\n};\n",
			want: []syntax.Block{{Line: 2, Depth: 1}},
		},
		{
			name:   "test callback contributes",
			text:   "describe('x', () => {\nif (a) {\n}\n});\n",
			isTest: true,
			want:   []syntax.Block{{Line: 2, Depth: 1}},
		},
		{
			name: "top level callback outside a test file contributes",
			text: "describe('x', () => {\nif (a) {\n}\n});\n",
			want: []syntax.Block{{Line: 2, Depth: 1}},
		},
		{
			name: "nested callback outside a test file does not",
			text: "function f() {\ndescribe('x', () => {\nif (a) {\n}\n});\n}\n",
		},
	})
}
