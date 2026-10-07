package gosyntax

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

func TestTypeDocSites(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
		want []syntax.DocSite
	}{
		{
			name: "single type names the symbol",
			body: "package x\n\n// Box holds it.\ntype Box struct{}\n",
			want: []syntax.DocSite{{Symbol: "Box", Kind: syntax.DocType, Line: 3}},
		},
		{
			name: "grouped types and the group doc",
			body: "package x\n\n// group.\ntype (\n\t// One is first.\n\tOne int\n\tTwo int\n)\n",
			want: []syntax.DocSite{
				{Symbol: "type group", Kind: syntax.DocType, Line: 3},
				{Symbol: "One", Kind: syntax.DocType, Line: 5},
			},
		},
		{
			name: "interface methods and embeds",
			body: "package x\n\ntype Port interface {\n\t// Run does it.\n\tRun()\n" +
				"\t// Reader is embedded.\n\tio.Reader\n}\n",
			want: []syntax.DocSite{
				{Symbol: "Port.Run", Kind: syntax.DocMethod, Line: 4},
				{Symbol: "Port.embedded interface", Kind: syntax.DocMethod, Line: 6},
			},
		},
		{
			name: "non type spec in a type group",
			body: "package x\n\n// group.\ntype ()\n",
			want: []syntax.DocSite{{Symbol: "type group", Kind: syntax.DocType, Line: 3}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := summarizeSource(t, testCase.body, plainSource(), "").DocSites
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("DocSites = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func TestSpecName(t *testing.T) {
	t.Parallel()

	if got := specName(nil); got != "type declaration" {
		t.Fatalf("specName(nil) = %q", got)
	}
}
