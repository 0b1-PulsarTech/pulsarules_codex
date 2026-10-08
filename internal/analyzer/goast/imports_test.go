package goast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"slices"
	"testing"
)

func TestImportNames(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		path   string
		want   []string
	}{
		{name: "default name", source: "import \"time\"", path: "time", want: []string{"time"}},
		{name: "aliased", source: "import clock \"time\"", path: "time", want: []string{"clock"}},
		{name: "blank import binds nothing", source: "import _ \"time\"", path: "time"},
		{name: "dot import", source: "import . \"time\"", path: "time", want: []string{"."}},
		{
			name:   "last path element",
			source: "import \"runtime/pprof\"",
			path:   "runtime/pprof",
			want:   []string{"pprof"},
		},
		{name: "other package", source: "import \"fmt\"", path: "time"},
		{name: "no imports", path: "time"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file := parse(t, testCase.source)
			if got := ImportNames(file, testCase.path); !slices.Equal(got, testCase.want) {
				t.Fatalf("ImportNames = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestAllImportNames(t *testing.T) {
	t.Parallel()

	file := parse(t, "import (\n\t\"fmt\"\n\tclock \"time\"\n\t_ \"embed\"\n\t. \"os\"\n)")
	got := slices.Sorted(maps.Keys(AllImportNames(file)))
	if want := []string{"clock", "fmt"}; !slices.Equal(got, want) {
		t.Fatalf("AllImportNames = %v, want %v", got, want)
	}
}

func parse(t *testing.T, source string) *ast.File {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "x.go", "package x\n"+source+"\n", 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return file
}
