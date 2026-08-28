package pproflabel

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestPprofImportName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		source     string
		wantName   string
		wantImport bool
	}{
		{
			name:       "no import at all",
			source:     "package foo\nfunc run() {}\n",
			wantImport: false,
		},
		{
			name:       "a plain import uses the default pprof name",
			source:     "package foo\nimport \"runtime/pprof\"\nfunc run() { _ = pprof.Labels }\n",
			wantName:   "pprof",
			wantImport: true,
		},
		{
			name:       "an aliased import uses the alias",
			source:     "package foo\nimport p \"runtime/pprof\"\nfunc run() { _ = p.Labels }\n",
			wantName:   "p",
			wantImport: true,
		},
		{
			name:       "a blank import cannot be selected on",
			source:     "package foo\nimport _ \"runtime/pprof\"\nfunc run() {}\n",
			wantImport: false,
		},
		{
			name:       "an unrelated import is not runtime/pprof",
			source:     "package foo\nimport \"fmt\"\nfunc run() { fmt.Println() }\n",
			wantImport: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file := parseFile(t, testCase.source)
			name, ok := pprofImportName(file)
			if ok != testCase.wantImport {
				t.Fatalf("ok = %v, want %v", ok, testCase.wantImport)
			}
			if ok && name != testCase.wantName {
				t.Fatalf("name = %q, want %q", name, testCase.wantName)
			}
		})
	}
}

func TestIsNameShadowed(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   bool
	}{
		{
			name: "a := short assignment shadows the name",
			source: "package foo\n" +
				"func run() {\n" +
				"\tpprof := 1\n" +
				"\t_ = pprof\n" +
				"}\n",
			want: true,
		},
		{
			name: "a var declaration shadows the name",
			source: "package foo\n" +
				"func run() {\n" +
				"\tvar pprof int\n" +
				"\t_ = pprof\n" +
				"}\n",
			want: true,
		},
		{
			name: "a parameter shadows the name",
			source: "package foo\n" +
				"func run(pprof int) {\n" +
				"\t_ = pprof\n" +
				"}\n",
			want: true,
		},
		{
			name: "no shadowing declaration at all",
			source: "package foo\n" +
				"func run() {\n" +
				"\tworkNormally()\n" +
				"}\n" +
				"func workNormally() {}\n",
			want: false,
		},
		{
			name: "a shadow inside a nested func literal does not shadow this scope",
			source: "package foo\n" +
				"func run() {\n" +
				"\tgo func() {\n" +
				"\t\tpprof := 1\n" +
				"\t\t_ = pprof\n" +
				"\t}()\n" +
				"}\n",
			want: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			params, body := parseFirstFuncBody(t, testCase.source)
			got := isNameShadowed(params, body, "pprof")
			if got != testCase.want {
				t.Fatalf("isNameShadowed() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func parseFile(t *testing.T, source string) *ast.File {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func parseFirstFuncBody(t *testing.T, source string) (*ast.FieldList, *ast.BlockStmt) {
	t.Helper()

	file := parseFile(t, source)

	var params *ast.FieldList
	var body *ast.BlockStmt
	ast.Inspect(file, func(node ast.Node) bool {
		if body != nil {
			return false
		}
		if funcDecl, ok := node.(*ast.FuncDecl); ok {
			params = funcDecl.Type.Params
			body = funcDecl.Body
			return false
		}
		return true
	})
	if body == nil {
		t.Fatal("no function declaration found in source")
	}
	return params, body
}
