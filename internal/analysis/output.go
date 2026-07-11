package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// CountBySeverity returns the number of findings at each severity level.
func CountBySeverity(findings []core.Finding) (errors, warnings, infos int) {
	for _, f := range findings {
		switch f.Severity {
		case core.SeverityError:
			errors++
		case core.SeverityWarning:
			warnings++
		case core.SeverityInfo:
			infos++
		}
	}
	return
}
