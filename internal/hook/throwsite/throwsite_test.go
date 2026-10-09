package throwsite

import (
	"slices"
	"strings"
	"testing"
)

type scanCase struct {
	name string
	ext  string
	src  string
	want []int
}

func TestScan(t *testing.T) {
	t.Parallel()

	testCases := []scanCase{
		{"ts throw in code", ".ts", "function a() {\n  throw new Error('x');\n}\n", []int{2}},
		{"repeat on one line", ".ts", "throw a; throw b\n\nthrow c\n", []int{1, 3}},
		{"ts line comment", ".ts", "// throw here\nx()\n", nil},
		{"ts block comment", ".ts", "/* a\n throw b */\nx()\n", nil},
		{"ts string", ".ts", "const a = 'throw x';\nconst b = \"throw y\"\n", nil},
		{"ts template literal", ".ts", "const a = `\nthrow x\n`\n", nil},
		{"ts template interpolation", ".ts", "const a = `${(() => { throw 1 })()}`\n", []int{1}},
		{"promise reject", ".ts", "const a = 1\nreturn Promise.reject(err)\n", []int{2}},
		{"promise reject in a string", ".ts", "const a = 'Promise.reject('\n", nil},
		{"method named throw", ".ts", "gen.throw(err)\n", nil},
		{"identifier containing throw", ".ts", "const rethrow = throwaway\n", nil},
		{"js file", ".js", "throw new Error()\n", []int{1}},
		{"jsx file", ".jsx", "throw new Error()\n", []int{1}},
		{"tsx file", ".TSX", "throw new Error()\n", []int{1}},
		{"dart throw", ".dart", "void a() {\n  throw StateError('x');\n}\n", []int{2}},
		{"dart rethrow", ".dart", "try {} catch (e) {\n  rethrow;\n}\n", []int{2}},
		{"dart comment", ".dart", "/// throw here\n// rethrow\n", nil},
		{"dart nested block comment", ".dart", "/* a /* b */ throw c */\n", nil},
		{
			"dart raw string", ".dart",
			"var a = r'throw \\d';\nvar b = r\"\"\"\nrethrow\n\"\"\";\n", nil,
		},
		{"dart string", ".dart", "var a = 'throw x';\n", nil},
		{"dart ignores Promise.reject", ".dart", "Promise.reject(x)\n", nil},
		{
			"svelte script", ".svelte",
			"<p>throw</p>\n<script>\n  throw error(1)\n</script>\n",
			[]int{3},
		},
		{"svelte markup only", ".svelte", "<script>let a = 1</script>\n<p>throw</p>\n", nil},
		{
			"jsx apostrophe then a throw", ".tsx",
			"const A = () => <p>Don't</p>\nfunction f() {\n  throw new Error('x')\n}\n",
			[]int{3},
		},
		{
			"unterminated literal then a throw in a string", ".ts",
			"const a = 'oops\nconst b = 'throw x'\nthrow y\n",
			[]int{3},
		},
		{
			"unterminated literal then a commented throw", ".ts",
			"const a = 'oops\nx() // throw z\nthrow y\n",
			[]int{3},
		},
		{"go file", ".go", "panic(1)\nthrow\n", nil},
		{"unknown file", ".md", "throw\n", nil},
	}
	runScanCases(t, testCases)
}

func TestScan_AfterAnOpenString(t *testing.T) {
	t.Parallel()

	testCases := []scanCase{
		{
			"jsdoc after jsx text", ".tsx",
			"<p>Don't panic</p>\n/**\n * Never throw here.\n */\nfunction a() {}\n",
			nil,
		},
		{
			"template literal after jsx text", ".tsx",
			"<p>Don't</p>\nconst a = `\nthrow x\n`\n", nil,
		},
		{"regex after jsx text", ".tsx", "<p>Don't</p>\nconst a = /throw/\n", nil},
		{"block comment after an open string", ".ts", "const a = 'oops\n/*\nthrow b\n*/\n", nil},
		{
			"dart triple quote after an open string", ".dart",
			"var a = 'oops\nvar b = '''\nthrow x\n''';\n", nil,
		},
		{"real throw after jsx text", ".tsx", "<p>Don't</p>\nthrow new Error('x')\n", []int{2}},
		{"real throw after two open strings", ".ts", "a = 'x\nb = 'y\nthrow z\n", []int{3}},
		{"throw after the quote on its own line", ".tsx", "<p>Don't throw</p>\n", nil},
		{"a trailing comment apostrophe opens no string", ".tsx", "throw a; // Don't\n", []int{1}},
		{"throw before a real open string", ".tsx", "throw a; <p>Don't</p>\n", []int{1}},
		{"open string on the last line", ".ts", "throw a\nconst b = 'oops", []int{1}},
		{"crlf line ends", ".tsx", "<p>Don't</p>\r\n/*\r\nthrow a\r\n*/\r\nthrow b\r\n", []int{5}},
		{
			"jsdoc then a lone backtick in jsx", ".tsx",
			"/**\n * Never throw here.\n */\nconst a = <p>Press `to open</p>\nthrow b\n", nil,
		},
		{"open quote then an open comment", ".ts", "var a = 'x\n/*\n * Never throw here.\n", nil},
		{"throw before an unclosed comment", ".ts", "throw a\n/* open\nthrow b\n", []int{1}},
		{"throw before an unclosed template", ".ts", "throw a\nconst b = `x\nthrow c\n", []int{1}},
		{"unclosed dart triple quote", ".dart", "rethrow;\nvar a = '''x\nthrow y\n", []int{1}},
		{"throw after an unclosed ${ in a template", ".ts", "const s = `a ${b\nthrow e\n", nil},
		{"throw after an unclosed ${ in dart", ".dart", "var a = '${b\nthrow e;\n", nil},
		{"throw before an unclosed ${", ".ts", "throw a\nconst s = `a ${b\nthrow e\n", []int{1}},
		{
			"svelte apostrophe then a markup opener in a string", ".svelte",
			"<script>\n  const a = 'oops\n  const s = \"<!--\"\n  throw e\n</script>\n",
			[]int{4},
		},
	}
	runScanCases(t, testCases)
}

func TestScan_JSXTextIsNotLexed(t *testing.T) {
	t.Parallel()

	testCases := []scanCase{
		{
			"a lone backtick in JSX text can pair with a later template", ".tsx",
			"/** Toolbar hint. */\nexport const Hint = () => <p>Press ` to open</p>\n" +
				"const a = `x`;\nconst msg = `Do not throw ${name}`;\n",
			[]int{4},
		},
		{
			"an apostrophe in an interpolation hides its brace and the rest", ".tsx",
			"const s = `\n  ${cond ? <b>Don't</b> : null}\n  never throw here\n`\n", nil,
		},
	}
	runScanCases(t, testCases)
}

func TestScan_FindsAThrowAfterManyOpenStrings(t *testing.T) {
	t.Parallel()

	const line = "<p>Don't panic</p>\n"
	src := strings.Repeat(line, (1<<20)/len(line)) + "throw a\n"
	hits := Scan(".tsx", src)
	if want := strings.Count(src, "\n"); len(hits) != 1 || hits[0].Line != want {
		t.Errorf("Scan hits = %v, want one throw on line %d", hits, want)
	}
}

func TestHandles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ext  string
		want bool
	}{
		{".dart", true},
		{".ts", true},
		{".tsx", true},
		{".mts", true},
		{".cts", true},
		{".js", true},
		{".jsx", true},
		{".JSX", true},
		{".vue", false},
		{".mjs", true},
		{".cjs", true},
		{".svelte", true},
		{".DART", true},
		{".go", false},
		{".md", false},
		{"", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.ext, func(t *testing.T) {
			t.Parallel()

			if got := Handles(testCase.ext); got != testCase.want {
				t.Errorf("Handles(%q) = %v, want %v", testCase.ext, got, testCase.want)
			}
		})
	}
}

func runScanCases(t *testing.T, testCases []scanCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := hitLines(Scan(testCase.ext, testCase.src))
			if !slices.Equal(got, testCase.want) {
				t.Errorf("Scan(%q) lines = %v, want %v", testCase.src, got, testCase.want)
			}
		})
	}
}

func hitLines(hits []Hit) []int {
	lines := make([]int, 0, len(hits))
	for _, hit := range hits {
		lines = append(lines, hit.Line)
	}
	return lines
}

func TestScan_NamesTheMatchedWord(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ext  string
		src  string
		want []Hit
	}{
		{"throw", ".ts", "throw a\n", []Hit{{1, "throw"}}},
		{"promise reject", ".js", "\nreturn Promise.reject(e)\n", []Hit{{2, "Promise.reject"}}},
		{"rethrow", ".dart", "rethrow;\n", []Hit{{1, "rethrow"}}},
		{
			"two words on one line", ".ts", "Promise.reject(a); throw b\n",
			[]Hit{{1, "Promise.reject"}, {1, "throw"}},
		},
		{"one word twice on a line", ".ts", "throw a; throw b\n", []Hit{{1, "throw"}}},
		{
			"two words",
			".ts",
			"throw a\nPromise.reject(b)\n",
			[]Hit{{1, "throw"}, {2, "Promise.reject"}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Scan(testCase.ext, testCase.src); !slices.Equal(got, testCase.want) {
				t.Errorf("Scan(%q) = %v, want %v", testCase.src, got, testCase.want)
			}
		})
	}
}
