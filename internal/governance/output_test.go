package governance

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

func TestCountBySeverity(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		{Severity: finding.SeverityError},
		{Severity: finding.SeverityError},
		{Severity: finding.SeverityWarning},
		{Severity: finding.SeverityInfo},
		{Severity: finding.SeverityInfo},
		{Severity: finding.SeverityInfo},
	}

	errors, warnings, infos := CountBySeverity(findings)
	if errors != 2 {
		t.Errorf("errors: got %d, want 2", errors)
	}
	if warnings != 1 {
		t.Errorf("warnings: got %d, want 1", warnings)
	}
	if infos != 3 {
		t.Errorf("infos: got %d, want 3", infos)
	}
}
