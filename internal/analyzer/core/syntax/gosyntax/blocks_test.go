package gosyntax

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

type blockCase struct {
	name string
	body string
	want []syntax.Block
}

func blockCases() []blockCase {
	return []blockCase{
		{
			name: "loops and branches deepen",
			body: "package x\n\nfunc run(ok bool) {\n" +
				"\tif ok {\n\t\tfor {\n\t\t}\n\t}\n}\n",
			want: []syntax.Block{{Line: 3, Depth: 0}, {Line: 4, Depth: 1}, {Line: 5, Depth: 2}},
		},
		{
			name: "else if chains stay level and else deepens",
			body: "package x\n\nfunc run(ok bool) {\n" +
				"\tif ok {\n\t} else if !ok {\n\t} else {\n\t}\n}\n",
			want: []syntax.Block{
				{Line: 3, Depth: 0}, {Line: 4, Depth: 1}, {Line: 5, Depth: 1}, {Line: 6, Depth: 1},
			},
		},
		{
			name: "case bodies are not walked",
			body: "package x\n\nfunc run(ok bool) {\n" +
				"\tswitch {\n\tcase ok:\n\t\tif ok {\n\t\t}\n\t}\n}\n",
			want: []syntax.Block{{Line: 3, Depth: 0}, {Line: 4, Depth: 1}},
		},
		{
			name: "range select typeswitch and bare block",
			body: "package x\n\nfunc run(val any) {\n\tfor range 3 {\n\t}\n" +
				"\tswitch val.(type) {\n\t}\n\tselect {\n\t}\n\t{\n\t}\n}\n",
			want: []syntax.Block{
				{Line: 3, Depth: 0},
				{Line: 4, Depth: 1},
				{Line: 6, Depth: 1},
				{Line: 8, Depth: 1},
				{Line: 10, Depth: 1},
			},
		},
		{name: "bodiless declaration yields nothing", body: "package x\n\nfunc run()\n"},
	}
}

func TestBlocks(t *testing.T) {
	t.Parallel()

	for _, testCase := range blockCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := summarizeSource(t, testCase.body, plainSource(), "").Blocks
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("Blocks = %+v, want %+v", got, testCase.want)
			}
		})
	}
}
