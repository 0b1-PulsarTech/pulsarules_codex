package cleaner

import (
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/markdown"
)

func protection(path, src string, literals language.Literals) func(offset int) bool {
	ext := filepath.Ext(path)
	code := markdown.CodeIn(ext, src)
	// Whatever might be code is left alone: rewriting shown code is the one unsafe mistake.
	// Literals are empty for Markdown and code regions are empty elsewhere, so one holds at a time.
	return func(offset int) bool { return literals.Contains(offset) || code.Possible(offset) }
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
		if found.Class.Removable() && protected(found.Offset) {
			report.Held = append(report.Held, found)
		}
	}
	return report
}
