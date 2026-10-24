package complexity

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
)

func TestMagicNumberAnalyzer_Analyze(t *testing.T) {
	t.Parallel()

	testCases := append(checkFileCasesMagicNumbers(), checkFileTestCase{
		name:   "bodyless declaration has nothing to flag",
		source: "package foo\nfunc f() int\n",
		expect: 0,
	})
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := astcache.New()
			if _, err := cache.Parse("foo.go", []byte(testCase.source)); err != nil {
				t.Fatalf("parse: %v", err)
			}
			got := NewMagicNumberAnalyzer().Analyze(&pass.Context{
				ASTCache:     cache,
				ChangedFiles: []pass.FileChange{{Path: "foo.go", Extension: ".go"}},
			})
			if len(got) != testCase.expect {
				t.Fatalf("got %d findings, want %d: %v", len(got), testCase.expect, got)
			}
			for _, hit := range got {
				if hit.AnalyzerID != MagicID || hit.Severity != finding.SeverityInfo {
					t.Fatalf("finding = %+v, want id %q at info", hit, MagicID)
				}
			}
		})
	}
}

func TestMagicNumberAnalyzer_NoCache(t *testing.T) {
	t.Parallel()

	if got := NewMagicNumberAnalyzer().Analyze(&pass.Context{}); len(got) != 0 {
		t.Fatalf("findings = %v, want none without an AST cache", got)
	}
}

func TestDescribeMagicNumber(t *testing.T) {
	t.Parallel()

	got := DescribeMagicNumber()
	if got.ID != "magic-number" || got.Rule != "code-smells" {
		t.Fatalf("descriptor = %+v", got)
	}
}
