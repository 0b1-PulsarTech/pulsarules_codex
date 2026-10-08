package topoffile

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func TestToolPragma_Languages(t *testing.T) {
	t.Parallel()

	runLanguageCases(t, slices.Concat(
		pragmaExemptCases(), pragmaProseCases(), pragmaPlacementCases(),
	))
}

func pragmaExemptCases() []languageCase {
	return []languageCase{
		clean("flow pragma", "a.js", "// @flow\nconst a = 1\n"),
		clean("flow strict pragma", "a.js", "// @flow strict\nconst a = 1\n"),
		clean("flow strict-local pragma", "a.js", "// @flow strict-local\nconst a = 1\n"),
		clean("noflow pragma", "a.mjs", "// @noflow\nconst a = 1\n"),
		clean("jsx pragma", "a.jsx", "// @jsx h\nconst a = 1\n"),
		clean("jsx frag pragma", "a.jsx", "// @jsxFrag Fragment\nexport {}\n"),
		clean("jsx runtime classic pragma", "a.jsx", "// @jsxRuntime classic\nexport {}\n"),
		clean("jsx runtime automatic pragma", "a.jsx", "// @jsxRuntime automatic\nexport {}\n"),
		clean("jsx import source pragma", "a.tsx", "// @jsxImportSource x\nexport {}\n"),
		clean("refresh pragma", "a.jsx", "// @refresh reset\nexport {}\n"),
		clean("pragma without space", "a.js", "//@flow\nconst a = 1\n"),
		clean("pragma with crlf", "a.js", "// @flow\r\nconst a = 1\r\n"),
		clean("pragma after a bom", "a.js", "\xef\xbb\xbf// @flow\nconst a = 1\n"),
	}
}

func pragmaProseCases() []languageCase {
	return []languageCase{
		flagged(
			"flow with prose",
			"a.js",
			"// @flow Router helpers for the app.\nconst a = 1\n",
			1,
		),
		flagged("flow with unknown mode", "a.js", "// @flow loose\nconst a = 1\n", 1),
		flagged(
			"jsx with prose",
			"a.jsx",
			"// @jsx h Utilities for rendering widgets.\nexport {}\n",
			1,
		),
		flagged("jsx without a factory", "a.jsx", "// @jsx\nexport {}\n", 1),
		flagged(
			"jsx import source with prose",
			"a.tsx",
			"// @jsxImportSource preact renders the dashboard\nexport {}\n",
			1,
		),
		flagged(
			"refresh with prose",
			"a.jsx",
			"// @refresh reset because the store is module level\nexport {}\n",
			1,
		),
		flagged("refresh with unknown mode", "a.jsx", "// @refresh hard\nexport {}\n", 1),
		flagged(
			"noflow with prose",
			"a.js",
			"// @noflow This module is legacy and does X.\nconst a = 1\n",
			1,
		),
		flagged("jsx runtime with unknown mode", "a.jsx", "// @jsxRuntime fancy\nexport {}\n", 1),
		flagged("empty line comment", "a.js", "//\nexport {}\n", 1),
		flagged("unlisted pragma", "a.js", "// @flowery\nconst a = 1\n", 1),
		flagged("pragma word later", "a.js", "// note @flow\nconst a = 1\n", 1),
	}
}

func pragmaPlacementCases() []languageCase {
	return []languageCase{
		flagged("banner after pragma", "a.js", "// @flow\n// note\nconst a = 1\n", 2),
		flagged("banner after two pragmas", "a.js", "// @flow\n// @jsx h\n// note\nconst a\n", 3),
		flagged(
			"block banner after pragma",
			"a.js",
			"// @flow\n/** Utilities. */\nconst a = 1\n",
			2,
		),
		flagged(
			"banner after shebang and pragma",
			"a.mjs",
			"#!/usr/bin/env node\n// @flow\n// CLI for X\nimport a from 'a'\n",
			3,
		),
		{
			name: "pragma in go", file: "a.go", src: "// @flow\npackage a\n",
			want: 1, line: 1, severity: core.SeverityError,
		},
		flagged("pragma in ts", "a.ts", "// @flow\n// note\nconst a = 1\n", 2),
	}
}
