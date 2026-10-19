package narration

import (
	"slices"
	"testing"
)

func TestQuoteRules(t *testing.T) {
	t.Parallel()

	for _, testCase := range slices.Concat(
		pythonCases(), shellSQLCases(), cLikeCases(), rustCases(),
	) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			runScan(t, testCase)
		})
	}
}

func pythonCases() []scanCase {
	return []scanCase{
		{
			"python upper f-string",
			"py",
			"print(F'total: {n}')  # it's the old code\n",
			1,
		},
		{
			"python f-string closes after a space",
			"py",
			"x = F'it is ' + y  # it's the old code\n",
			1,
		},
		{
			"python bytes closes after a paren",
			"py",
			"x = B'(a)'  # we used to call it's helper\n",
			1,
		},
		{
			"python raw bytes",
			"py",
			"pat = Rb'\\d+'  # it's the old code\n",
			1,
		},
		{
			"python t-string",
			"py",
			"msg = t'hi {name}'  # it's the old code\n",
			1,
		},
		{
			"python upper f-string hides a hash",
			"py",
			"x = F'a # the old code'\n",
			0,
		},
		{
			"python upper bytes hides a hash",
			"py",
			"x = B'a # the old code'\n",
			0,
		},
		{
			"python raw bytes hides a hash",
			"py",
			"x = Rb'a # the old code'\n",
			0,
		},
		{
			"python t-string hides a hash",
			"py",
			"x = t'a # the old code'\n",
			0,
		},
		{
			"python escaped quote stays in the string",
			"py",
			"x = 'a\\' # the old code'\n",
			0,
		},
	}
}

func shellSQLCases() []scanCase {
	return []scanCase{
		{
			"shell glued word quote",
			"sh",
			"echo foo'bar' # it's the old code\n",
			1,
		},
		{
			"shell quoted text hides a hash",
			"sh",
			"echo 'a # the old code'\n",
			0,
		},
		{
			"shell backslash is no escape in single quotes",
			"bash",
			"echo 'a\\' # the old code\n",
			1,
		},
		{
			"shell dollar quote",
			"zsh",
			"x=$'a # the old code'\n",
			0,
		},
		{
			"sql bit literal then string",
			"sql",
			"SELECT B'0101', 'x' -- it's the old code\n",
			1,
		},
		{
			"sql hex literal",
			"sql",
			"SELECT X'2D2D' || 'a' -- ok\n",
			0,
		},
		{
			"sql bit literal hides dashes",
			"sql",
			"SELECT B'01 -- the old code'\n",
			0,
		},
		{
			"shell escaped quote then quote hides a hash",
			"sh",
			"echo 'a\\'  ' # the old code'\n",
			0,
		},
		{"sql backslash quote then dashes", "sql", "SELECT 'a\\' ' -- the old code'\n", 0},
		{"shell apostrophe word then comment", "sh", "echo it's # the old code\n", 1},
	}
}

func cLikeCases() []scanCase {
	return slices.Concat(cLikeBasicCases(), cLikeExtendedCases())
}

func cLikeBasicCases() []scanCase {
	return []scanCase{
		{
			"dart raw string ends at a backslash quote",
			"dart",
			"var x = r'C:\\'; // the old code\n",
			1,
		},
		{
			"c char literal after a case keyword",
			"c",
			"case' ': // it's the old code\n",
			1,
		},
		{
			"c char literal after a paren",
			"c",
			"f('\\''); // the old code\n",
			1,
		},
		{
			"c wide char prefix",
			"cpp",
			"x = L'a'; // it's the old code\n",
			1,
		},
		{
			"c wide string hides slashes",
			"cpp",
			"x = u8'a // the old code';\n",
			0,
		},
		{
			"js return glued to a string",
			"js",
			"return'x' // it's the old code\n",
			1,
		},
		{
			"js return glued string hides slashes",
			"ts",
			"return'a // the old code'\n",
			0,
		},
	}
}

func cLikeExtendedCases() []scanCase {
	return []scanCase{
		{
			"js equals glued string hides slashes",
			"ts",
			"x='a // the old code'\n",
			0,
		},
		{
			"jsx apostrophe in text",
			"tsx",
			"return <p>Don't touch</p>; {/* it's the old code */}\n",
			1,
		},
		{
			"jsx text after an element",
			"jsx",
			"<p>Don't</p> // the old code\n",
			1,
		},
		{
			"ts template with an escaped backtick",
			"ts",
			"s = `a \\` b`; // the old code\n",
			1,
		},
		{
			"dart raw string backslash quote then slashes",
			"dart",
			"var s = r'C:\\' ' // the old code';\n",
			0,
		},
		{
			"dart unclosed raw string then comment",
			"dart",
			"var s = r'abc; // the old code\n",
			1,
		},
		{
			"cpp digit separator then comment",
			"cpp",
			"int x = 1'000; // it's the old code\n",
			1,
		},
	}
}

func rustCases() []scanCase {
	return []scanCase{
		{
			"rust byte raw string keeps a backslash",
			"rs",
			"x = br\"a\\\" // x the old code\n",
			1,
		},
		{
			"rust unclosed raw string hides the rest",
			"rs",
			"let s = r#\"a // the old code\n",
			0,
		},
		{"rust byte raw string", "rs", "let s = br\"a // the old code\";\n", 0},
		{"rust c raw string", "rs", "let s = cr#\"a\" // the old code\"#;\n", 0},
		{"rust byte raw then comment", "rs", "let s = br\"a\"; // the old code\n", 1},
		{
			"rust static bound",
			"rs",
			"fn f<T: 'static>(x: T) { // it's the old code\n",
			1,
		},
		{
			"rust dyn with a static bound",
			"rs",
			"let b: Box<dyn Fn() + 'static> = x; // it's the old code\n",
			1,
		},
		{
			"rust loop label",
			"rs",
			"'outer: loop { // it's the old code\n",
			1,
		},
		{
			"rust where clause lifetime",
			"rs",
			"where T: 'a, // we used to call it's helper\n",
			1,
		},
		{
			"rust lifetime in generics",
			"rs",
			"fn f<'a>(x: &str) { // it's the old code\n",
			1,
		},
		{
			"rust lifetime reference",
			"rs",
			"fn f(x: &'a str) -> &'a str { // it's the old code\n",
			1,
		},
		{
			"rust escaped quote hides slashes",
			"rs",
			"let s = \"a\\\" // the old code\";\n",
			0,
		},
	}
}
