package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/findings"
)

func TestReportGovernance(t *testing.T) {
	t.Parallel()

	warning := finding.Finding{
		AnalyzerID: "file-size", Severity: finding.SeverityWarning, Message: "too long",
		File: "a.go", Line: 3,
	}
	failure := finding.Finding{
		AnalyzerID: "commit-lint",
		Severity:   finding.SeverityError,
		Message:    "bad",
	}
	testCases := []struct {
		name    string
		report  findings.Report
		want    string
		wantErr error
	}{
		{
			name:   "nothing found",
			report: findings.Report{},
			want:   "governance: no findings\n",
		},
		{
			name:   "nothing found but generated files were filtered",
			report: findings.Report{Suppressed: 2},
			want:   "governance: no findings, 2 suppressed in generated files\n",
		},
		{
			name:   "warnings pass",
			report: findings.Report{Findings: []finding.Finding{warning}, Suppressed: 2},
			want: "[WARNING] file-size: too long (a.go:3)\n" +
				"\n0 error(s), 1 warning(s), 0 info(s), 2 suppressed in generated files\n",
		},
		{
			name:    "an error fails",
			report:  findings.Report{Findings: []finding.Finding{failure}},
			want:    "[ERROR] commit-lint: bad\n\n1 error(s), 0 warning(s), 0 info(s)\n",
			wantErr: command.ErrFindings,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var out strings.Builder
			err := governance.ReportGovernance(&out, testCase.report)
			if !errors.Is(err, testCase.wantErr) || out.String() != testCase.want {
				t.Fatalf("reportGovernance() = %q, %v, want %q, %v",
					out.String(), err, testCase.want, testCase.wantErr)
			}
		})
	}
}

func TestReportGovernance_SuppressionPhrase(t *testing.T) {
	t.Parallel()

	report := findings.Report{
		Findings: []finding.Finding{
			{AnalyzerID: "x", Severity: finding.SeverityInfo, Message: "m"},
		},
		Suppressed: 7,
	}
	var out strings.Builder
	if err := governance.ReportGovernance(&out, report); err != nil {
		t.Fatalf("reportGovernance: %v", err)
	}

	const phrase = "7 suppressed in generated files"
	if !strings.HasSuffix(out.String(), ", "+phrase+"\n") {
		t.Fatalf("cli %q must end with the phrase %q", out.String(), phrase)
	}
}
