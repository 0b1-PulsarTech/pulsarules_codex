package langspec

import (
	"strings"
	"testing"
)

type containsCase struct {
	name   string
	ext    string
	src    string
	marker string
	inside bool
}

func runContainsCases(t *testing.T, testCases []containsCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			offset := strings.Index(testCase.src, testCase.marker)
			if offset < 0 {
				t.Fatalf("marker missing from %q", testCase.src)
			}
			got := ScanLiterals(testCase.ext, testCase.src).Contains(offset)
			if got != testCase.inside {
				t.Errorf("Contains = %v, want %v for %q", got, testCase.inside, testCase.src)
			}
		})
	}
}

func TestScanLiterals_ContainsGo(t *testing.T) {
	t.Parallel()

	testCases := []containsCase{
		{"go interpreted", ".go", "const a = \"x\u00A0y\"", "\u00A0", true},
		{"go raw", ".go", "const a = `x\u00A0y`", "\u00A0", true},
		{"go raw quote and openers", ".go", "a := `say \"x\u00A0\" // y /* z`", "\u00A0", true},
		{"go raw with rune quote", ".go", "a := `say 'x\u00A0' // y`", "\u00A0", true},
		{"go rune", ".go", "const a = '\u00A0'", "\u00A0", true},
		{"go escaped quote", ".go", "a = \"\\\"\" + \"\u00A0\"", "\u00A0", true},
		{"go line comment", ".go", "// say \"hi\"\nx := 1 // \u00A0", "\u00A0", false},
		{"go block comment quote", ".go", "/* ' */ var a\u00A0b int", "\u00A0", false},
		{"go identifier", ".go", "var a\u00A0b int", "\u00A0", false},
		{"go comment after literal", ".go", "a := \"x\" // \u00A0", "\u00A0", false},
	}
	runContainsCases(t, testCases)
}

func TestLiterals_Escape(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ext  string
		char rune
		want string
	}{
		{"go bmp", ".go", '\u00A0', `\u00A0`},
		{"go astral", ".go", '\U0001F600', `\U0001F600`},
		{"dart bmp", ".dart", '\u200D', `\u200D`},
		{"dart astral", ".dart", '\U0001F600', `\u{1F600}`},
		{"ts bmp", ".ts", '\u2026', `\u2026`},
		{"ts astral", ".ts", '\U000E0001', `\u{E0001}`},
		{"svelte astral", ".svelte", '\U0001F600', `\u{1F600}`},
		{"mjs bmp", ".mjs", '\u00AD', `\u00AD`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := ScanLiterals(testCase.ext, "").Escape(testCase.char); got != testCase.want {
				t.Errorf("Escape(%U) = %q, want %q", testCase.char, got, testCase.want)
			}
		})
	}
}

func TestLiterals_Raw(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		ext    string
		src    string
		marker string
		want   bool
	}{
		{"go raw", ".go", "a := `x\u00A0y`", "\u00A0", true},
		{"go interpreted", ".go", "a := \"x\u00A0y\"", "\u00A0", false},
		{"go rune", ".go", "a := '\u00A0'", "\u00A0", false},
		{"go outside a literal", ".go", "a\u00A0b := `x`", "\u00A0", false},
		{"dart raw single", ".dart", "var a = r'x\u00A0';", "\u00A0", true},
		{"dart raw double", ".dart", "var a = r\"x\u00A0\";", "\u00A0", true},
		{"dart raw triple", ".dart", "var a = r\"\"\"x\n\u00A0\"\"\";", "\u00A0", true},
		{"dart plain", ".dart", "var a = 'x\u00A0';", "\u00A0", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			offset := strings.Index(testCase.src, testCase.marker)
			got := ScanLiterals(testCase.ext, testCase.src).Raw(offset)
			if got != testCase.want {
				t.Errorf("Raw = %v, want %v for %q", got, testCase.want, testCase.src)
			}
		})
	}
}

func TestLiterals_Covers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		ext    string
		src    string
		marker string
		want   bool
	}{
		{"go file", ".go", "x := 1 // \u00A0", "\u00A0", true},
		{"dart file", ".dart", "x // \u00A0", "\u00A0", true},
		{"ts file", ".ts", "x // \u00A0", "\u00A0", true},
		{"svelte script", ".svelte", "<script>x // \u00A0</script>", "\u00A0", true},
		{"markup before script", ".svelte", "<p>\u00A0</p><script>x</script>", "\u00A0", false},
		{"markup after script", ".vue", "<script>x</script><p>\u00A0</p>", "\u00A0", false},
		{"svelte attribute", ".svelte", "<input placeholder=\"R$\u00A00\" />", "\u00A0", false},
		{"rust file", ".rs", "x // \u00A0", "\u00A0", false},
		{"php file", ".php", "x // \u00A0", "\u00A0", false},
		{"markdown file", ".md", "x \u00A0", "\u00A0", false},
		{"unknown extension", ".zzz", "x \u00A0", "\u00A0", false},
		{"cut-off go file", ".go", "/* \u00A0", "\u00A0", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			offset := strings.Index(testCase.src, testCase.marker)
			got := ScanLiterals(testCase.ext, testCase.src).Covers(offset)
			if got != testCase.want {
				t.Errorf("Covers = %v, want %v for %q", got, testCase.want, testCase.src)
			}
		})
	}
}

func TestLiterals_Incomplete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ext  string
		src  string
		want bool
	}{
		{"go closed", ".go", "const a = \"x\u00A0\"\n", false},
		{"go unterminated comment", ".go", "/* todo\nconst a = \"x\u00A0\"\n", true},
		{"go odd backtick", ".go", "var a = `half\nconst b = \"x\u00A0\"\nvar c = `y`\n", true},
		{"ts cut-off template", ".ts", "const a = `x\u00A0 ${b\n", true},
		{"dart cut-off triple quote", ".dart", "var a = '''x\u00A0\n", true},
		{"svelte cut-off script comment", ".svelte", "<script>/* x\u00A0</script>", true},
		{"unscanned language", ".rs", "/* x\u00A0", false},
		{"markdown", ".md", "`x\u00A0", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			found := ScanLiterals(testCase.ext, testCase.src)
			if got := found.Incomplete(); got != testCase.want {
				t.Errorf("Incomplete = %v, want %v for %q", got, testCase.want, testCase.src)
			}
			offset := strings.Index(testCase.src, "\u00A0")
			if testCase.want && (found.Covers(offset) || found.Contains(offset)) {
				t.Errorf("an incomplete scan must claim nothing: %q", testCase.src)
			}
		})
	}
}

type unreadCase struct {
	name          string
	ext           string
	src           string
	line          int
	incomplete    bool
	firstString   bool
	lastIsCovered bool
}

func checkUnread(t *testing.T, testCase unreadCase) {
	t.Helper()
	found := ScanLiterals(testCase.ext, testCase.src)
	if found.UnreadFrom() != testCase.line {
		t.Errorf("UnreadFrom = %d, want %d", found.UnreadFrom(), testCase.line)
	}
	if found.Incomplete() != testCase.incomplete {
		t.Errorf("Incomplete = %v, want %v", found.Incomplete(), testCase.incomplete)
	}
	first := strings.Index(testCase.src, "\u00A0")
	if found.Contains(first) != testCase.firstString {
		t.Errorf("Contains(first) = %v, want %v", found.Contains(first), testCase.firstString)
	}
	last := strings.LastIndex(testCase.src, "\u00A0")
	if found.Covers(last) != testCase.lastIsCovered {
		t.Errorf("Covers(last) = %v, want %v", found.Covers(last), testCase.lastIsCovered)
	}
}

func TestLiterals_UnreadFrom(t *testing.T) {
	t.Parallel()

	const (
		svltEOF = "<script>\nlet a = '\u00A0';\nlet b = 'c"
		jsx     = "const a = \"R$\u00A010\";\nconst b = <p>\n  Don't\u00A0show\n</p>;\n"
		svlt    = "<script>\nlet a = '\u00A0';\nlet b = 'c\n</script>\n<p>\u00A0</p>"
	)
	testCases := []unreadCase{
		{
			"a file read whole",
			".ts",
			"const a = \"R$\u00A010\";\nx\u00A0y\n",
			0,
			false,
			true,
			true,
		},
		{"a quote cut at a newline", ".tsx", jsx, 3, true, true, false},
		{"a literal on line 1 cut at a newline", ".ts", "x = 'y\u00A0\nz", 1, true, false, false},
		{
			"a literal that runs to EOF",
			".ts",
			"const a = \"R$\u00A010\";\nx = 'y\u00A0",
			0,
			true,
			false,
			false,
		},
		{
			"a comment that runs to EOF",
			".go",
			"var a = \"\u00A0\"\n/* x\u00A0",
			0,
			true,
			false,
			false,
		},
		{"a cut inside a svelte script", ".svelte", svlt, 3, true, true, false},
		{"a literal cut at offset zero", ".ts", "'y\u00A0\nz", 1, true, false, false},
		{"a svelte script that runs to EOF", ".svelte", svltEOF, 0, true, false, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			checkUnread(t, testCase)
		})
	}
}
