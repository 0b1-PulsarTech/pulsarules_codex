package clean

import (
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/langspec"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/mark"
)

func protection(path, src string, literals langspec.Literals) func(offset int) bool {
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		return literals.Contains
	}
	// Whatever might be code is left alone: rewriting shown code is the one unsafe mistake.
	return mark.CodeRegions(src).Possible
}

func rescan(report Report, text string) Report {
	return settle(report, text, langspec.ScanLiterals(filepath.Ext(report.Path), text))
}

func settle(report Report, text string, literals langspec.Literals) Report {
	report.Remaining = mark.Scan(text)
	report.literals = literals
	protected := protection(report.Path, text, literals)
	report.Held = nil
	for _, found := range report.Remaining {
		if (found.Class == mark.ClassStrip || found.Class == mark.ClassSpace) &&
			protected(found.Offset) {
			report.Held = append(report.Held, found)
		}
	}
	return report
}
