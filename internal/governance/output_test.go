package governance

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestCountBySeverity(t *testing.T) {
	t.Parallel()

	findings := []pass.Finding{
		{Severity: pass.SeverityError},
		{Severity: pass.SeverityError},
		{Severity: pass.SeverityWarning},
		{Severity: pass.SeverityInfo},
		{Severity: pass.SeverityInfo},
		{Severity: pass.SeverityInfo},
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
