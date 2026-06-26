package install

import (
	"fmt"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/target"
)

// printReport writes a Strategy's progress notes to stdout and its warnings to
// stderr, keeping all install output in the CLI layer.
func printReport(report target.Report) {
	for _, note := range report.Notes {
		_, _ = fmt.Println(note)
	}
	for _, warning := range report.Warnings {
		_, _ = fmt.Fprintln(os.Stderr, warning)
	}
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
