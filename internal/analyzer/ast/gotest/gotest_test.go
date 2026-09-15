package gotest

import (
	"go/ast"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
)

func lastFunc(t *testing.T, source string) *ast.FuncDecl {
	t.Helper()
	cache := astcache.New()
	file, err := cache.Parse("p_test.go", []byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fn, ok := file.Decls[len(file.Decls)-1].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("last decl is not a func")
	}
	return fn
}

func TestTestKind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   string
	}{
		{"test", "package p\nimport \"testing\"\nfunc TestA(t *testing.T) {}\n", "Test"},
		{"example", "package p\nfunc ExampleA() {}\n", KindExample},
		{
			"method is never runnable",
			"package p\nimport \"testing\"\nfunc (S) TestA(t *testing.T) {}\n",
			"",
		},
		{"helper is no kind", "package p\nimport \"testing\"\nfunc h(t *testing.T) {}\n", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			fn := lastFunc(t, testCase.source)
			if got, _ := TestKind(fn); got != testCase.want {
				t.Fatalf("TestKind = %q, want %q", got, testCase.want)
			}
		})
	}
}
