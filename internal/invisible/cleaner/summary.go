package cleaner

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/markdown"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

type Tally struct {
	Acted      int
	Remaining  int
	Held       int
	Removable  int
	Unreadable int
}

func (tally *Tally) Add(report Report) {
	tally.Acted += len(report.Acted)
	if report.Skipped() {
		tally.Unreadable += len(report.Remaining)
		return
	}
	tally.Remaining += len(report.Remaining)
	tally.Held += len(report.Held)
	tally.Removable += report.Removable()
}

const (
	unreadable = "in files clean cannot read: fix their syntax first"
	heldNote   = "where clean does not edit (string literals or possible code)"
)

func (report Report) HeldNote(found invisible.Mark) string {
	switch {
	case !report.IsHeld(found):
		return ""
	case markdown.IsMarkdown(filepath.Ext(report.Path)):
		return "in possible code: judge it by hand; clean leaves it"
	case report.literals.Raw(found.Offset):
		return "in a raw string literal: move it into an interpreted string " +
			"as an escape (\\uXXXX); clean leaves it"
	}
	return "in a string literal: write it as an escape (\\uXXXX); clean leaves it"
}

func (tally Tally) WriteLine() string {
	line := fmt.Sprintf(
		"removed %d carrier(s); %d marker(s) left for a human",
		tally.Acted, tally.Remaining+tally.Unreadable,
	)
	if tally.Held > 0 {
		line += fmt.Sprintf("; %d of them left %s", tally.Held, heldNote)
	}
	if tally.Unreadable > 0 {
		line += fmt.Sprintf("; %d of them %s", tally.Unreadable, unreadable)
	}
	return line
}

func (tally Tally) ReportLine() string {
	line := fmt.Sprintf("%d marker(s) found", tally.Remaining)
	if tally.Removable > 0 {
		line += "; re-run with --write to remove the removable ones"
	}
	if tally.Held > 0 {
		line += fmt.Sprintf("; %d left %s", tally.Held, heldNote)
	}
	if tally.Unreadable > 0 {
		line += fmt.Sprintf("; %d more %s", tally.Unreadable, unreadable)
	}
	return line
}

func markerLines(marks []invisible.Mark) string {
	const shown = 8
	var lines printable.LineList
	for _, found := range marks {
		lines.Add(found.Line)
	}
	text := "markers on " + lines.Label(shown)
	if lines.Extra(shown) > 0 {
		text += ", ..."
	}
	return text
}
