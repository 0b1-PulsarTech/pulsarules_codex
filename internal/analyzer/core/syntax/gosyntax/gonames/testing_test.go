package gonames

import (
	"go/ast"
	"slices"
	"testing"
)

func TestIsTestingType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		header string
		param  string
		want   bool
	}{
		{name: "T pointer", header: "import \"testing\"", param: "*testing.T", want: true},
		{name: "B pointer", header: "import \"testing\"", param: "*testing.B", want: true},
		{name: "F pointer", header: "import \"testing\"", param: "*testing.F", want: true},
		{name: "M pointer", header: "import \"testing\"", param: "*testing.M", want: true},
		{name: "TB value", header: "import \"testing\"", param: "testing.TB", want: true},
		{name: "T value", header: "import \"testing\"", param: "testing.T"},
		{name: "TB pointer", header: "import \"testing\"", param: "*testing.TB"},
		{name: "wrapped", header: "import \"testing\"", param: "(*testing.T)", want: true},
		{name: "other type", header: "import \"testing\"", param: "*testing.PB"},
		{name: "aliased", header: "import tt \"testing\"", param: "*tt.T", want: true},
		{name: "dot import", header: "import . \"testing\"", param: "*T", want: true},
		{name: "no import", header: "import \"os\"", param: "*testing.T"},
		{name: "plain ident without dot import", header: "import \"testing\"", param: "*T"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file := parseFile(t, "package pkg\n\n"+testCase.header+"\n\nvar _ func(val "+
				testCase.param+")\n")
			names := testingNames(file)
			spec, _ := file.Decls[1].(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
			funcType, _ := spec.Type.(*ast.FuncType)
			got := isTestingType(funcType.Params.List[0].Type, names)
			if got != testCase.want {
				t.Fatalf("isTestingType(%s) = %v", testCase.param, got)
			}
		})
	}
}

func exemptNames(t *testing.T, source string) []string {
	t.Helper()

	var exempt []string
	for _, name := range collectNames(t, source) {
		if name.Exempt {
			exempt = append(exempt, name.Name)
		}
	}
	return exempt
}

func TestCollect_ExemptsTestingParameters(t *testing.T) {
	t.Parallel()

	const testingImport = "package pkg\n\nimport \"testing\"\n\n"
	testCases := []struct {
		name   string
		source string
		want   []string
	}{
		{"testing T", testingImport + "func TestX(t *testing.T) {}\n", []string{"t"}},
		{"testing B", testingImport + "func Bench(b *testing.B) {}\n", []string{"b"}},
		{"testing F", testingImport + "func FuzzX(f *testing.F) {}\n", []string{"f"}},
		{"testing M", testingImport + "func TestMain(m *testing.M) {}\n", []string{"m"}},
		{"testing TB", testingImport + "func help(t testing.TB) {}\n", []string{"t"}},
		{
			"parenthesised pointer",
			testingImport + "func paren(t (*testing.T)) {}\n",
			[]string{"t"},
		},
		{
			"function literal parameter",
			testingImport + "var run = func(t *testing.T) {}\n",
			[]string{"t"},
		},
		{
			"aliased import",
			"package pkg\n\nimport tst \"testing\"\n\nfunc TestX(t *tst.T) {}\n",
			[]string{"t"},
		},
		{
			"dot import",
			"package pkg\n\nimport . \"testing\"\n\nfunc TestX(t *T) {}\n",
			[]string{"t"},
		},
		{
			"every name of a shared field",
			testingImport + "func run(a, b *testing.T) {}\n",
			[]string{"a", "b"},
		},
		{"T by value", testingImport + "func run(t testing.T) {}\n", nil},
		{"TB by pointer", testingImport + "func run(t *testing.TB) {}\n", nil},
		{"other testing type", testingImport + "func run(p *testing.PB) {}\n", nil},
		{"no testing import", "package pkg\n\nfunc run(t *testing.T) {}\n", nil},
		{"bare T without a dot import", testingImport + "func run(t *T) {}\n", nil},
		{"receiver of that type", testingImport + "func (t *testing.T) run() {}\n", nil},
		{"variable of that type", testingImport + "func run() { var t *testing.T; _ = t }\n", nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := exemptNames(t, testCase.source); !slices.Equal(got, testCase.want) {
				t.Fatalf("exempt names = %q, want %q", got, testCase.want)
			}
		})
	}
}
