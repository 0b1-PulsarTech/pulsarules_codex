package gosrc

import (
	"go/token"
	"slices"
	"testing"
)

func parseFixture(t *testing.T) File {
	t.Helper()

	const src = "//go:build linux\n\n// Package p does x.\npackage p // import \"m/p\"\n\n" +
		"// preamble\nimport \"C\"\nimport a \"m/a\"\n\nfunc F(x int) int { return x + " +
		"a.G() + H() }\n"
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

func TestParseGoFile_Body(t *testing.T) {
	t.Parallel()

	file := parseFixture(t)
	if file.body[0].kind != token.SEMICOLON || file.body[1].kind != token.FUNC {
		t.Fatalf(
			"body starts %+v, want the semicolon ending the imports, then func", file.body[:2],
		)
	}
}

func TestParseGoFile_UnresolvedNames(t *testing.T) {
	t.Parallel()

	file := parseFixture(t)
	unresolvedNames := map[string]bool{}
	for _, tok := range file.body {
		if tok.kind == token.IDENT && file.unresolved[tok.off] {
			unresolvedNames[tok.lit] = true
		}
	}
	for name, want := range map[string]bool{"a": true, "H": true, "x": false, "F": false} {
		if unresolvedNames[name] != want {
			t.Fatalf("unresolved[%s] = %v, want %v", name, unresolvedNames[name], want)
		}
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

func TestDropOptionalSemicolons(t *testing.T) {
	t.Parallel()

	semi := goToken{kind: token.SEMICOLON}
	ident := goToken{kind: token.IDENT, lit: "x"}
	testCases := []struct {
		name string
		body []goToken
		want int
	}{
		{"before a brace", []goToken{ident, semi, {kind: token.RBRACE}}, 2},
		{"before a paren", []goToken{ident, semi, {kind: token.RPAREN}}, 2},
		{"between statements", []goToken{ident, semi, ident}, 3},
		{"trailing", []goToken{ident, semi}, 2},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := len(dropOptionalSemicolons(testCase.body)); got != testCase.want {
				t.Fatalf("kept %d tokens, want %d", got, testCase.want)
			}
		})
	}
}

func TestParseGoFile_Keys(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		want []string
	}{
		{"map key", "package p\n\nvar M = map[string]int{Key: 1}\n", []string{"Key"}},
		{"array index key", "package p\n\nvar A = [...]int{Idx: 1}\n", []string{"Idx"}},
		{"struct field key", "package p\n\nvar S = T{Name: 1}\n", []string{"Name"}},
		{
			"local variable key",
			"package p\n\nfunc F() any {\n\tk := 1\n\treturn map[int]int{k: 1}\n}\n",
			nil,
		},
		{"value is not a key", "package p\n\nvar M = map[string]int{\"a\": Val}\n", nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file, err := Parse(testCase.src)
			if err != nil {
				t.Fatalf("parseGoFile: %v", err)
			}
			var got []string
			for _, tok := range file.body {
				if tok.kind == token.IDENT && file.keys[tok.off] {
					got = append(got, tok.lit)
				}
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("keys = %q, want %q", got, testCase.want)
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

func TestParse_ImportCommentOnNextLine(t *testing.T) {
	t.Parallel()

	sameLine := mustParse(t, "package p // import \"m/p\"\n\nfunc F() {}\n")
	nextLine := mustParse(t, "package p\n// import \"m/p\"\n\nfunc F() {}\n")
	if len(nextLine.body) <= len(sameLine.body) {
		t.Fatalf("body = %d tokens, want more than %d: a next-line comment is content",
			len(nextLine.body), len(sameLine.body))
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
