package install

import (
	"fmt"
	"io"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/output"
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

func printWarning(out io.Writer, label string, err error) {
	_, _ = fmt.Fprintf(out, "warning: %s: %s\n", label, escape.Line(err.Error()))
}

func printDependencyPulls(pulled []output.DependencyPull) {
	for _, dep := range pulled {
		_, _ = fmt.Printf(
			"pulled in by dependency: %s (required by %s)\n",
			dep.Skill,
			dep.RequiredBy,
		)
	}
}
