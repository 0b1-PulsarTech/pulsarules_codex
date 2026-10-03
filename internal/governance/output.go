package governance

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

func CountBySeverity(findings []pass.Finding) (errors, warnings, infos int) {
	for _, finding := range findings {
		switch finding.Severity {
		case pass.SeverityError:
			errors++
		case pass.SeverityWarning:
			warnings++
		case pass.SeverityInfo:
			infos++
		}
	}
	return
}
