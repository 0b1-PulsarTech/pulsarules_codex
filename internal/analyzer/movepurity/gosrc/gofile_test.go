package gosrc

import (
	"slices"
	"testing"
)

func parseFixture(t *testing.T) File {
	t.Helper()

	const src = "//go:build linux\n\n// Package p does x.\npackage p // import \"m/p\"\n\n" +
		"// preamble\nimport \"C\"\nimport a \"m/a\"\n\n" +
		"func F(x int) int { return x + a.G() + H() }\n"
	file, err := Parse(src)
	if err != nil {
		t.Fatalf("parseGoFile: %v", err)
	}
	return file
}

func TestParseGoFile(t *testing.T) {
	t.Parallel()

	file := parseFixture(t)
	if file.Pkg != "p" {
		t.Fatalf("pkg = %q, want p", file.Pkg)
	}
	if len(file.preamble) != 2 || file.docStart < 0 || file.docEnd <= file.docStart {
		t.Fatalf("preamble = %+v doc [%d,%d), want two comments and a doc range",
			file.preamble, file.docStart, file.docEnd)
	}
	if want := []string{"// preamble"}; !slices.Equal(file.Header, want) {
		t.Fatalf("header = %q, want %q (import comment dropped)", file.Header, want)
	}
	wantImports := []ImportSpec{{"", "C"}, {"a", "m/a"}}
	if !slices.Equal(file.Imports, wantImports) {
		t.Fatalf("imports = %+v, want %+v", file.Imports, wantImports)
	}
}

func TestParseGoFile_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
	}{
		{"no package clause", "func F() {}\n"},
		{"syntax error", "package p\n\nfunc F( {\n"},
		{"late import", "package p\n\nfunc F() {}\n\nimport \"os\"\n"},
		{"empty", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if _, err := Parse(testCase.src); err == nil {
				t.Fatalf("parseGoFile(%q) succeeded, want an error", testCase.src)
			}
		})
	}
}

func mustParse(t *testing.T, src string) File {
	t.Helper()

	file, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	return file
}

func TestIsImportComment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		off  int
		lit  string
		want bool
	}{
		{
			"line comment after the clause",
			"package p // import \"m/p\"\n",
			10,
			"// import \"m/p\"",
			true,
		},
		{
			"block comment after the clause",
			"package p /* import \"m/p\" */\n",
			10,
			"/* import \"m/p\" */",
			true,
		},
		{"on the next line", "package p\n// import \"m/p\"\n", 10, "// import \"m/p\"", false},
		{"another comment", "package p // note\n", 10, "// note", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := isImportComment(
				testCase.src,
				9,
				testCase.off,
				testCase.lit,
			); got != testCase.want {
				t.Fatalf("isImportComment() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestParse_IgnoreConstraint(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		want bool
	}{
		{"before the clause", "//go:build ignore\n\npackage p\n", true},
		{"old syntax", "// +build ignore\n\npackage p\n", true},
		{"another tag", "//go:build linux\n\npackage p\n", false},
		{"a tag containing the word", "//go:build !ignored\n\npackage p\n", false},
		{"after the clause", "package p\n\n//go:build ignore\n\nfunc F() {}\n", false},
		{"in the doc after a constraint", "//go:build ignore\n\n// Package p.\npackage p\n", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := mustParse(t, testCase.src).IsIgnored; got != testCase.want {
				t.Fatalf("IsIgnored = %v, want %v", got, testCase.want)
			}
		})
	}
}
