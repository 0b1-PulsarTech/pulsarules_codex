package narration

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
		{"js br is not a raw prefix", "js", "x = br\"y\\\" // the old code\"\n", 0},
		{"js cr is not a raw prefix", "js", "x = cr\"y\\\" // the old code\"\n", 0},
		{
			"cs reversed verbatim keeps a backslash",
			"cs",
			"var s = @$\"C:\\\" // the old code\n",
			1,
		},
		{
			"cs interpolated escapes a backslash",
			"cs",
			"var s = $\"a\\\" // the old code\";\n",
			0,
		},
		{"cs verbatim closes at the end of line", "cs", "var s = @\"a\"", 0},
		{"cs verbatim doubled quote ends the line", "cs", "var s = @\"a\"\"\"", 0},
		{"cs verbatim at column one", "cs", "@\"a // the old code\"\n", 0},
		{"objc at-string is not verbatim", "java", "x = @\"a\\\" // the old code\";\n", 0},
		{"hashed single quote is not raw", "js", "x = r#'a // the old code'\n", 0},
		{"cs verbatim hides slashes", "cs", "var s = @\"a // the old code\";\n", 0},
		{"cs verbatim backslash is literal", "cs", "var s = @\"C:\\\" // the old code\n", 1},
		{
			"cs verbatim doubled quote before a backslash",
			"cs",
			"var s = @\"a\"\"\\\" // the old code\"\n",
			1,
		},
		{"cs verbatim doubled quote", "cs", "var s = @\"a\"\" // the old code\";\n", 0},
		{"cs interpolated verbatim", "cs", "var s = $@\"a // the old code\";\n", 0},
		{"cs reversed interpolated verbatim", "cs", "var s = @$\"a // the old code\";\n", 0},
		{
			"cs regular string backslash escapes",
			"cs",
			"var s = \"a\\\" // the old code\";\n",
			0,
		},
	}
}
