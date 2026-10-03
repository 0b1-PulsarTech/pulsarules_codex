package historynarration

import (
	"slices"
	"testing"
)

func TestQuoteOpenersAndEscapes(t *testing.T) {
	t.Parallel()

	for _, testCase := range slices.Concat(
		quoteOpenerCases(), scalarCases(), tomlCases(), rustLiteralCases(),
	) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			runScan(t, testCase)
		})
	}
}

func quoteOpenerCases() []scanCase {
	return []scanCase{
		{"yaml sequence opener", "hash.yaml", "k: ['a # the old code']\n", 0},
		{"yaml mapping opener", "hash.yaml", "k: {'a # the old code': 1}\n", 0},
		{"yaml double quote escape", "hash.yaml", "k: \"a\\\" # the old code\"\n", 0},
		{"shell double quote escape", "hash.sh", "echo \"a\\\" # the old code\"\n", 0},
		{"cpp wide prefix", "clike.cpp", "x = L'a // the old code';\n", 0},
		{"js yield prefix", "clike", "x = yield'a // the old code';\n", 0},
		{"js double quote after an identifier", "clike.js", "x = tag\"a // the old code\";\n", 0},
		{"unclosed double quote hides the rest", "hash.py", "x = \"open # the old code\n", 0},
		{
			"python raw prefix still escapes a quote",
			"hash.py",
			"x = r\"a\\\" # the old code\"\n",
			0,
		},
	}
}

func scalarCases() []scanCase {
	return slices.Concat(scalarBasicCases(), scalarExtendedCases())
}

func scalarBasicCases() []scanCase {
	return []scanCase{
		{
			"hash glued to a word is not a comment",
			"hash.yaml",
			"url: http://h/p#the-old-code\n",
			0,
		},
		{
			"yaml apostrophe in a value",
			"hash.yaml",
			"description: Bob's retry # it's what we used to call backoff\n",
			1,
		},
		{
			"yaml single apostrophe",
			"hash.yaml",
			"name: Bob's # the old code\n",
			1,
		},
		{
			"yaml prefix-like letter before the apostrophe",
			"hash.yaml",
			"brand: L'Oreal # it's the old code\n",
			1,
		},
		{
			"yaml e before the apostrophe",
			"hash.yaml",
			"name: E'lise # it's the old code\n",
			1,
		},
		{
			"yaml name with an apostrophe",
			"hash.yml",
			"name: O'Brien # it's the old code\n",
			1,
		},
		{
			"yaml quoted value hides the hash",
			"hash.yaml",
			"name: 'a # the old code'\n",
			0,
		},
	}
}

func scalarExtendedCases() []scanCase {
	return []scanCase{
		{
			"yaml quoted value at line start",
			"hash.yaml",
			"'a # the old code': x\n",
			0,
		},
		{
			"yaml list item string",
			"hash.yaml",
			"- 'a # the old code'\n",
			0,
		},
		{
			"yaml flow sequence string",
			"hash.yaml",
			"k: ['a', 'b # the old code']\n",
			0,
		},
		{
			"yaml flow map string",
			"hash.yaml",
			"k: {a: 'b # the old code'}\n",
			0,
		},
		{
			"yaml double quoted mid-scalar is prose",
			"hash.yaml",
			"say: he said \"hi # the old code\"\n",
			1,
		},
		{"yaml flow sequence string", "hash.yaml", "k: ['a\\', ' # the old code']\n", 0},
		{"yaml flow mapping string", "hash.yaml", "k: {a: 'x # the old code'}\n", 0},
	}
}

func tomlCases() []scanCase {
	return []scanCase{
		{
			"toml string after equals",
			"hash.toml",
			"city = \"x\" # N'Djamena, it's the old code\n",
			1,
		},
		{
			"toml literal string after equals",
			"hash.toml",
			"a = 'x # the old code'\n",
			0,
		},
		{
			"toml single quote value then comment",
			"hash.toml",
			"a = 'x' # it's the old code\n",
			1,
		},
	}
}

func rustLiteralCases() []scanCase {
	return []scanCase{
		{
			"rust char literal hides slashes",
			"clike.rs",
			"let c = 'x'; let d = '/'; // the old code\n",
			1,
		},
		{
			"rust char literal holds a double quote",
			"clike.rs",
			"let c = '\"'; // the old code\n",
			1,
		},
		{
			"rust escaped char literal",
			"clike.rs",
			"let c = '\\''; // the old code\n",
			1,
		},
		{
			"rust byte char literal",
			"clike.rs",
			"let c = b'\\''; // the old code\n",
			1,
		},
		{
			"rust unicode escape char",
			"clike.rs",
			"let c = '\\u{1F600}'; // the old code\n",
			1,
		},
		{
			"rust non-ascii char literal",
			"clike.rs",
			"let c = '\u00e9'; // the old code\n",
			1,
		},
		{
			"rust string hides slashes",
			"clike.rs",
			"let s = \"a // the old code\";\n",
			0,
		},
		{
			"rust raw string hides slashes",
			"clike.rs",
			"let s = r\"a // the old code\";\n",
			0,
		},
		{
			"rust hashed raw string with an inner quote",
			"clike.rs",
			"let s = r#\"a\" // the old code\"#;\n",
			0,
		},
		{
			"rust hashed raw string then a comment",
			"clike.rs",
			"let s = r#\"a\"#; // the old code\n",
			1,
		},
	}
}
