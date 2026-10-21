package character

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestFrontmatterFindings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		extension string
		src       string
		wantLines []int
	}{
		{"provenance key", ".md", "---\ntitle: A\ngenerator: Claude\n---\n", []int{3}},
		{"clean frontmatter", ".md", "---\ntitle: A\n---\n", nil},
		{"not markdown", ".go", "---\ngenerator: Claude\n---\n", nil},
		{"no frontmatter", ".md", "generator: Claude\n", nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			fc := pass.FileChange{Path: "a" + testCase.extension, Extension: testCase.extension}
			reporter := finding.NewReporter(frontmatterID, finding.SeverityError, finding.CatSyntax)

			findings := frontmatterFindings(fc, testCase.src, reporter)

			if len(findings) != len(testCase.wantLines) {
				t.Fatalf("findings = %+v, want lines %v", findings, testCase.wantLines)
			}
			for index, finding := range findings {
				if finding.Line != testCase.wantLines[index] || finding.RuleID != "source-files" {
					t.Errorf("finding = %+v, want line %d rule source-files",
						finding, testCase.wantLines[index])
				}
			}
		})
	}
}

func BenchmarkFrontmatterFindings(b *testing.B) {
	src := "---\ntitle: A page\nauthor: Someone\ndate: 2026-01-01\ntags: [a, b]\n" +
		"generator: Claude\nweight: 3\ndraft: false\n---\n\nbody\n"
	fc := pass.FileChange{Path: "a.md", Extension: ".md"}
	reporter := finding.NewReporter(frontmatterID, finding.SeverityError, finding.CatSyntax)

	b.ReportAllocs()
	for b.Loop() {
		frontmatterFindings(fc, src, reporter)
	}
}
