package findings

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

type Report struct {
	Findings   []finding.Finding
	Suppressed int
}

func (report Report) HasErrors() bool {
	errors, _, _ := report.tally()
	return errors > 0
}

func (report Report) Summary() string {
	if len(report.Findings) == 0 {
		return "no findings" + report.suppressedTail()
	}
	errors, warnings, infos := report.tally()
	return fmt.Sprintf(
		"%d error(s), %d warning(s), %d info(s)%s",
		errors,
		warnings,
		infos,
		report.suppressedTail(),
	)
}

func (report Report) Suppression() string {
	if report.Suppressed == 0 {
		// An unfiltered run carries no "0 suppressed" tail.
		return ""
	}
	return fmt.Sprintf("%d suppressed in generated files", report.Suppressed)
}

func (report Report) suppressedTail() string {
	if note := report.Suppression(); note != "" {
		return ", " + note
	}
	return ""
}

func (report Report) tally() (errors, warnings, infos int) {
	for _, hit := range report.Findings {
		switch hit.Severity {
		case finding.SeverityError:
			errors++
		case finding.SeverityWarning:
			warnings++
		case finding.SeverityInfo:
			infos++
		}
	}
	return errors, warnings, infos
}
