package report

import (
	"fmt"
	"io"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

type Report struct {
	Notes    []string
	Warnings []string
}

func (rep *Report) Note(format string, args ...any) {
	rep.Notes = append(rep.Notes, fmt.Sprintf(format, args...))
}

func (rep *Report) Warn(format string, args ...any) {
	rep.Warnings = append(rep.Warnings, fmt.Sprintf(format, args...))
}

func (rep *Report) Merge(sub Report) {
	rep.Notes = append(rep.Notes, sub.Notes...)
	rep.Warnings = append(rep.Warnings, sub.Warnings...)
}

func (rep Report) Print(stdout, stderr io.Writer) {
	for _, note := range rep.Notes {
		_, _ = fmt.Fprintln(stdout, printable.Line(note))
	}
	for _, warning := range rep.Warnings {
		_, _ = fmt.Fprintln(stderr, printable.Line(warning))
	}
}
