package uninstall

import (
	"fmt"
	"io"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/target"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/escape"
)

func printReport(stdout, stderr io.Writer, report target.Report) {
	for _, note := range report.Notes {
		_, _ = fmt.Fprintln(stdout, escape.Line(note))
	}
	for _, warning := range report.Warnings {
		_, _ = fmt.Fprintln(stderr, escape.Line(warning))
	}
}
