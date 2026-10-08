package language

import (
	"strings"
	"testing"
)

func TestScanLiterals_ContainsDart(t *testing.T) {
	t.Parallel()

	testCases := []containsCase{
		{"single", ".dart", "var a = 'x\u00A0y';", "\u00A0", true},
		{"triple double", ".dart", "var a = \"\"\"x\n\u00A0\"\"\";", "\u00A0", true},
		{"raw", ".dart", "var a = r'x\\\u00A0y';", "\u00A0", true},
		{"interpolation in single", ".dart", "var a = 'x ${b\u00A0c} y';", "\u00A0", false},
		{"interpolation in double", ".dart", "var a = \"x ${b\u00A0c} y\";", "\u00A0", false},
		{"nested literal in interpolation", ".dart", "'${m['\u00A0']}'", "\u00A0", true},
		{"comment quote", ".dart", "// it's\nvar a\u00A0b;", "\u00A0", false},
		{"backtick is plain code", ".dart", "var a\u00A0b = `x`;", "\u00A0", false},
		{"slash is division", ".dart", "var a = b / c\u00A0/ d;", "\u00A0", false},
		{"upper case extension", ".DART", "var a = 'x\u00A0';", "\u00A0", true},
	}
	runContainsCases(t, testCases)
}

func TestScanLiterals_ContainsJS(t *testing.T) {
	t.Parallel()

	testCases := []containsCase{
		{"mjs string", ".mjs", "const NBSP = '\u00A0';", "\u00A0", true},
		{"ts string", ".ts", "const a = \"x\u00A0\";", "\u00A0", true},
		{"template text", ".ts", "const a = `x\u00A0${b}`;", "\u00A0", true},
		{"template interpolation", ".ts", "const a = `x${b\u00A0c}`;", "\u00A0", false},
		{"double quote holds a dollar brace", ".js", "a = \"${\"; b\u00A0c = 1;", "\u00A0", false},
		{"single quote holds a dollar brace", ".js", "a = '${'; b\u00A0c = 1;", "\u00A0", false},
		{"cts", ".cts", "const a = '\u00A0';", "\u00A0", true},
		{"mts", ".mts", "const a = '\u00A0';", "\u00A0", true},
		{"tsx", ".tsx", "const a = '\u00A0';", "\u00A0", true},
		{"jsx", ".jsx", "const a = '\u00A0';", "\u00A0", true},
		{"cjs", ".cjs", "const a = '\u00A0';", "\u00A0", true},
		{"CRLF continuation", ".js", "a = 'x\\\r\n\u00A0';", "\u00A0", true},
		{"double quote LF continuation", ".js", "a = \"x\\\n\u00A0\";", "\u00A0", true},
		{"double quote CRLF continuation", ".ts", "a = \"x\\\r\n\u00A0\";", "\u00A0", true},
		{"python is not scanned", ".py", "a = 'x\u00A0'", "\u00A0", false},
		{"markdown has no literal", ".md", "say \"a\u00A0b\"", "\u00A0", false},
		{"unknown extension", ".zzz", "a = 'x\u00A0'", "\u00A0", false},
	}
	runContainsCases(t, testCases)
}

func TestScanLiterals_ContainsRegex(t *testing.T) {
	t.Parallel()

	testCases := []containsCase{
		{"regex body", ".mjs", "x = /a\u00A0b\"/g;", "\u00A0", true},
		{"after paren", ".js", "m.matchAll(/<li>\u00A0/g)", "\u00A0", true},
		{"string next to a regex", ".js", "s.replace(/\"/g, \"R$\u00A010\")", "\u00A0", true},
		{"backtick in a regex", ".ts", "x = /`/; y\u00A0z = 1;", "\u00A0", false},
		{"division", ".js", "a / b\u00A0/ c", "\u00A0", false},
		{"regex in svelte script", ".svelte", "<script>x = /a\u00A0/;</script>", "\u00A0", true},
		{"slash in svelte markup", ".svelte", "<p>a = /\u00A0/</p>", "\u00A0", false},
		{"dart has no regex literal", ".dart", "var a = /\u00A0/;", "\u00A0", false},
	}
	runContainsCases(t, testCases)
}

func TestScanLiterals_ContainsSvelte(t *testing.T) {
	t.Parallel()

	testCases := []containsCase{
		{"script string", ".svelte", "<script>let a = '\u00A0';</script>", "\u00A0", true},
		{"vue script string", ".vue", "<script>let a = '\u00A0';</script>", "\u00A0", true},
		{"markup apostrophe", ".svelte", "<p>it's\u00A0here</p>", "\u00A0", false},
		{
			"markup after script",
			".svelte",
			"<script>let a = 'b';</script><p>'\u00A0'</p>",
			"\u00A0",
			false,
		},
		{
			"marker in the first of two scripts",
			".svelte",
			"<script context=\"module\">let a = '\u00A0';</script>\n<script>let b = 1;</script>",
			"\u00A0",
			true,
		},
		{
			"vue marker in the first of two scripts",
			".vue",
			"<script>let a = '\u00A0';</script>\n<script setup>let b = 1;</script>",
			"\u00A0",
			true,
		},
		{
			"string in a style is not a js string",
			".svelte",
			"<style>a::after { content: \"\u00A0\"; }</style>",
			"\u00A0",
			false,
		},
		{
			"vue style string is not a js string",
			".vue",
			"<script>let a = 1;</script><style>a::after { content: '\u00A0'; }</style>",
			"\u00A0",
			false,
		},
	}
	runContainsCases(t, testCases)
}

func TestScanLiterals_LeavesOtherLanguagesAlone(t *testing.T) {
	t.Parallel()

	testCases := []containsCase{
		{"rust string", ".rs", "let a = \"x\u00A0\";", "\u00A0", false},
		{"rust char", ".rs", "let a = '\u00A0';", "\u00A0", false},
		{"php string", ".php", "$a = 'x\u00A0';", "\u00A0", false},
		{"java string", ".java", "String a = \"x\u00A0\";", "\u00A0", false},
		{"kotlin string", ".kt", "val a = \"x\u00A0\"", "\u00A0", false},
		{"swift string", ".swift", "let a = \"x\u00A0\"", "\u00A0", false},
		{"c string", ".c", "char *a = \"x\u00A0\";", "\u00A0", false},
		{"csharp string", ".cs", "var a = \"x\u00A0\";", "\u00A0", false},
		{"scala string", ".scala", "val a = \"x\u00A0\"", "\u00A0", false},
		{"proto string", ".proto", "option a = \"x\u00A0\";", "\u00A0", false},
	}
	runContainsCases(t, testCases)
}

func TestLiterals_RawJS(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		src    string
		marker string
		raw    bool
		regex  bool
	}{
		{"String.raw template", "x = String.raw`a\u00A0b`;", "\u00A0", true, false},
		{"String.raw escaped backtick", "x = String.raw`a\\`\u00A0`;", "\u00A0", true, false},
		{"String.raw then a blank", "x = String.raw `a\u00A0b`;", "\u00A0", true, false},
		{"String.raw then blanks", "x = String.raw \t `a\u00A0b`;", "\u00A0", true, false},
		{"String.raw then a newline", "x = String.raw\n`a\u00A0b`;", "\u00A0", true, false},
		{"String.raw then a comment", "x = String.raw/* c */`a\u00A0b`;", "\u00A0", true, false},
		{"String.raw at the file start", "String.raw`a\u00A0b`", "\u00A0", true, false},
		{"another tag is not raw", "x = tag`a\u00A0b`;", "\u00A0", false, false},
		{"longer word is not raw", "x = MyString.raw`a\u00A0b`;", "\u00A0", false, false},
		{"raw as a plain word is not raw", "x = a.raw`a\u00A0b`;", "\u00A0", false, false},
		{"String.raw call is not a tag", "x = String.raw(`a\u00A0b`);", "\u00A0", false, false},
		{"plain template", "x = `a\u00A0b`;", "\u00A0", false, false},
		{"single quoted", "x = 'a\u00A0b';", "\u00A0", false, false},
		{"regex", "x = /a\u00A0b/;", "\u00A0", false, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			offset := strings.Index(testCase.src, testCase.marker)
			found := ScanLiterals(".js", testCase.src)
			if got := found.Raw(offset); got != testCase.raw {
				t.Errorf("Raw = %v, want %v for %q", got, testCase.raw, testCase.src)
			}
			if got := found.Regex(offset); got != testCase.regex {
				t.Errorf("Regex = %v, want %v for %q", got, testCase.regex, testCase.src)
			}
		})
	}
}

func inScript(body string) string { return "<script>" + body + "</script>" }

func TestScanLiterals_CommentsHideQuotes(t *testing.T) {
	t.Parallel()

	testCases := []containsCase{
		{"go block", ".go", "/* it's */ var a = \"x\u00A0\"", "\u00A0", true},
		{"go line", ".go", "// it's\nvar a = \"x\u00A0\"", "\u00A0", true},
		{"dart block", ".dart", "/* it's */ var a = 'x\u00A0';", "\u00A0", true},
		{"dart nested block", ".dart", "/* a /* b */ it's */ var a = 'x\u00A0';", "\u00A0", true},
		{
			"dart after nested", ".dart",
			"/* /* */ */ /* it's */ var a = 'x\u00A0';", "\u00A0", true,
		},
		{"ts block does not nest", ".ts", "/* a /* b */ var a = 'x\u00A0';", "\u00A0", true},
		{"dart line", ".dart", "// it's\nvar a = 'x\u00A0';", "\u00A0", true},
		{"ts block", ".ts", "/* it's */ const a = 'x\u00A0';", "\u00A0", true},
		{"ts line", ".ts", "// it's\nconst a = 'x\u00A0';", "\u00A0", true},
		{"svelte block", ".svelte", inScript("/* it's */ let a = 'x\u00A0';"), "\u00A0", true},
		{"svelte line", ".svelte", inScript("// it's\nlet a = 'x\u00A0';"), "\u00A0", true},
		{"vue block", ".vue", inScript("/* it's */ let a = 'x\u00A0';"), "\u00A0", true},
		{"svelte html", ".svelte", inScript("<!-- it's -->let a = 'x\u00A0';"), "\u00A0", true},
	}
	runContainsCases(t, testCases)
}
