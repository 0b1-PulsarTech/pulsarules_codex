package install

import (
	"fmt"
	"io"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/selection"
)

func printWarning(out io.Writer, label string, err error) {
	_, _ = fmt.Fprintf(out, "warning: %s: %s\n", label, printable.Line(err.Error()))
}

func printDependencyPulls(out io.Writer, pulled []selection.DependencyPull) {
	for _, dep := range pulled {
		_, _ = fmt.Fprintf(
			out,
			"pulled in by dependency: %s (required by %s)\n",
			printable.Line(dep.Skill),
			printable.Line(dep.RequiredBy),
		)
	}
}
