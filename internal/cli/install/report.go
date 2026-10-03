package install

import (
	"fmt"
	"io"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/selection"
)

func printReport(stdout, stderr io.Writer, report host.Report) {
	for _, note := range report.Notes {
		_, _ = fmt.Fprintln(stdout, printable.Line(note))
	}
	for _, warning := range report.Warnings {
		_, _ = fmt.Fprintln(stderr, printable.Line(warning))
	}
}

func printWarning(out io.Writer, label string, err error) {
	_, _ = fmt.Fprintf(out, "warning: %s: %s\n", label, printable.Line(err.Error()))
}

func printDependencyPulls(out io.Writer, pulled []selection.DependencyPull) {
	for _, dep := range pulled {
		_, _ = fmt.Fprintf(
			out,
			"pulled in by dependency: %s (required by %s)\n",
			dep.Skill,
			dep.RequiredBy,
		)
	}
}
