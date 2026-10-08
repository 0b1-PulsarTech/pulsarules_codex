package complexity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func checkFileCasesMagicNumbers() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "magic number detected",
			source: "package foo\n" +
				"func f() {\n" +
				"	x := 42\n" +
				"	_ = x\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "octal file mode literal not flagged as magic number",
			source: "package foo\n" +
				"func f() {\n" +
				"	x := 0o750\n" +
				"	_ = x\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "decimal multiple of eight still flagged as magic number",
			source: "package foo\n" +
				"func f() {\n" +
				"	x := 488\n" +
				"	_ = x\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "strconv ParseInt bit-size argument not flagged as magic number",
			source: "package foo\n" +
				"import \"strconv\"\n" +
				"func f() {\n" +
				"	x, _ := strconv.ParseInt(\"1\", 10, 64)\n" +
				"	_ = x\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "aliased strconv import bit-size argument not flagged as magic number",
			source: "package foo\n" +
				"import conv \"strconv\"\n" +
				"func f() {\n" +
				"	x, _ := conv.ParseInt(\"1\", 10, 64)\n" +
				"	_ = x\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "local value named strconv does not exempt its last argument",
			source: "package foo\n" +
				"import \"strconv\"\n" +
				"func f(strconv parser) {\n" +
				"	x, _ := strconv.ParseInt(\"1\", 10, 77)\n" +
				"	_ = x\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "strconv ParseFloat bit-size argument not flagged as magic number",
			source: "package foo\n" +
				"import \"strconv\"\n" +
				"func f() {\n" +
				"	x, _ := strconv.ParseFloat(\"1\", 64)\n" +
				"	_ = x\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "bare sixty-four in a business expression still flagged as magic number",
			source: "package foo\n" +
				"func f(x int) int {\n" +
				"	return x * 64\n" +
				"}\n",
			expect: 1,
		},
	}
}

func TestIsOctalModeLiteral(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		val  string
		want bool
	}{
		{"lowercase o prefix", "0o750", true},
		{"uppercase o prefix", "0O750", true},
		{"legacy octal form not matched", "0750", false},
		{"decimal not matched", "750", false},
		{"decimal divisible by eight not matched", "488", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := isOctalModeLiteral(testCase.val); got != testCase.want {
				t.Fatalf("isOctalModeLiteral(%q) = %v, want %v", testCase.val, got, testCase.want)
			}
		})
	}
}

func TestFindMagicNumbers_SkipsTestFiles(t *testing.T) {
	t.Parallel()

	const source = "package foo\n" +
		"func f() {\n" +
		"	x := 42\n" +
		"	_ = x\n" +
		"}\n"

	testCases := []struct {
		name   string
		isTest bool
		want   int
	}{
		{name: "production file still flags", want: 1},
		{name: "test file is exempt", isTest: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "foo.go", source, 0)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			fn, ok := file.Decls[0].(*ast.FuncDecl)
			if !ok {
				t.Fatalf("decl 0 = %T, want *ast.FuncDecl", file.Decls[0])
			}
			fc := pass.FileChange{Path: "foo.go", Extension: ".go", IsTest: testCase.isTest}
			reporter := finding.NewReporter("complexity", finding.SeverityInfo, finding.CatAST)

			if got := findMagicNumbers(fset, fc, file, fn, reporter); len(got) != testCase.want {
				t.Errorf("findMagicNumbers = %+v, want %d finding(s)", got, testCase.want)
			}
		})
	}
}

func TestFindMagicNumbers_ReportsEveryLiteral(t *testing.T) {
	t.Parallel()

	const source = "package foo\n" +
		"func f() {\n" +
		"	x := 42\n" +
		"	y := 77\n" +
		"	z := 91\n" +
		"	_, _, _ = x, y, z\n" +
		"}\n"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", source, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("decl 0 = %T, want *ast.FuncDecl", file.Decls[0])
	}
	reporter := finding.NewReporter("complexity", finding.SeverityInfo, finding.CatAST)

	got := findMagicNumbers(
		fset, pass.FileChange{Path: "foo.go", Extension: ".go"}, file, fn, reporter,
	)
	if len(got) != 3 {
		t.Fatalf("findMagicNumbers = %+v, want one finding per literal", got)
	}
	for idx, want := range []string{"42", "77", "91"} {
		if !strings.Contains(got[idx].Message, want) {
			t.Errorf("finding %d = %q, want it to name %s", idx, got[idx].Message, want)
		}
	}
}
