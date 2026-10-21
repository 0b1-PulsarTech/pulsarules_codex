package findings

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

func severities(errors, warnings, infos int) []finding.Finding {
	out := make([]finding.Finding, 0, errors+warnings+infos)
	for range errors {
		out = append(out, finding.Finding{Severity: finding.SeverityError})
	}
	for range warnings {
		out = append(out, finding.Finding{Severity: finding.SeverityWarning})
	}
	for range infos {
		out = append(out, finding.Finding{Severity: finding.SeverityInfo})
	}
	return out
}

func TestReport_Summary(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		report Report
		want   string
	}{
		{name: "nothing at all", report: Report{}, want: "no findings"},
		{
			name:   "nothing reported but some suppressed",
			report: Report{Suppressed: 3},
			want:   "no findings, 3 suppressed in generated files",
		},
		{
			name:   "counts every severity",
			report: Report{Findings: severities(2, 1, 3)},
			want:   "2 error(s), 1 warning(s), 3 info(s)",
		},
		{
			name:   "suppressed count is appended in full",
			report: Report{Findings: severities(0, 1, 0), Suppressed: 353},
			want:   "0 error(s), 1 warning(s), 0 info(s), 353 suppressed in generated files",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.report.Summary(); got != testCase.want {
				t.Errorf("Summary() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestReport_Suppression(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		suppressed int
		want       string
	}{
		{name: "nothing suppressed says nothing", suppressed: 0, want: ""},
		{name: "one is still announced", suppressed: 1, want: "1 suppressed in generated files"},
		{
			name:       "the count is stated in full",
			suppressed: 353,
			want:       "353 suppressed in generated files",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := (Report{Suppressed: testCase.suppressed}).Suppression(); got != testCase.want {
				t.Errorf("Suppression() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestReport_HasErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		findings []finding.Finding
		want     bool
	}{
		{name: "empty", findings: nil, want: false},
		{name: "warnings and infos are not errors", findings: severities(0, 2, 2), want: false},
		{name: "one error is enough", findings: severities(1, 2, 2), want: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := (Report{Findings: testCase.findings}).HasErrors(); got != testCase.want {
				t.Errorf("HasErrors() = %v, want %v", got, testCase.want)
			}
		})
	}
}
