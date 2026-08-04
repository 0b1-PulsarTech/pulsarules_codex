package naming

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestFileIndexTypeDerived pins the exemption that clears the `base` reports:
// a name that repeats the type it was built from is precise, not vague.
func TestFileIndexTypeDerived(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		ident  string
		// nth selects which binding of ident to judge, 1-based; 0 means the
		// first one.
		nth  int
		want bool
	}{
		{
			name:   "constructor result named after its type",
			source: "package foo\nfunc f(conn int) { base := NewBase(conn); _ = base }\n",
			ident:  "base",
			want:   true,
		},
		{
			name:   "qualified constructor result",
			source: "package foo\nfunc f(conn int) { base := dbtx.NewBase(conn); _ = base }\n",
			ident:  "base",
			want:   true,
		},
		{
			name:   "composite literal named after its type",
			source: "package foo\nfunc f() { manager := pkg.Manager{}; _ = manager }\n",
			ident:  "manager",
			want:   true,
		},
		{
			name:   "parameter named after a generic pointer type",
			source: "package foo\nfunc New(base *dbtx.Base[Q]) {}\n",
			ident:  "base",
			want:   true,
		},
		{
			name:   "a type declaration does not exempt its own name",
			source: "package foo\ntype Data struct{}\n",
			ident:  "Data",
			want:   false,
		},
		{
			name: "the exemption is per binding site, not per file",
			source: "package foo\n" +
				"func f() { manager := NewManager(); _ = manager }\n" +
				"func g() { manager := find(); _ = manager }\n",
			ident: "manager",
			// The SECOND manager, bound from an unrelated call, stays noise
			// even though the first one earned an exemption.
			nth:  2,
			want: false,
		},
		{
			name:   "arithmetic result borrows no type",
			source: "package foo\nfunc f(minor, n int) { base := minor / n; _ = base }\n",
			ident:  "base",
			want:   false,
		},
		{
			name:   "unrelated call result",
			source: "package foo\nfunc f() { manager := find(); _ = manager }\n",
			ident:  "manager",
			want:   false,
		},
		{
			name:   "var spec with an explicit type",
			source: "package foo\nvar data pkg.Data\n",
			ident:  "data",
			want:   true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			f := parseSource(t, testCase.source)
			idx := newFileIndex(f)
			pos := identPos(t, f, testCase.ident, testCase.nth)
			if got := idx.typeDerived[pos]; got != testCase.want {
				t.Fatalf("typeDerived for %q = %v, want %v", testCase.ident, got, testCase.want)
			}
		})
	}
}

func TestHasNumberedSibling(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		ident  string
		stem   string
		want   bool
	}{
		{
			name:   "bare stem present",
			source: "package foo\nvar user string\nvar user1 string\n",
			ident:  "user1",
			stem:   "user",
			want:   true,
		},
		{
			name:   "another low counter present",
			source: "package foo\nvar step1 string\nvar step2 string\n",
			ident:  "step1",
			stem:   "step",
			want:   true,
		},
		{
			name:   "alone in the file",
			source: "package foo\nvar user1 string\n",
			ident:  "user1",
			stem:   "user",
			want:   false,
		},
		{
			name:   "a semantic sibling is not a counter",
			source: "package foo\nvar sha1 string\nvar sha256 string\n",
			ident:  "sha1",
			stem:   "sha",
			want:   false,
		},
		{
			name:   "a different stem is not a sibling",
			source: "package foo\nvar user1 string\nvar order2 string\n",
			ident:  "user1",
			stem:   "user",
			want:   false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			idx := indexSource(t, testCase.source)
			got := idx.hasNumberedSibling(testCase.ident, testCase.stem)
			if got != testCase.want {
				t.Fatalf(
					"hasNumberedSibling(%q, %q) = %v, want %v",
					testCase.ident, testCase.stem, got, testCase.want,
				)
			}
		})
	}
}

func indexSource(t *testing.T, source string) *fileIndex {
	t.Helper()
	return newFileIndex(parseSource(t, source))
}

func parseSource(t *testing.T, source string) *ast.File {
	t.Helper()

	f, err := parser.ParseFile(token.NewFileSet(), "foo.go", source, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return f
}

// identPos finds the nth (1-based; 0 means first) occurrence of name, so a
// case can pin one specific binding rather than the whole file's namespace.
func identPos(t *testing.T, f *ast.File, name string, nth int) token.Pos {
	t.Helper()

	seen := 0
	var found token.Pos
	ast.Inspect(f, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok || id.Name != name || found != token.NoPos {
			return true
		}
		seen++
		if seen >= max(nth, 1) {
			found = id.NamePos
		}
		return true
	})
	if found == token.NoPos {
		t.Fatalf("identifier %q occurrence %d not found", name, nth)
	}
	return found
}
