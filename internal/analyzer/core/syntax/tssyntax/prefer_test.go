package tssyntax

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/tssyntax/tsalias"
)

type preferCase struct {
	name        string
	path        string
	root        string
	aliases     []tsalias.Alias
	literals    []string
	wantPrefer  []string
	wantUpward  []bool
	wantForeign []bool
}

func extraPreferCases() []preferCase {
	kit := tsalias.Alias{
		Spec: "$lib/*", Target: "web/src/lib/*", Root: "web", Origin: tsalias.OriginKit,
	}
	paths := tsalias.Alias{
		Spec: "@/*", Target: "web/src/*", Origin: tsalias.OriginPaths,
	}
	return []preferCase{
		{
			name:        "outside the package root is foreign",
			path:        "pkg/src/a.ts",
			root:        "pkg",
			literals:    []string{"'../../x'", "'../b'", "'../..'"},
			wantPrefer:  []string{"", "", ""},
			wantUpward:  []bool{false, true, false},
			wantForeign: []bool{true, false, true},
		},
		{
			name:       "kit alias is not offered outside src",
			path:       "web/scripts/seed.mjs",
			aliases:    []tsalias.Alias{kit},
			literals:   []string{"'../src/lib/db.js'"},
			wantPrefer: []string{""},
			wantUpward: []bool{true},
		},
		{
			name:       "kit alias is offered inside src",
			path:       "web/src/routes/page.js",
			aliases:    []tsalias.Alias{kit},
			literals:   []string{"'../lib/db.js'"},
			wantPrefer: []string{"$lib/db.js"},
			wantUpward: []bool{false},
		},
		{
			name:       "tsconfig paths are not offered to node run scripts",
			path:       "web/src/a/b.mjs",
			aliases:    []tsalias.Alias{paths},
			literals:   []string{"'../c.js'"},
			wantPrefer: []string{""},
			wantUpward: []bool{true},
		},
		{
			name:       "tsconfig paths are offered to typescript",
			path:       "web/src/a/b.ts",
			aliases:    []tsalias.Alias{paths},
			literals:   []string{"'../c.js'"},
			wantPrefer: []string{"@/c.js"},
			wantUpward: []bool{false},
		},
	}
}

func resolutionPreferCases() []preferCase {
	aliases := []tsalias.Alias{
		{Spec: "@/*", Target: "web/src/*"},
		{Spec: "@/ui/*", Target: "web/other/*"},
		{Spec: "virtual:r", Target: "web/src/r.d.ts"},
		{Spec: "@env", Target: "web/src/env.ts"},
		{Spec: "react", Target: "web/node_modules/@types/react"},
		{Spec: "#x/*", Target: "web/src/ui/*.d.ts"},
	}
	return []preferCase{
		{
			name:       "longest alias prefix decides",
			path:       "web/src/ui/a.ts",
			aliases:    aliases,
			literals:   []string{"'@/ui/b'", "'@/b'"},
			wantPrefer: []string{"", ""},
			wantUpward: []bool{false, false},
		},
		{
			name:       "no spelling that does not resolve to the same file",
			path:       "web/src/sub/a.ts",
			aliases:    aliases,
			literals:   []string{"'virtual:r'", "'@env'", "'react'", "'../r.d.ts'"},
			wantPrefer: []string{"", "", "", ""},
			wantUpward: []bool{false, false, false, true},
		},
		{
			name:       "no neighbour spelling for a declaration or a bare ts target",
			path:       "web/src/a.ts",
			aliases:    aliases,
			literals:   []string{"'virtual:r'", "'@env'"},
			wantPrefer: []string{"", ""},
			wantUpward: []bool{false, false},
		},
		{
			name:       "no neighbour spelling inside a dependency directory",
			path:       "web/vendor/node_modules/a.ts",
			aliases:    []tsalias.Alias{{Spec: "dep", Target: "web/vendor/node_modules/dep.js"}},
			literals:   []string{"'dep'"},
			wantPrefer: []string{""},
			wantUpward: []bool{false},
		},
		{
			name:       "an extension written out is a neighbour",
			path:       "web/src/a.ts",
			aliases:    aliases,
			literals:   []string{"'@/env.ts'", "'#x/y'"},
			wantPrefer: []string{"./env.ts", ""},
			wantUpward: []bool{false, false},
		},
	}
}

func preferCases() []preferCase {
	aliases := []tsalias.Alias{
		{Spec: "#lib/*", Target: "web/src/lib/*"},
		{Spec: "#deep/*", Target: "web/src/lib/deep/*"},
		{Spec: "$cfg", Target: "web/config"},
	}
	return []preferCase{
		{
			name:       "alias into the own directory",
			path:       "web/src/lib/a.ts",
			aliases:    aliases,
			literals:   []string{"'#lib/b'", `"#lib/c"`},
			wantPrefer: []string{"./b", "./c"},
			wantUpward: []bool{false, false},
		},
		{
			name:       "alias into a sub directory is not a neighbour",
			path:       "web/src/lib/a.ts",
			aliases:    aliases,
			literals:   []string{"'#lib/sub/b'", "'#deep/x'"},
			wantPrefer: []string{"", ""},
			wantUpward: []bool{false, false},
		},
		{
			name:       "neighbour imports are fine",
			path:       "web/src/lib/a.ts",
			aliases:    aliases,
			literals:   []string{"'./b'", "'./sub/c'", "'.'", "'./'"},
			wantPrefer: []string{"", "", "", ""},
			wantUpward: []bool{false, false, false, false},
		},
		{
			name:       "parent import covered by the deepest alias",
			path:       "web/src/lib/deep/a.ts",
			aliases:    aliases,
			literals:   []string{"'../x'", "'../../routes/y'"},
			wantPrefer: []string{"#lib/x", ""},
			wantUpward: []bool{false, true},
		},
		{
			name:       "parent import covered by an exact alias",
			path:       "web/config/sub/a.ts",
			aliases:    aliases,
			literals:   []string{"'..'", "'../x'"},
			wantPrefer: []string{"$cfg", ""},
			wantUpward: []bool{false, true},
		},
		{
			name:       "parent import with no alias is upward",
			path:       "pkg/src/a.ts",
			literals:   []string{"'../b'", "'../../c'", "'..'"},
			wantPrefer: []string{"", "", ""},
			wantUpward: []bool{true, true, true},
		},
		{
			name:       "root level file",
			path:       "a.ts",
			literals:   []string{"'../b'", "'./b'"},
			wantPrefer: []string{"", ""},
			wantUpward: []bool{true, false},
		},
		{
			name:       "non strings and bare packages are skipped",
			path:       "web/src/lib/a.ts",
			aliases:    aliases,
			literals:   []string{"pkg", "'", "'pkg'", "`../x`", "'/abs'"},
			wantPrefer: []string{"", "", "", "", ""},
			wantUpward: []bool{false, false, false, false, false},
		},
	}
}

func TestApplyPrefer(t *testing.T) {
	t.Parallel()

	testCases := append(preferCases(), extraPreferCases()...)
	testCases = append(testCases, resolutionPreferCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file := &syntax.File{Path: testCase.path}
			for _, literal := range testCase.literals {
				file.Imports = append(file.Imports, syntax.Import{Literal: literal})
			}
			applyPrefer(file, importer{root: testCase.root, aliases: testCase.aliases})
			prefer := make([]string, 0, len(file.Imports))
			upward := make([]bool, 0, len(file.Imports))
			foreign := make([]bool, 0, len(file.Imports))
			for _, imported := range file.Imports {
				prefer = append(prefer, imported.Prefer)
				upward = append(upward, imported.Upward)
				foreign = append(foreign, imported.Foreign)
			}
			wantForeign := testCase.wantForeign
			if wantForeign == nil {
				wantForeign = make([]bool, len(file.Imports))
			}
			if !slices.Equal(prefer, testCase.wantPrefer) ||
				!slices.Equal(upward, testCase.wantUpward) ||
				!slices.Equal(foreign, wantForeign) {
				t.Fatalf("prefer %q upward %v foreign %v\nwant %q %v %v",
					prefer, upward, foreign, testCase.wantPrefer, testCase.wantUpward, wantForeign)
			}
		})
	}
}
