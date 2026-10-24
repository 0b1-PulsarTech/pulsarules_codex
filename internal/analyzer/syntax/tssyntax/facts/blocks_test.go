package facts

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
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

			got := functionsOf(t, testCase.text, Options{IsTest: testCase.isTest}).blocks
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

func TestElses(t *testing.T) {
	t.Parallel()

	runElses(t, []elseCase{
		{name: "no else", text: "if (a) {\n}\n"},
		{
			name: "else after return",
			text: "function f() {\nif (a) {\nreturn 1;\n} else {\nreturn 2;\n}\n}\n",
			want: []syntax.Else{{Line: 4, AfterExit: true}},
		},
		{
			name: "else on its own line is located at the keyword",
			text: "if (a) {\nthrow new Error('x');\n}\nelse\n{\n}\n",
			want: []syntax.Else{{Line: 4, AfterExit: true}},
		},
		{
			name: "else if chain counts each else",
			text: "if (a) {\nf();\n} else if (b) {\ng();\n} else {\nh();\n}\n",
			want: []syntax.Else{{Line: 3}, {Line: 5}},
		},
		{
			name: "break and continue exit without a block",
			text: "for (;;) {\nif (a) break;\nelse f();\nif (b) continue; else g();\n}\n",
			want: []syntax.Else{{Line: 3, AfterExit: true}, {Line: 4, AfterExit: true}},
		},
		{
			name: "a comment after the keyword does not move it",
			text: "if (a) {\n} else /* note\n*/ {\n}\n",
			want: []syntax.Else{{Line: 2}},
		},
		{
			name: "ternary is not an else",
			text: "const x = a ? 1 : 2;\n",
		},
		{
			name: "exit before the last statement is not an exit",
			text: "if (a) {\nreturn 1;\nf();\n} else {\n}\n",
			want: []syntax.Else{{Line: 4}},
		},
	})
}

func TestElsesChains(t *testing.T) {
	t.Parallel()

	runElses(t, []elseCase{
		{
			name: "an earlier branch that continues on keeps the else",
			text: "if (a) f();\nelse if (b) return;\nelse g();\n",
			want: []syntax.Else{{Line: 2}, {Line: 3}},
		},
		{
			name: "every branch exiting marks every else",
			text: "if (a) return;\nelse if (b) return;\nelse g();\n",
			want: []syntax.Else{{Line: 2, AfterExit: true}, {Line: 3, AfterExit: true}},
		},
		{
			name: "init then done then advance",
			text: "if (s == null) {\ninit();\n} else if (s.done) {\nreturn;\n} else {\nadvance();\n}\n",
			want: []syntax.Else{{Line: 3}, {Line: 5}},
		},
		{
			name: "break to the label on the if itself is no exit",
			text: "mark: if (a) {\nbreak mark;\n} else {\nf();\n}\n",
			want: []syntax.Else{{Line: 3}},
		},
		{
			name: "break to an outer loop label is an exit",
			text: "outer: for (;;) {\nif (a) {\nbreak outer;\n} else {\nf();\n}\n}\n",
			want: []syntax.Else{{Line: 4, AfterExit: true}},
		},
		{
			name: "empty then block is no exit",
			text: "if (a) {} else {\nf();\n}\n",
			want: []syntax.Else{{Line: 1}},
		},
	})
}

type elseCase struct {
	name string
	text string
	want []syntax.Else
}

func runElses(t *testing.T, testCases []elseCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := collectOf(t, testCase.text, Options{}).Elses
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("elses %v, want %v", got, testCase.want)
			}
		})
	}
}
