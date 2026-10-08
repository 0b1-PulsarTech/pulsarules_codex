package cleaner

import (
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/markdown"
)

func protection(path, src string, literals language.Literals) func(offset int) bool {
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		return literals.Contains
	}
	// Whatever might be code is left alone: rewriting shown code is the one unsafe mistake.
	return markdown.CodeRegions(src).Possible
}

func rescan(report Report, text string) Report {
	return settle(report, text, language.ScanLiterals(filepath.Ext(report.Path), text))
}

func settle(report Report, text string, literals language.Literals) Report {
	report.Remaining = invisible.Scan(text)
	report.literals = literals
	protected := protection(report.Path, text, literals)
	report.Held = nil
	for _, found := range report.Remaining {
		if (found.Class == invisible.ClassStrip || found.Class == invisible.ClassSpace) &&
			protected(found.Offset) {
			report.Held = append(report.Held, found)
		}
	}
	return report
}
