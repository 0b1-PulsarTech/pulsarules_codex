package testfunc

import (
	"go/ast"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

func TestTestingAlias(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   string
	}{
		{"plain import", "package p\nimport \"testing\"\n", "testing"},
		{"aliased import", "package p\nimport tst \"testing\"\n", "tst"},
		{"no testing import", "package p\nimport \"os\"\n", "testing"},
		{"no imports", "package p\n", "testing"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			file, err := cache.Parse("p.go", []byte(testCase.source))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got := testingAlias(file); got != testCase.want {
				t.Fatalf("testingAlias = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestTestingAliasNilFile(t *testing.T) {
	t.Parallel()

	var file *ast.File
	if got := testingAlias(file); got != "testing" {
		t.Fatalf("testingAlias(nil) = %q", got)
	}
}
