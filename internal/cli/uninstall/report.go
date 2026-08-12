package uninstall

import (
	"fmt"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
)

func printReport(report host.Report) {
	for _, note := range report.Notes {
		_, _ = fmt.Println(note)
	}
	for _, warning := range report.Warnings {
		_, _ = fmt.Fprintln(os.Stderr, warning)
	}
}
