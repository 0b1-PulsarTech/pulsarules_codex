package tsfacts

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

type importCase struct {
	name       string
	text       string
	want       []syntax.Import
	wantGroups []string
}

func importCases() []importCase {
	return []importCase{
		{name: "no imports leaves the groups empty", text: "const a = 1;\n"},
		{
			name: "groups by specifier",
			text: "import fs from \"node:fs\";\nimport x from 'pkg';\nimport y from " +
				"'@scope/pkg/deep';\nimport z from './z';\nimport w from '../w';\nimport v " +
				"from '/abs';\nimport '.';\n",
			want: []syntax.Import{
				{Literal: "\"node:fs\"", Line: 1, Group: 0},
				{Literal: "'pkg'", Line: 2, Group: 1},
				{Literal: "'@scope/pkg/deep'", Line: 3, Group: 1},
				{Literal: "'./z'", Line: 4, Group: 2},
				{Literal: "'../w'", Line: 5, Group: 2},
				{Literal: "'/abs'", Line: 6, Group: 2},
				{Literal: "'.'", Line: 7, Group: 2},
			},
			wantGroups: []string{"node:", "package", "relative"},
		},
		{
			name: "import equals require and type imports",
			text: "\nimport a = require('./a');\nimport type { T } from 'types';\nimport b = " +
				"Ns.b;\n",
			want: []syntax.Import{
				{Literal: "'./a'", Line: 2, Group: 2},
				{Literal: "'types'", Line: 3, Group: 1},
			},
			wantGroups: []string{"node:", "package", "relative"},
		},
		{
			name: "re-exports and dynamic imports are not imports",
			text: "export * from 'a';\nexport { b } from 'b';\nconst c = import('c');\n",
		},
	}
}

func TestImports(t *testing.T) {
	t.Parallel()

	testCases := importCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			facts := collectOf(t, testCase.text, Options{})
			if !slices.Equal(facts.Imports, testCase.want) {
				t.Fatalf("imports\n got %+v\nwant %+v", facts.Imports, testCase.want)
			}
			if !slices.Equal(facts.ImportGroups, testCase.wantGroups) {
				t.Fatalf("groups %v, want %v", facts.ImportGroups, testCase.wantGroups)
			}
		})
	}
}
