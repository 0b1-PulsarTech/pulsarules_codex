package doccomment

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
)

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
			sites := docSites(file.Decls[0], core.FileChange{}, file)
			if got := isDeprecatedOnly(sites[0].group); got != testCase.want {
				t.Fatalf("isDeprecatedOnly = %v, want %v", got, testCase.want)
			}
		})
	}
}
