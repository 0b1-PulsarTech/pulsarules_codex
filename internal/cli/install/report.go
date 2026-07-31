package install

import (
	"fmt"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/selection"
)

func printReport(report host.Report) {
	for _, note := range report.Notes {
		_, _ = fmt.Println(note)
	}
	for _, warning := range report.Warnings {
		_, _ = fmt.Fprintln(os.Stderr, warning)
	}
}

func printDependencyPulls(pulled []selection.DependencyPull) {
	for _, dep := range pulled {
		_, _ = fmt.Printf(
			"pulled in by dependency: %s (required by %s)\n",
			dep.Skill,
			dep.RequiredBy,
		)
	}
}
