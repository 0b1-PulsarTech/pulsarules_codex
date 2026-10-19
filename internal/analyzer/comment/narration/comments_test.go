package narration

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestScanSegments_Boundaries(t *testing.T) {
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
		{"python hash comment", "py", "x = 1  # was replaced by y\n", 1},
		{"python string with hash-free text", "py", "x = \"was replaced by\"\n", 0},
		{
			"url slashes are not a line comment in c",
			"ts",
			"char *u = \"http://x used to be\";\n",
			0,
		},
		{
			"multi-line non-go block comment",
			"ts",
			"/* the guard was\n replaced by a check */\n",
			1,
		},
		{"ts string with a slash comment", "ts", "s = \"see // was replaced by b\";\n", 0},
		{"ts glob does not open a block", "ts", "a = \"/api/*\"; b = \"the old code\";\n", 0},
		{"dart single quoted", "dart", "var x = 'x // previously this ran';\n", 0},
		{"dart raw string", "dart", "var x = r'/* the old code */';\n", 0},
		{"dart raw double quoted", "dart", "var x = r\"// the old code\";\n", 0},
		{"proto json_name", "ts", "string a = 1 [json_name = \"a // no longer used\"];\n", 0},
		{"escaped quote stays in the string", "ts", "s = \"a\\\" // the old code\";\n", 0},
		{"comment after a closed string", "ts", "s = \"a\"; // the old code\n", 1},
		{"ts template literal", "ts", "s = `\n// the old code\n`;\n", 0},
		{
			"rust lifetime is not a string",
			"rs",
			"fn f<'a>(x: &'a str) { // the old code\n",
			1,
		},
		{"sql string with a dash comment", "sql", "SELECT 'a -- was replaced by b';\n", 0},
		{"sql doubled quote escape", "sql", "SELECT 'it''s -- the old code';\n", 0},
		{"sql backslash is no escape", "sql", "SELECT 'a\\' -- the old code\n", 1},
		{"python string with a hash", "py", "x = \"a # was replaced by b\"\n", 0},
		{"clean comment", "go", "// guards the cache\n", 0},
	}
}

func lineStartCases() []scanCase {
	return []scanCase{
		{"shell backtick is not a string", "sh", "echo `date # the old code\n", 1},
		{"python hash after a tab", "py", "x = 1\t# the old code\n", 1},
		{"ts comment glued to a semicolon", "ts", "foo();// the old code\n", 1},
		{"c comment glued to a brace", "c", "}// the old code\n", 1},
		{"sql dashes need no space", "sql", "SELECT 1;-- the old code\n", 1},
		{"url slashes are not a comment", "ts", "see http://x.y/the-old-code\n", 0},
		{"shell hash glued to a word", "sh", "echo a#the old code\n", 0},
	}
}

func TestScanSegments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		ext    string
		source string
		want   []segment
	}{
		{"line comment", ".py", "x = 1 # a\n", []segment{{1, " a"}}},
		{"empty line comment", ".py", "x\n#\n", []segment{{2, ""}}},
		{"block comment", ".c", "/* a */ x\n", []segment{{1, " a "}}},
		{
			"block comment lines and the closer line",
			".c",
			"x\n/* a\n  b\n*/\n",
			[]segment{{2, " a"}, {3, "  b"}, {4, ""}},
		},
		{
			"empty lines of a block comment leave a gap",
			".c",
			"/* a\n\nb */\n",
			[]segment{{1, " a"}, {3, "b "}},
		},
		{"block comment opened at the end of its line", ".c", "/*\na\n", []segment{{2, "a"}}},
		{"unterminated block comment", ".c", "/* a\nb", []segment{{1, " a"}, {2, "b"}}},
		{"no comment", ".c", "x = \"// a\"\n", nil},
		{"unknown extension", ".zzz", "// a\n", nil},
		{
			"html comment among markup, script and style",
			".svelte",
			"<!-- a -->\n<script>\n// b\n</script>\n<style>\n/* c */\n</style>\n",
			[]segment{{1, " a "}, {3, " b"}, {6, " c "}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := scanSegments(testCase.source, testCase.ext)
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("scanSegments(%q) = %v, want %v", testCase.source, got, testCase.want)
			}
		})
	}
}

func TestScanFileDartBlockCommentsNest(t *testing.T) {
	t.Parallel()

	source := "/* outer one /* inner two */ outer three\n   the old code used to be nested */\n"
	fc := pass.FileChange{Path: "f.dart", Extension: ".dart"}
	got := scanFile([]byte(source), fc, NewAnalyzer().reporter(), parsedGo{})
	if len(got) == 0 || got[0].Line != 2 {
		t.Fatalf("findings = %+v, want the first on line 2", got)
	}
}
