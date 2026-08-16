package textmarkers

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func BenchmarkFrontmatterFindings(b *testing.B) {
	src := "---\ntitle: A page\nauthor: Someone\ndate: 2026-01-01\ntags: [a, b]\n" +
		"generator: Claude\nweight: 3\ndraft: false\n---\n\nbody\n"
	fc := core.FileChange{Path: "a.md", Extension: ".md"}
	reporter := core.NewReporter("ai-frontmatter", core.SeverityError, core.CatSyntax)

	b.ReportAllocs()
	for b.Loop() {
		frontmatterFindings(fc, src, reporter)
	}
}
