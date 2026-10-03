package uninstall

import (
	"fmt"
	"io"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

func printReport(stdout, stderr io.Writer, report host.Report) {
	for _, note := range report.Notes {
		_, _ = fmt.Fprintln(stdout, printable.Line(note))
	}
	for _, warning := range report.Warnings {
		_, _ = fmt.Fprintln(stderr, printable.Line(warning))
	}
}
