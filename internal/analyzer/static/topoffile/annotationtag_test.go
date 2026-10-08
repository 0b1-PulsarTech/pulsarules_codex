package topoffile

import (
	"slices"
	"testing"
)

func TestAnnotationTag_Languages(t *testing.T) {
	t.Parallel()

	runLanguageCases(t, slices.Concat(
		annotationTagCases(), annotationTypedefCases(), annotationImportCases(),
		annotationImportRejectCases(), annotationImportProseCases(),
	))
}

func annotationTagCases() []languageCase {
	return []languageCase{
		clean("satisfies annotation", "a.mjs", "/** @satisfies {X} */\nexport default {}\n"),
		clean(
			"satisfies multi-line",
			"a.js",
			"/**\n * @satisfies {import('x').Y}\n */\nexport default {}\n",
		),
		clean("type with tab separators", "a.js", "/**\t@type\t{X} */\nexport default {}\n"),
		flagged("other tag with braces", "a.js", "/** @returns {X} */\nexport {}\n", 1),
		flagged(
			"satisfies with prose",
			"a.js",
			"/** @satisfies {X} - the app config */\nexport {}\n",
			1,
		),
	}
}

func annotationTypedefCases() []languageCase {
	return []languageCase{
		clean("single-line typedef", "a.js", "/** @typedef {X} Name */\nexport {}\n"),
		clean(
			"typedef with nested braces",
			"a.js",
			"/** @typedef {{a: {b: string}}} Name */\nexport {}\n",
		),
		clean("multi-line typedef", "a.js", "/**\n * @typedef {X} Name\n */\nexport {}\n"),
		flagged("typedef with prose", "a.js", "/** @typedef {X} Name text */\nexport {}\n", 1),
		flagged(
			"typedef with a tab before prose",
			"a.js",
			"/** @typedef {X} Name\ttext */\nexport {}\n",
			1,
		),
		flagged("typedef without name", "a.js", "/** @typedef {X} */\nexport {}\n", 1),
		flagged(
			"multi-line typedef with property",
			"a.js",
			"/**\n * @typedef {Object} Name\n * @property {string} a\n */\nexport {}\n",
			1,
		),
		flagged(
			"type followed by a prose line",
			"a.js",
			"/**\n * @type {X}\n * The app config.\n */\nexport default {}\n",
			1,
		),
		clean(
			"run of annotation blocks",
			"a.js",
			"/** @import {A} from 'a' */\n/** @type {A} */\nexport default {}\n",
		),
		flagged(
			"run ending in prose",
			"a.js",
			"/** @type {A} */\n/** Notes. */\nexport default {}\n",
			2,
		),
		flagged("empty jsdoc", "a.js", "/***/\nexport {}\n", 1),
	}
}

func annotationImportCases() []languageCase {
	return []languageCase{
		clean("import annotation", "a.js", "/** @import {A, B} from './types' */\nexport {}\n"),
		clean("import double quotes", "a.js", "/** @import {A} from \"x\" */\nexport {}\n"),
		clean("import namespace", "a.js", "/** @import * as vite from 'vite' */\nexport {}\n"),
		clean("import default", "a.js", "/** @import Vite from 'vite' */\nexport {}\n"),
		clean("import with tab separators", "a.js", "/** @import\t{A} from 'a' */\nexport {}\n"),
		clean(
			"import block with two imports",
			"a.js",
			"/**\n * @import {A} from 'a'\n * @import {B} from 'b'\n */\nexport {}\n",
		),
	}
}

func annotationImportRejectCases() []languageCase {
	return []languageCase{
		flagged("import without binding", "a.js", "/** @import from 'a' */\nexport {}\n", 1),
		flagged("import with a lone word", "a.js", "/** @import x */\nexport {}\n", 1),
		flagged("import with a star only", "a.js", "/** @import * from 'a' */\nexport {}\n", 1),
		flagged("import with a wrong keyword", "a.js", "/** @import A to 'a' */\nexport {}\n", 1),
		flagged(
			"import braces wrong keyword",
			"a.js",
			"/** @import {A} to 'a' */\nexport {}\n",
			1,
		),
		flagged("import without from", "a.js", "/** @import {A} */\nexport {}\n", 1),
		flagged("import unquoted module", "a.js", "/** @import {A} from xax */\nexport {}\n", 1),
		flagged("import unbalanced quote", "a.js", "/** @import {A} from 'x */\nexport {}\n", 1),
		flagged("import lone quote", "a.js", "/** @import {A} from ' */\nexport {}\n", 1),
		flagged(
			"import mismatched quotes",
			"a.js",
			"/** @import {A} from 'x\" */\nexport {}\n",
			1,
		),
	}
}

func annotationImportProseCases() []languageCase {
	return []languageCase{
		flagged(
			"import with prose after the module",
			"a.js",
			"/** @import {A} from 'a' - shared types */\nexport {}\n",
			1,
		),
		flagged(
			"import with a quoted prose tail",
			"a.js",
			"/** @import {A} from 'a' plus the helpers for 'b' */\nexport {}\n",
			1,
		),
		flagged(
			"import block with a prose line",
			"a.js",
			"/**\n * @import {A} from 'a'\n * Shared router types.\n"+
				" * @import {B} from 'b'\n */\nexport {}\n",
			1,
		),
		flagged(
			"import block opening with prose",
			"a.js",
			"/**\n * Shared types.\n * @import {A} from 'a'\n */\nexport {}\n",
			1,
		),
		flagged(
			"import namespace without as",
			"a.js",
			"/** @import * vite from 'vite' */\nexport {}\n",
			1,
		),
		flagged(
			"import default with extra word",
			"a.js",
			"/** @import A B from 'a' */\nexport {}\n",
			1,
		),
	}
}
