package gonames

import (
	"go/ast"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

func collectNames(t *testing.T, content string) []syntax.Name {
	t.Helper()

	cache := astcache.New()
	file, err := cache.Parse("pkg.go", []byte(content))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return Collect(cache.FileSet(), file)
}

func parseFile(t *testing.T, content string) *ast.File {
	t.Helper()

	file, err := astcache.New().Parse("pkg.go", []byte(content))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return file
}

func TestCollect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
		want []syntax.Name
	}{
		{
			name: "package function and parameter",
			body: "package pkg\n\nfunc Run(val int) {}\n",
			want: []syntax.Name{
				{Name: "pkg", Kind: syntax.NamePackage, Line: 1},
				{Name: "Run", Kind: syntax.NameFunction, Line: 3, Exported: true},
				{Name: "val", Kind: syntax.NameParameter, Line: 3},
			},
		},
		{
			name: "testing parameter is exempt",
			body: "package pkg\n\nimport \"testing\"\n\nfunc TestRun(t *testing.T) {}\n",
			want: []syntax.Name{
				{Name: "pkg", Kind: syntax.NamePackage, Line: 1},
				{Name: "TestRun", Kind: syntax.NameFunction, Line: 5, Exported: true},
				{Name: "t", Kind: syntax.NameParameter, Line: 5, Exempt: true},
			},
		},
		{
			name: "upper-case local variable is exported",
			body: "package pkg\n\nvar Top = 1\n\nfunc run() { Local := 1; _ = Local }\n",
			want: []syntax.Name{
				{Name: "pkg", Kind: syntax.NamePackage, Line: 1},
				{Name: "Top", Kind: syntax.NameVariable, Line: 3, Exported: true},
				{Name: "run", Kind: syntax.NameFunction, Line: 5},
				{Name: "Local", Kind: syntax.NameVariable, Line: 5, Exported: true},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := collectNames(t, testCase.body)
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("names = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func TestIsExported(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		kind syntax.NameKind
		want bool
	}{
		{name: "upper function", text: "Run", kind: syntax.NameFunction, want: true},
		{name: "upper field", text: "Run", kind: syntax.NameField, want: true},
		{name: "lower function", text: "run", kind: syntax.NameFunction},
		{name: "upper parameter", text: "Run", kind: syntax.NameParameter},
		{name: "upper label", text: "Run", kind: syntax.NameLabel},
		{name: "empty", text: "", kind: syntax.NameType},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := isExported(testCase.text, testCase.kind); got != testCase.want {
				t.Fatalf("isExported = %v", got)
			}
		})
	}
}
