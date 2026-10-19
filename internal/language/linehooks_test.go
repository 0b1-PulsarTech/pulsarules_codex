package language

import (
	"slices"
	"testing"
)

func TestScanComments_QuoteOpenersAndEscapes(t *testing.T) {
	t.Parallel()

	for _, testCase := range slices.Concat(
		quoteOpenerCases(), scalarCases(), tomlCases(), rustLiteralCases(),
	) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			runComments(t, testCase)
		})
	}
}

func quoteOpenerCases() []scanCase {
	return []scanCase{
		{"yaml sequence opener", "yaml", "k: ['a # the old code']\n", 0},
		{"yaml mapping opener", "yaml", "k: {'a # the old code': 1}\n", 0},
		{"yaml double quote escape", "yaml", "k: \"a\\\" # the old code\"\n", 0},
		{"shell double quote escape", "sh", "echo \"a\\\" # the old code\"\n", 0},
		{"cpp wide prefix", "cpp", "x = L'a // the old code';\n", 0},
		{"js yield prefix", "ts", "x = yield'a // the old code';\n", 0},
		{"js double quote after an identifier", "js", "x = tag\"a // the old code\";\n", 0},
		{"unclosed double quote hides the rest", "py", "x = \"open # the old code\n", 0},
		{
			"python raw prefix still escapes a quote",
			"py",
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
			"yaml",
			"url: http://h/p#the-old-code\n",
			0,
		},
		{
			"yaml apostrophe in a value",
			"yaml",
			"description: Bob's retry # it's what we used to call backoff\n",
			1,
		},
		{
			"yaml single apostrophe",
			"yaml",
			"name: Bob's # the old code\n",
			1,
		},
		{
			"yaml prefix-like letter before the apostrophe",
			"yaml",
			"brand: L'Oreal # it's the old code\n",
			1,
		},
		{
			"yaml e before the apostrophe",
			"yaml",
			"name: E'lise # it's the old code\n",
			1,
		},
		{
			"yaml name with an apostrophe",
			"yml",
			"name: O'Brien # it's the old code\n",
			1,
		},
		{
			"yaml quoted value hides the hash",
			"yaml",
			"name: 'a # the old code'\n",
			0,
		},
	}
}

func scalarExtendedCases() []scanCase {
	return []scanCase{
		{
			"yaml quoted value at line start",
			"yaml",
			"'a # the old code': x\n",
			0,
		},
		{
			"yaml list item string",
			"yaml",
			"- 'a # the old code'\n",
			0,
		},
		{
			"yaml flow sequence string",
			"yaml",
			"k: ['a', 'b # the old code']\n",
			0,
		},
		{
			"yaml flow map string",
			"yaml",
			"k: {a: 'b # the old code'}\n",
			0,
		},
		{
			"yaml double quoted mid-scalar is prose",
			"yaml",
			"say: he said \"hi # the old code\"\n",
			1,
		},
		{"yaml flow sequence string", "yaml", "k: ['a\\', ' # the old code']\n", 0},
		{"yaml flow mapping string", "yaml", "k: {a: 'x # the old code'}\n", 0},
	}
}

func tomlCases() []scanCase {
	return []scanCase{
		{
			"toml string after equals",
			"toml",
			"city = \"x\" # N'Djamena, it's the old code\n",
			1,
		},
		{
			"toml literal string after equals",
			"toml",
			"a = 'x # the old code'\n",
			0,
		},
		{
			"toml literal string ends at a backslash",
			"toml",
			"a = 'x\\' # it's the old code\n",
			1,
		},
		{
			"toml single quote value then comment",
			"toml",
			"a = 'x' # it's the old code\n",
			1,
		},
	}
}

func rustLiteralCases() []scanCase {
	return []scanCase{
		{
			"rust char literal hides slashes",
			"rs",
			"let c = 'x'; let d = '/'; // the old code\n",
			1,
		},
		{
			"rust char literal holds a double quote",
			"rs",
			"let c = '\"'; // the old code\n",
			1,
		},
		{
			"rust escaped char literal",
			"rs",
			"let c = '\\''; // the old code\n",
			1,
		},
		{
			"rust byte char literal",
			"rs",
			"let c = b'\\''; // the old code\n",
			1,
		},
		{
			"rust unicode escape char",
			"rs",
			"let c = '\\u{1F600}'; // the old code\n",
			1,
		},
		{
			"rust non-ascii char literal",
			"rs",
			"let c = '\u00e9'; // the old code\n",
			1,
		},
		{
			"rust string hides slashes",
			"rs",
			"let s = \"a // the old code\";\n",
			0,
		},
		{
			"rust raw string hides slashes",
			"rs",
			"let s = r\"a // the old code\";\n",
			0,
		},
		{
			"rust hashed raw string with an inner quote",
			"rs",
			"let s = r#\"a\" // the old code\"#;\n",
			0,
		},
		{"rust block comments nest", "rs", "/* a /* b */ the old code */\nlet x = 1;\n", 1},
		{
			"rust hashed raw string then a comment",
			"rs",
			"let s = r#\"a\"#; // the old code\n",
			1,
		},
	}
}

func TestLineHooks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		hook func(src string, at int) bool
		src  string
		at   int
		want bool
	}{
		{"no identifier before a quote", noIdentBefore, "x = 'a'", 4, true},
		{"identifier glued before a quote", noIdentBefore, "x = f'a'", 5, false},
		{"keyword before a quote", wordOrNone, "return'a'", 6, true},
		{"string prefix before a quote", wordOrNone, "L'a'", 1, true},
		{"await before a quote", wordOrNone, "await'a'", 5, true},
		{"void before a quote", wordOrNone, "void'a'", 4, true},
		{"delete before a quote", wordOrNone, "delete'a'", 6, true},
		{"throw before a quote", wordOrNone, "throw'a'", 5, true},
		{"other word before a quote", wordOrNone, "don't", 3, false},
		{"digit before a quote", wordOrNone, "1'000", 1, false},
		{"scalar at the start of a line", scalarStart, "a: 1\n'b'", 5, true},
		{"scalar after a flow opener", scalarStart, "k: ['b'", 4, true},
		{"scalar after a colon and a blank", scalarStart, "k: 'b'", 3, true},
		{"scalar after a colon without a blank", scalarStart, "k:'b'", 2, false},
		{"scalar after a word", scalarStart, "k: a'b'", 4, false},
		{"hash after a blank", lineOpens, "a #", 2, true},
		{"hash glued to a word", lineOpens, "a#", 1, false},
		{"hash at the start of a line", lineOpens, "a\n#", 2, true},
		{"slashes after a colon", lineOpens, "a://", 2, false},
		{"slashes after a semicolon", lineOpens, ";//", 1, true},
		{"dashes glued to a word", lineOpens, "a--", 1, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.hook(testCase.src, testCase.at); got != testCase.want {
				t.Fatalf(
					"hook(%q, %d) = %v, want %v",
					testCase.src,
					testCase.at,
					got,
					testCase.want,
				)
			}
		})
	}
}
