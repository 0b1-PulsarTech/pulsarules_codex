package uninstall

import (
	"fmt"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/target"
)

// printReport writes a Strategy's progress notes to stdout and its warnings
// to stderr, keeping all uninstall output in the CLI layer, mirroring
// install's printReport.
func printReport(report target.Report) {
	for _, note := range report.Notes {
		_, _ = fmt.Println(note)
	}
	for _, warning := range report.Warnings {
		_, _ = fmt.Fprintln(os.Stderr, warning)
	}
}
