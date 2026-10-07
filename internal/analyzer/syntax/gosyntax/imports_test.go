package gosyntax

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func TestImports(t *testing.T) {
	t.Parallel()

	const body = "package x\n\nimport (\n\t\"fmt\"\n\tal \"github.com/a/b\"\n" +
		"\t\"example.com/app/sub\"\n\t\"example.com/app\"\n)\n"
	testCases := []struct {
		name       string
		body       string
		module     string
		wantGroups []string
		want       []syntax.Import
	}{
		{
			name: "three groups", body: body, module: "example.com/app",
			wantGroups: []string{"stdlib", "external", "example.com/app"},
			want: []syntax.Import{
				{Literal: `"fmt"`, Line: 4, Group: 0},
				{Literal: `"github.com/a/b"`, Line: 5, Group: 1},
				{Literal: `"example.com/app/sub"`, Line: 6, Group: 2},
				{Literal: `"example.com/app"`, Line: 7, Group: 2},
			},
		},
		{name: "no module path leaves it empty", body: body},
		{name: "no imports", body: "package x\n", module: "example.com/app"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := summarizeSource(t, testCase.body, plainSource(), testCase.module)
			if !slices.Equal(got.Imports, testCase.want) ||
				!slices.Equal(got.ImportGroups, testCase.wantGroups) {
				t.Fatalf("Imports = %+v groups %v", got.Imports, got.ImportGroups)
			}
		})
	}
}
