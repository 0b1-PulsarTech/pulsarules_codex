package doc

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

func TestCheckFileMessages(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		source     string
		wantSymbol string
		wantHint   string
	}{
		{
			"type group is labelled as a group",
			"package p\n\n// group doc\ntype (\n\tA int\n\tB int\n)\n",
			"type group carries a doc comment",
			"type name and fields",
		},
		{
			"single type uses its own name",
			"package p\n\n// S is a thing.\ntype S struct{}\n",
			"S carries a doc comment",
			"type name and fields",
		},
		{
			"interface method is qualified",
			"package p\n\ntype I interface {\n\t// M does.\n\tM()\n}\n",
			"I.M carries a doc comment",
			"method name and signature",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			file, err := cache.Parse("p.go", []byte(testCase.source))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			fc := pass.FileChange{Path: "p.go", Extension: ".go"}
			got := checkFile(cache.FileSet(), fc, file, NewAnalyzer().reporter())
			if len(got) == 0 {
				t.Fatal("no finding")
			}
			if got[0].Message != testCase.wantSymbol ||
				!strings.Contains(got[0].Suggestion, testCase.wantHint) {
				t.Fatalf("finding = %+v", got[0])
			}
		})
	}
}

func TestIsDeprecatedOnly(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   bool
	}{
		{"one line", "package p\n\n// Deprecated: use G.\nfunc F() {}\n", true},
		{"three lines", "package p\n\n// Deprecated: a\n// b\n// c\nfunc F() {}\n", true},
		{"four lines", "package p\n\n// Deprecated: a\n// b\n// c\n// d\nfunc F() {}\n", false},
		{"prose first", "package p\n\n// F does.\n// Deprecated: a\nfunc F() {}\n", false},
		{"second paragraph", "package p\n\n// Deprecated: a\n//\n// b\nfunc F() {}\n", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			file, err := cache.Parse("p.go", []byte(testCase.source))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			sites := docSites(file.Decls[0], pass.FileChange{}, file)
			if got := isDeprecatedOnly(sites[0].group); got != testCase.want {
				t.Fatalf("isDeprecatedOnly = %v, want %v", got, testCase.want)
			}
		})
	}
}

func deprecatedCases() []checkFileCase {
	return []checkFileCase{
		{
			"deprecated plus four lines is flagged",
			"p.go",
			"package p\n\n// Deprecated: use G.\n// a\n// b\n// c\nfunc F() {}\n",
			1,
		},
		{
			"deprecated spanning three lines is exempt",
			"p.go",
			"package p\n\n// Deprecated: use G,\n// because a\n// and b.\nfunc F() {}\n",
			0,
		},
	}
}
