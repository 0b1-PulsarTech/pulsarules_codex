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

func TestReport_Text(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		findings []finding.Finding
		layout   Layout
		header   string
		want     string
	}{
		{
			name:     "empty set renders nothing even with a header",
			findings: nil,
			layout:   CLI(),
			header:   "Governance checks:",
			want:     "",
		},
		{
			name:     "cli upcases the severity and appends the rule",
			findings: []finding.Finding{locatedFinding()},
			layout:   CLI(),
			want: "[WARNING] file-size: file is too long (internal/a.go:42)\n" +
				"  \u2192 split it\n" +
				"  rule: Keep functions short and files under one concept.\n",
		},
		{
			name:     "hook indents, downcases and drops the rule",
			findings: []finding.Finding{locatedFinding()},
			layout:   Hook(),
			want: "  [warning] file-size: file is too long (internal/a.go:42)\n" +
				"    \u2192 split it\n",
		},
		{
			name:     "a finding without a file omits the location",
			findings: []finding.Finding{unlocatedFinding()},
			layout:   CLI(),
			want:     "[ERROR] commit-lint: missing emoji\n",
		},
		{
			name:     "a file without a line omits the colon",
			findings: []finding.Finding{fileOnlyFinding()},
			layout:   CLI(),
			want:     "[INFO] imports: regroup imports (internal/b.go)\n",
		},
		{
			name:     "the header precedes the findings",
			findings: []finding.Finding{unlocatedFinding()},
			layout:   Hook(),
			header:   "Governance checks:",
			want:     "Governance checks:\n  [error] commit-lint: missing emoji\n",
		},
		{
			name:     "an all-blank rule body prints no rule line",
			findings: []finding.Finding{blankRuleFinding()},
			layout:   CLI(),
			want:     "[WARNING] naming: bad name\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := Report{Findings: testCase.findings}.Text(testCase.layout, testCase.header)
			if got != testCase.want {
				t.Fatalf("Report.Text() =\n%q\nwant\n%q", got, testCase.want)
			}
		})
	}
}

func TestReport_Text_Escapes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		findings []finding.Finding
		layout   Layout
		want     string
	}{
		{
			name: "a newline in the file is quoted, not rendered",
			findings: []finding.Finding{{
				AnalyzerID: "file-size", Severity: finding.SeverityWarning, Message: "m",
				File: "a\n  [error] x: y.go", Line: 2,
			}},
			layout: Hook(),
			want:   "  [warning] file-size: m (\"a\\n  [error] x: y.go\":2)\n",
		},
		{
			name: "a newline in a file without a line is quoted",
			findings: []finding.Finding{{
				AnalyzerID: "file-size", Severity: finding.SeverityWarning, Message: "m",
				File: "a\n  [error] x: y.go",
			}},
			layout: Hook(),
			want:   "  [warning] file-size: m (\"a\\n  [error] x: y.go\")\n",
		},
		{
			name: "a newline in the analyzer id is escaped",
			findings: []finding.Finding{
				{
					AnalyzerID: "golangci-lint/x\n  [error] y",
					Severity:   finding.SeverityWarning,
					Message:    "m",
				},
			},
			layout: Hook(),
			want:   "  [warning] golangci-lint/x\\n  [error] y: m\n",
		},
		{
			name: "a newline in message and suggestion is escaped",
			findings: []finding.Finding{{
				AnalyzerID: "move", Severity: finding.SeverityError,
				Message:    "rename a\n  [error] forged: x",
				Suggestion: "do\r\nthis",
			}},
			layout: Hook(),
			want: "  [error] move: rename a\\n  [error] forged: x\n" +
				"    \u2192 do\\r\\nthis\n",
		},
		{
			name: "a line break in the rule summary is escaped",
			findings: []finding.Finding{{
				AnalyzerID: "move", Severity: finding.SeverityWarning, Message: "m",
				RuleSummary: "Keep it\r[ERROR] forged\u2028[ERROR] again",
			}},
			layout: CLI(),
			want: "[WARNING] move: m\n" +
				"  rule: Keep it\\r[ERROR] forged\\u2028[ERROR] again\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := Report{Findings: testCase.findings}.Text(testCase.layout, "")
			if got != testCase.want {
				t.Fatalf("Report.Text() =\n%q\nwant\n%q", got, testCase.want)
			}
		})
	}
}

func TestReport_TextRendersEveryFinding(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{AnalyzerID: "one", Severity: finding.SeverityError, Message: "first"},
		{AnalyzerID: "two", Severity: finding.SeverityWarning, Message: "second"},
		{AnalyzerID: "three", Severity: finding.SeverityInfo, Message: "third"},
	}

	got := Report{Findings: findings}.Text(CLI(), "")
	if lines := strings.Count(got, "\n"); lines != len(findings) {
		t.Fatalf("rendered %d lines, want %d:\n%s", lines, len(findings), got)
	}
	for _, want := range []string{"[ERROR] one", "[WARNING] two", "[INFO] three"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}
