package character

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func BenchmarkFrontmatterFindings(b *testing.B) {
	src := "---\ntitle: A page\nauthor: Someone\ndate: 2026-01-01\ntags: [a, b]\n" +
		"generator: Claude\nweight: 3\ndraft: false\n---\n\nbody\n"
	fc := pass.FileChange{Path: "a.md", Extension: ".md"}
	reporter := finding.NewReporter(FrontmatterID, finding.SeverityError, finding.CatSyntax)

	b.ReportAllocs()
	for b.Loop() {
		frontmatterFindings(fc, src, reporter)
	}
}
