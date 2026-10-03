package findings

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func locatedFinding() pass.Finding {
	return pass.Finding{
		AnalyzerID:  "file-size",
		Severity:    pass.SeverityWarning,
		Message:     "file is too long",
		File:        "internal/a.go",
		Line:        42,
		Suggestion:  "split it",
		RuleSummary: "Keep functions short and files under one concept.",
	}
}

func blankRuleFinding() pass.Finding {
	return pass.Finding{
		AnalyzerID:  "naming",
		Severity:    pass.SeverityWarning,
		Message:     "bad name",
		RuleSummary: "\n   \n\n",
	}
}

func unlocatedFinding() pass.Finding {
	return pass.Finding{
		AnalyzerID: "commit-lint",
		Severity:   pass.SeverityError,
		Message:    "missing emoji",
	}
}

func fileOnlyFinding() pass.Finding {
	return pass.Finding{
		AnalyzerID: "imports",
		Severity:   pass.SeverityInfo,
		Message:    "regroup imports",
		File:       "internal/b.go",
	}
}

func TestFormatFindings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		findings []pass.Finding
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
			findings: []pass.Finding{locatedFinding()},
			style:    StyleCLI,
			want: "[WARNING] file-size: file is too long (internal/a.go:42)\n" +
				"  \u2192 split it\n" +
				"  rule: Keep functions short and files under one concept.\n",
		},
		{
			name:     "hook indents, downcases and drops the rule",
			findings: []pass.Finding{locatedFinding()},
			style:    StyleHook,
			want: "  [warning] file-size: file is too long (internal/a.go:42)\n" +
				"    \u2192 split it\n",
		},
		{
			name:     "a finding without a file omits the location",
			findings: []pass.Finding{unlocatedFinding()},
			style:    StyleCLI,
			want:     "[ERROR] commit-lint: missing emoji\n",
		},
		{
			name:     "a file without a line omits the colon",
			findings: []pass.Finding{fileOnlyFinding()},
			style:    StyleCLI,
			want:     "[INFO] imports: regroup imports (internal/b.go)\n",
		},
		{
			name:     "the header precedes the findings",
			findings: []pass.Finding{unlocatedFinding()},
			style:    StyleHook,
			header:   "Governance checks:",
			want:     "Governance checks:\n  [error] commit-lint: missing emoji\n",
		},
		{
			name:     "an all-blank rule body prints no rule line",
			findings: []pass.Finding{blankRuleFinding()},
			style:    StyleCLI,
			want:     "[WARNING] naming: bad name\n",
		},
		{
			name:     "an unknown style falls back to the cli layout",
			findings: []pass.Finding{unlocatedFinding()},
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

func TestFormatFindings_Escapes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		findings []pass.Finding
		style    FindingStyle
		want     string
	}{
		{
			name: "a newline in the file is quoted, not rendered",
			findings: []pass.Finding{{
				AnalyzerID: "file-size", Severity: pass.SeverityWarning, Message: "m",
				File: "a\n  [error] x: y.go", Line: 2,
			}},
			style: StyleHook,
			want:  "  [warning] file-size: m (\"a\\n  [error] x: y.go\":2)\n",
		},
		{
			name: "a newline in a file without a line is quoted",
			findings: []pass.Finding{{
				AnalyzerID: "file-size", Severity: pass.SeverityWarning, Message: "m",
				File: "a\n  [error] x: y.go",
			}},
			style: StyleHook,
			want:  "  [warning] file-size: m (\"a\\n  [error] x: y.go\")\n",
		},
		{
			name: "a newline in the analyzer id is escaped",
			findings: []pass.Finding{
				{
					AnalyzerID: "golangci-lint/x\n  [error] y",
					Severity:   pass.SeverityWarning,
					Message:    "m",
				},
			},
			style: StyleHook,
			want:  "  [warning] golangci-lint/x\\n  [error] y: m\n",
		},
		{
			name: "a newline in message and suggestion is escaped",
			findings: []pass.Finding{{
				AnalyzerID: "move", Severity: pass.SeverityError,
				Message:    "rename a\n  [error] forged: x",
				Suggestion: "do\r\nthis",
			}},
			style: StyleHook,
			want: "  [error] move: rename a\\n  [error] forged: x\n" +
				"    \u2192 do\\r\\nthis\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := FormatFindings(testCase.findings, testCase.style, "")
			if got != testCase.want {
				t.Fatalf("FormatFindings() =\n%q\nwant\n%q", got, testCase.want)
			}
		})
	}
}

func TestFormatFindingsRendersEveryFinding(t *testing.T) {
	t.Parallel()

	findings := []pass.Finding{
		{AnalyzerID: "one", Severity: pass.SeverityError, Message: "first"},
		{AnalyzerID: "two", Severity: pass.SeverityWarning, Message: "second"},
		{AnalyzerID: "three", Severity: pass.SeverityInfo, Message: "third"},
	}

	got := FormatFindings(findings, StyleCLI, "")
	if lines := strings.Count(got, "\n"); lines != len(findings) {
		t.Fatalf("rendered %d lines, want %d:\n%s", lines, len(findings), got)
	}
	for _, want := range []string{"[ERROR] one", "[WARNING] two", "[INFO] three"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}
