package goast

import (
	"go/ast"
	"slices"
	"testing"
)

func TestInterfaceMethodSites(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name:   "methods in order",
			source: "type I interface {\n\tfirst() (int, int)\n\tsecond()\n}",
			want:   []string{"first", "second"},
		},
		{
			name:   "embedded interface skipped",
			source: "type I interface {\n\tfmt.Stringer\n\tonly() int\n}",
			want:   []string{"only"},
		},
		{name: "empty", source: "type I interface{}"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var got []string
			ast.Inspect(parse(t, testCase.source), func(node ast.Node) bool {
				if iface, ok := node.(*ast.InterfaceType); ok {
					for _, site := range InterfaceMethodSites(iface) {
						got = append(got, site.Name)
					}
				}
				return true
			})
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("sites = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestFuncDeclSite(t *testing.T) {
	t.Parallel()

	fn := firstFunc(t, parse(t, "func run() (int, error) { return 0, nil }"))
	site := FuncDeclSite(fn)
	if site.Name != "run" || site.Pos != fn.Pos() || site.Results != fn.Type.Results {
		t.Fatalf("site = %+v, want run at %v with the declared results", site, fn.Pos())
	}
}
