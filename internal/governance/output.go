package governance

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"

func CountBySeverity(findings []finding.Finding) (errors, warnings, infos int) {
	for _, f := range findings {
		switch f.Severity {
		case finding.SeverityError:
			errors++
		case finding.SeverityWarning:
			warnings++
		case finding.SeverityInfo:
			infos++
		}
	}
	return
}
