package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func CountBySeverity(findings []core.Finding) (errors, warnings, infos int) {
	for _, finding := range findings {
		switch finding.Severity {
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
