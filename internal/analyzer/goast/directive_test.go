package goast

import (
	"go/ast"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

func docOf(t *testing.T, source string) *ast.CommentGroup {
	t.Helper()
	cache := astcache.New()
	file, err := cache.Parse("p.go", []byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("first decl is not a func")
	}
	return fn.Doc
}

func TestIsDirectiveGroup(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   bool
	}{
		{"no doc", "package p\n\nfunc f() {}\n", true},
		{"go directive", "package p\n\n//go:generate foo\nfunc f() {}\n", true},
		{"go noinline", "package p\n\n//go:noinline\nfunc f() {}\n", true},
		{"nolint directive", "package p\n\n//nolint:ireturn // reason\nfunc f() {}\n", true},
		{"bare nolint", "package p\n\n//nolint\nfunc f() {}\n", true},
		{"spaced nolint is prose", "package p\n\n// nolint:ireturn\nfunc f() {}\n", false},
		{"lint ignore", "package p\n\n//lint:ignore SA1019 reason\nfunc f() {}\n", true},
		{"goverter unspaced", "package p\n\n//goverter:converter\nfunc f() {}\n", true},
		{
			"goverter spaced",
			"package p\n\n// goverter:converter\n// goverter:output:file x.go\nfunc f() {}\n",
			true,
		},
		{"pulsarules spaced", "package p\n\n// pulsarules:process-lifetime\nfunc f() {}\n", true},
		{
			"pulsarules marker with its reason",
			"package p\n\n//pulsarules:process-lifetime the ticker\n" +
				"// lives for the process\nfunc f() {}\n",
			false,
		},
		{
			"spaced marker plus prose",
			"package p\n\n// pulsarules: x\n// prose\nfunc f() {}\n",
			false,
		},
		{
			"one line marker with reason",
			"package p\n\n//pulsarules:process-lifetime the ticker lives for the process\n" +
				"func f() {}\n",
			true,
		},
		{"todo shaped prose", "package p\n\n//todo:remove this hack\nfunc f() {}\n", false},
		{"empty block comment", "package p\n\n/**/\nfunc f() {}\n", true},
		{"empty star block comment is prose", "package p\n\n/** */\nfunc f() {}\n", false},
		{"stars line is prose", "package p\n\n// ***\nfunc f() {}\n", false},
		{"slash star slash is blank", "package p\n\n//*/\nfunc f() {}\n", true},
		{"spaced closer is blank", "package p\n\n// */\nfunc f() {}\n", true},
		{"ignore is prose", "package p\n\n// ignore: unused\nfunc f() {}\n", false},
		{"eslint is prose", "package p\n\n// eslint-foo\nfunc f() {}\n", false},
		{"export directive", "package p\n\n//export Foo\nfunc f() {}\n", true},
		{"line directive", "package p\n\n//line x.go:3\nfunc f() {}\n", true},
		{"prose", "package p\n\n// does a thing\nfunc f() {}\n", false},
		{
			"prose plus directive",
			"package p\n\n// does a thing\n//nolint:ireturn // r\nfunc f() {}\n",
			false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := IsDirectiveGroup(docOf(t, testCase.source)); got != testCase.want {
				t.Fatalf("IsDirectiveGroup = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestIsSimplificationGroup(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   bool
	}{
		{"no doc", "package p\n\nfunc f() {}\n", false},
		{
			"marker",
			"package p\n\n// simplification: caps at 7.\n// Upgrade path: x.\nfunc f() {}\n",
			true,
		},
		{"prose", "package p\n\n// does a thing\nfunc f() {}\n", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := IsSimplificationGroup(docOf(t, testCase.source)); got != testCase.want {
				t.Fatalf("IsSimplificationGroup = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestHintFor(t *testing.T) {
	t.Parallel()

	const mixedHint = "delete the prose lines, keep the directive lines"

	testCases := []struct {
		name   string
		source string
		want   string
	}{
		{"no doc", "package p\n\nfunc f() {}\n", "fallback"},
		{"prose only", "package p\n\n// does a thing\nfunc f() {}\n", "fallback"},
		{
			"prose with a blank line",
			"package p\n\n// does a thing\n//\nfunc f() {}\n",
			"fallback",
		},
		{
			"directive then prose",
			"package p\n\n//nolint:x // why\n// does a thing\nfunc f() {}\n",
			mixedHint,
		},
		{
			"directive, blank line, prose",
			"package p\n\n//nolint:x\n//\n// does a thing\nfunc f() {}\n",
			mixedHint,
		},
		{
			"empty block comment then prose",
			"package p\n\n/**/\n// does a thing\nfunc f() {}\n",
			"fallback",
		},
		{
			"directive group without prose",
			"package p\n\n//nolint:x\n//go:noinline\nfunc f() {}\n",
			"fallback",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := HintFor(docOf(t, testCase.source), "fallback"); got != testCase.want {
				t.Fatalf("HintFor = %q, want %q", got, testCase.want)
			}
		})
	}
}
