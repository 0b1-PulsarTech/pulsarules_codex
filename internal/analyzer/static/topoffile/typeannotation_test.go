package topoffile

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func TestTypeAnnotation_Languages(t *testing.T) {
	t.Parallel()

	runLanguageCases(t, slices.Concat(
		annotationShapeCases(), annotationLayoutCases(), annotationBannerCases(),
		annotationDocblockCases(), annotationExtensionCases(), annotationDirectiveCases(),
	))
}

func annotationShapeCases() []languageCase {
	return []languageCase{
		clean(
			"type annotation before export",
			"vite.config.js",
			"/** @type {import('vite').UserConfig} */\nexport default {}\n",
		),
		clean("type annotation crlf", "a.js", "/** @type {X} */\r\nexport default {}\r\n"),
		clean(
			"multi-line type annotation",
			"a.js",
			"/**\n * @type {{\n *   a: string\n * }}\n */\nexport default {}\n",
		),
		clean(
			"type annotation after ts-check",
			"a.js",
			"// @ts-check\n/** @type {X} */\nexport default {}\n",
		),
		flagged("type annotation at eof", "a.js", "/** @type {X} */", 1),
		flagged("type annotation at eof newline", "a.js", "/** @type {X} */\n", 1),
		flagged("prose containing type", "a.js", "/** Config. @type {X} */\nexport {}\n", 1),
		flagged("type with trailing prose", "a.js", "/** @type {X} the config */\nexport {}\n", 1),
		flagged("type without braces", "a.js", "/** @type X */\nexport default {}\n", 1),
		flagged("type name before braces", "a.js", "/** @type Config {X} */\nexport {}\n", 1),
		flagged("unclosed type braces", "a.js", "/** @type {X */\nexport default {}\n", 1),
		flagged("unterminated block", "a.js", "/** @type {X}\nexport default {}\n", 1),
		flagged("real jsdoc banner", "a.js", "/** Utilities for X. */\nexport const a = 1\n", 1),
		flagged("plain block with type", "a.js", "/* @type {X} */\nexport default {}\n", 1),
		flagged("tag alone", "a.js", "/** @type */\nexport default {}\n", 1),
	}
}

func annotationLayoutCases() []languageCase {
	return []languageCase{
		clean("blank before code", "a.js", "/** @type {X} */\n\nexport default {}\n"),
		clean("blank between blocks", "a.js", "/** @type {X} */\n\n/** @type {Y} */\nexport {}\n"),
		clean("same-line code", "a.js", "/** @type {X} */ export default {}\n"),
		flagged("same-line comment", "a.js", "/** @type {X} */ // note\nexport {}\n", 1),
		flagged("comment after annotation", "a.js", "/** @type {X} */\n// note\nexport {}\n", 2),
		flagged("block after annotation", "a.js", "/** @type {X} */\n/* note */\nexport {}\n", 2),
		flagged("blank then comment", "a.js", "/** @type {X} */\n\n// note\nexport {}\n", 3),
		flagged(
			"plain multi-line block with type",
			"a.js",
			"/*\n * @type {X}\n */\nexport default {}\n",
			1,
		),
	}
}

func annotationExtensionCases() []languageCase {
	const config = "/** @type {X} */\nexport default {}\n"
	return []languageCase{
		clean("annotation in js", "a.js", config),
		clean("annotation in mjs", "a.mjs", config),
		clean("annotation in cjs", "a.cjs", config),
		clean("annotation in jsx", "a.jsx", config),
		flagged("annotation in ts", "a.ts", config, 1),
		flagged("annotation in tsx", "a.tsx", config, 1),
		{
			name: "go ignores annotation", file: "a.go",
			src: "/** @type {X} */\npackage a\n", want: 1, line: 1, severity: core.SeverityError,
		},
	}
}

func annotationDirectiveCases() []languageCase {
	return []languageCase{
		clean(
			"eslint next-line before code",
			"a.js",
			"/** @type {X} */\n// eslint-disable-next-line no-undef\nexport default {}\n",
		),
		clean(
			"ts-expect-error before code",
			"a.js",
			"/** @type {X} */\n// @ts-expect-error\nexport default {}\n",
		),
		clean(
			"directive between blocks",
			"a.js",
			"/** @type {X} */\n// @ts-ignore\n/** @type {Y} */\nexport {}\n",
		),
		flagged(
			"comment after directive",
			"a.js",
			"/** @type {X} */\n// @ts-ignore\n// note\nexport {}\n",
			3,
		),
		clean(
			"multi-line directive block",
			"a.js",
			"/** @type {X} */\n/* eslint-disable\n   no-console */\nexport default {}\n",
		),
		flagged(
			"banner after multi-line directive block",
			"a.js",
			"/** @type {X} */\n/* eslint-disable\n */\n// Router utilities.\nexport default {}\n",
			4,
		),
		flagged(
			"banner after directive and import",
			"a.js",
			"/** @import {A} from 'a' */\n// eslint-disable-next-line\n"+
				"// Router utilities.\nexport {}\n",
			3,
		),
	}
}

func annotationDocblockCases() []languageCase {
	return []languageCase{
		clean(
			"docblock pragma after annotation",
			"a.jsx",
			"/** @type {X} */\n/** @jsx h */\nexport {}\n",
		),
		flagged(
			"docblock pragma with prose after annotation",
			"a.jsx",
			"/** @type {X} */\n/**\n * @jsx h\n * Renders the page.\n */\nexport {}\n",
			2,
		),
		flagged("directive at eof", "a.js", "/** @type {X} */\n// @ts-ignore\n", 1),
	}
}

func annotationBannerCases() []languageCase {
	return []languageCase{
		flagged(
			"banner block after import",
			"a.js",
			"/** @import {A} from 'a' */\n/** Router utilities. */\nexport {}\n",
			2,
		),
		flagged(
			"banner line after import",
			"a.js",
			"/** @import {A} from 'a' */\n// Router utilities.\nexport {}\n",
			2,
		),
		flagged(
			"banner on the closing line of a block",
			"a.js",
			"/**\n * @type {X}\n */ // note\nexport {}\n",
			3,
		),
		flagged(
			"banner after the annotation on one line",
			"a.js",
			"/** @type {X} */ /** B. */\nx\n",
			1,
		),
	}
}

func flagged(name, file, src string, line int) languageCase {
	return languageCase{
		name: name, file: file, src: src, want: 1, line: line, severity: core.SeverityWarning,
	}
}

func clean(name, file, src string) languageCase {
	return languageCase{name: name, file: file, src: src}
}
