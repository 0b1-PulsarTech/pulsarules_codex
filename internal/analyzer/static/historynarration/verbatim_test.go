package historynarration

import "testing"

func TestVerbatimAndRawPrefixes(t *testing.T) {
	t.Parallel()

	for _, testCase := range verbatimCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			runScan(t, testCase)
		})
	}
}

func verbatimCases() []scanCase {
	return []scanCase{
		{"js br is not a raw prefix", "clike.js", "x = br\"y\\\" // the old code\"\n", 0},
		{"js cr is not a raw prefix", "clike.js", "x = cr\"y\\\" // the old code\"\n", 0},
		{
			"cs reversed verbatim keeps a backslash",
			"clike.cs",
			"var s = @$\"C:\\\" // the old code\n",
			1,
		},
		{
			"cs interpolated escapes a backslash",
			"clike.cs",
			"var s = $\"a\\\" // the old code\";\n",
			0,
		},
		{"cs verbatim closes at the end of line", "clike.cs", "var s = @\"a\"", 0},
		{"cs verbatim doubled quote ends the line", "clike.cs", "var s = @\"a\"\"\"", 0},
		{"cs verbatim at column one", "clike.cs", "@\"a // the old code\"\n", 0},
		{"objc at-string is not verbatim", "clike.m", "x = @\"a\\\" // the old code\";\n", 0},
		{"hashed single quote is not raw", "clike.js", "x = r#'a // the old code'\n", 0},
		{"cs verbatim hides slashes", "clike.cs", "var s = @\"a // the old code\";\n", 0},
		{"cs verbatim backslash is literal", "clike.cs", "var s = @\"C:\\\" // the old code\n", 1},
		{
			"cs verbatim doubled quote before a backslash",
			"clike.cs",
			"var s = @\"a\"\"\\\" // the old code\"\n",
			1,
		},
		{"cs verbatim doubled quote", "clike.cs", "var s = @\"a\"\" // the old code\";\n", 0},
		{"cs interpolated verbatim", "clike.cs", "var s = $@\"a // the old code\";\n", 0},
		{"cs reversed interpolated verbatim", "clike.cs", "var s = @$\"a // the old code\";\n", 0},
		{
			"cs regular string backslash escapes",
			"clike.cs",
			"var s = \"a\\\" // the old code\";\n",
			0,
		},
	}
}
