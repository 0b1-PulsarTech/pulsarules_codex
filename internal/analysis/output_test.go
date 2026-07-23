package analysis

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func TestCountBySeverity(t *testing.T) {
	t.Parallel()

	findings := []core.Finding{
		{Severity: core.SeverityError},
		{Severity: core.SeverityError},
		{Severity: core.SeverityWarning},
		{Severity: core.SeverityInfo},
		{Severity: core.SeverityInfo},
		{Severity: core.SeverityInfo},
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
