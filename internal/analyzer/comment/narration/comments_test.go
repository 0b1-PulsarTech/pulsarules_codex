package narration

import (
	"slices"
	"testing"
)

func TestScanSegments(t *testing.T) {
	t.Parallel()

	for _, testCase := range slices.Concat(boundaryCases(), lineStartCases()) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			runScan(t, testCase)
		})
	}
}

func boundaryCases() []scanCase {
	return []scanCase{
		{"domain prose: no longer names", "go", "// the pair no longer names a " +
			"live template\n", 0},
		{"domain prose: previously sealed", "go", "// never clobbers a previously " +
			"sealed PIN\n", 0},
		{
			"domain prose: no longer match",
			"go",
			"// an erased contact can no longer match anyone\n",
			0,
		},
		{"string literal is not a comment", "go", "x := \"was replaced by\"\n", 0},
		{"identifier is not a phrase", "go", "used_to_be := 1\n", 0},
		{"trailing comment is flagged", "go", "x := 1 // used to be 2\n", 1},
		{"code after a block comment is ignored", "go", "/* x */ s := \"no longer needed\"\n", 0},
		{"raw string fixture is not a comment", "go", "src := `\n// used to be a loop\n`\n", 0},
		{"url in a string is not a comment", "go", "u := \"http://x used to be\"\n", 0},
		{"html comment in a raw string", "go", "s := `<!-- used to be -->`\n", 0},
		{"comma after previously", "go", "// previously, it looped\n", 1},
		{"previously it with a period", "go", "// that is what we did previously it.\n", 1},
		{"phrase split across two lines", "go", "// the guard was\n// replaced by a check\n", 1},
		{"blank line separates blocks", "go", "// the guard was\n\n// replaced by a check\n", 0},
		{"used to begin", "go", "// the offset is used to begin the scan\n", 0},
		{"used to become", "go", "// a draft is used to become a template\n", 0},
		{"used to bear", "go", "// the key used to bear the payload\n", 0},
		{"used to liven", "go", "// the TTL used to liven the cache\n", 0},
		{
			"old version is domain prose",
			"go",
			"// keeps the old version of the record for audit\n",
			0,
		},
		{"previously iterated", "go", "// the slice previously iterated\n", 0},
		{"python hash comment", "hash.py", "x = 1  # was replaced by y\n", 1},
		{"python string with hash-free text", "hash.py", "x = \"was replaced by\"\n", 0},
		{
			"multi-line non-go block comment",
			"clike",
			"/* the guard was\n replaced by a check */\n",
			1,
		},
		{"clean comment", "go", "// guards the cache\n", 0},
	}
}

func lineStartCases() []scanCase {
	return []scanCase{
		{"shell backtick is not a string", "hash.sh", "echo `date # the old code\n", 1},
		{"python hash after a tab", "hash.py", "x = 1\t# the old code\n", 1},
		{"ts comment glued to a semicolon", "clike", "foo();// the old code\n", 1},
		{"c comment glued to a brace", "clike.c", "}// the old code\n", 1},
		{"sql dashes need no space", "sql", "SELECT 1;-- the old code\n", 1},
		{"url slashes are not a comment", "clike", "see http://x.y/the-old-code\n", 0},
		{"shell hash glued to a word", "hash.sh", "echo a#the old code\n", 0},
	}
}
