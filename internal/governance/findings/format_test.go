package findings

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

func locatedFinding() finding.Finding {
	return finding.Finding{
		AnalyzerID:  "file-size",
		Severity:    finding.SeverityWarning,
		Message:     "file is too long",
		File:        "internal/a.go",
		Line:        42,
		Suggestion:  "split it",
		RuleSummary: "Keep functions short and files under one concept.",
	}
}

func blankRuleFinding() finding.Finding {
	return finding.Finding{
		AnalyzerID:  "naming",
		Severity:    finding.SeverityWarning,
		Message:     "bad name",
		RuleSummary: "\n   \n\n",
	}
}

func unlocatedFinding() finding.Finding {
	return finding.Finding{
		AnalyzerID: "commit-lint",
		Severity:   finding.SeverityError,
		Message:    "missing emoji",
	}
}

func fileOnlyFinding() finding.Finding {
	return finding.Finding{
		AnalyzerID: "imports",
		Severity:   finding.SeverityInfo,
		Message:    "regroup imports",
		File:       "internal/b.go",
	}
}

func TestFormatFindings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		findings []finding.Finding
		style    FindingStyle
		header   string
		want     string
	}{
		{
			name:     "empty set renders nothing even with a header",
			findings: nil,
			style:    StyleCLI,
			header:   "Governance checks:",
			want:     "",
		},
		{
			name:     "cli upcases the severity and appends the rule",
			findings: []finding.Finding{locatedFinding()},
			style:    StyleCLI,
			want: "[WARN] file-size: file is too long (internal/a.go:42)\n" +
				"  \u2192 split it\n" +
				"  rule: Keep functions short and files under one concept.\n",
		},
		{
			name:     "hook indents, downcases and drops the rule",
			findings: []finding.Finding{locatedFinding()},
			style:    StyleHook,
			want: "  [warn] file-size: file is too long (internal/a.go:42)\n" +
				"    \u2192 split it\n",
		},
		{
			name:     "a finding without a file omits the location",
			findings: []finding.Finding{unlocatedFinding()},
			style:    StyleCLI,
			want:     "[ERROR] commit-lint: missing emoji\n",
		},
		{
			name:     "a file without a line omits the colon",
			findings: []finding.Finding{fileOnlyFinding()},
			style:    StyleCLI,
			want:     "[INFO] imports: regroup imports (internal/b.go)\n",
		},
		{
			name:     "the header precedes the findings",
			findings: []finding.Finding{unlocatedFinding()},
			style:    StyleHook,
			header:   "Governance checks:",
			want:     "Governance checks:\n  [error] commit-lint: missing emoji\n",
		},
		{
			name:     "an all-blank rule body prints no rule line",
			findings: []finding.Finding{blankRuleFinding()},
			style:    StyleCLI,
			want:     "[WARN] naming: bad name\n",
		},
		{
			name:     "an unknown style falls back to the cli layout",
			findings: []finding.Finding{unlocatedFinding()},
			style:    FindingStyle("nonsense"),
			want:     "[ERROR] commit-lint: missing emoji\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := FormatFindings(testCase.findings, testCase.style, testCase.header)
			if got != testCase.want {
				t.Fatalf("FormatFindings() =\n%q\nwant\n%q", got, testCase.want)
			}
		})
	}
}

func TestFormatFindingsRendersEveryFinding(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{AnalyzerID: "one", Severity: finding.SeverityError, Message: "first"},
		{AnalyzerID: "two", Severity: finding.SeverityWarning, Message: "second"},
		{AnalyzerID: "three", Severity: finding.SeverityInfo, Message: "third"},
	}

	got := FormatFindings(findings, StyleCLI, "")
	if lines := strings.Count(got, "\n"); lines != len(findings) {
		t.Fatalf("rendered %d lines, want %d:\n%s", lines, len(findings), got)
	}
	for _, want := range []string{"[ERROR] one", "[WARN] two", "[INFO] three"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}
