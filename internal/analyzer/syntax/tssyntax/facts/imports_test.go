package facts

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

type importCase struct {
	name       string
	text       string
	local      []string
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
			wantGroups: []string{"node:", "external", "local"},
		},
		{
			name: "builtins aliases and virtual modules",
			text: "import a from 'bun:test';\nimport b from 'fs';\nimport c from '$app/x';\n" +
				"import d from 'virtual:y';\nimport e from '#lib/e';\nimport f from '$lib';\n" +
				"import g from '$lib/g';\nimport h from '@/h';\nimport i from '~/i';\n" +
				"import j from '@scope/pkg';\nimport k from '$libx';\n",
			want: []syntax.Import{
				{Literal: "'bun:test'", Line: 1, Group: 0},
				{Literal: "'fs'", Line: 2, Group: 1},
				{Literal: "'$app/x'", Line: 3, Group: 1},
				{Literal: "'virtual:y'", Line: 4, Group: 1},
				{Literal: "'#lib/e'", Line: 5, Group: 2},
				{Literal: "'$lib'", Line: 6, Group: 2},
				{Literal: "'$lib/g'", Line: 7, Group: 2},
				{Literal: "'@/h'", Line: 8, Group: 2},
				{Literal: "'~/i'", Line: 9, Group: 2},
				{Literal: "'@scope/pkg'", Line: 10, Group: 1},
				{Literal: "'$libx'", Line: 11, Group: 1},
			},
			wantGroups: []string{"node:", "external", "local"},
		},
		{
			name: "discovered aliases are local",
			text: "import a from 'app/x';\nimport b from 'app';\nimport c from 'other/x';\n" +
				"import d from '$app/state';\n",
			local: []string{"app/*", "$app/*"},
			want: []syntax.Import{
				{Literal: "'app/x'", Line: 1, Group: 2},
				{Literal: "'app'", Line: 2, Group: 1},
				{Literal: "'other/x'", Line: 3, Group: 1},
				{Literal: "'$app/state'", Line: 4, Group: 2},
			},
			wantGroups: []string{"node:", "external", "local"},
		},
		{
			name: "import equals require and type imports",
			text: "\nimport a = require('./a');\nimport type { T } from 'types';\nimport b = " +
				"Ns.b;\n",
			want: []syntax.Import{
				{Literal: "'./a'", Line: 2, Group: 2},
				{Literal: "'types'", Line: 3, Group: 1},
			},
			wantGroups: []string{"node:", "external", "local"},
		},
		{
			name: "re-exports and dynamic imports are not imports",
			text: "export * from 'a';\nexport { b } from 'b';\nconst c = import('c');\n",
		},
	}
}

func TestImportFactsAlone(t *testing.T) {
	t.Parallel()

	testCases := importCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := importsOf(
				t,
				testCase.text,
				testCase.local,
			); !slices.Equal(
				got,
				testCase.want,
			) {
				t.Fatalf("imports\n got %+v\nwant %+v", got, testCase.want)
			}
		})
	}
}

func TestImports(t *testing.T) {
	t.Parallel()

	testCases := importCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file := collectOf(t, testCase.text, Options{Local: testCase.local})
			if !slices.Equal(file.Imports, testCase.want) {
				t.Fatalf("imports\n got %+v\nwant %+v", file.Imports, testCase.want)
			}
			if !slices.Equal(file.ImportGroups, testCase.wantGroups) {
				t.Fatalf("groups %v, want %v", file.ImportGroups, testCase.wantGroups)
			}
		})
	}
}
