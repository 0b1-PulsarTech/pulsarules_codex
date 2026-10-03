package hook

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/clean"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/escape"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/mark"
)

func (disp *Dispatcher) autoCleanEdited(filePath string) string {
	root := disp.resolveProjectDir()
	if root == "" || filePath == "" {
		return ""
	}
	report, err := clean.New(root).CleanFile(filePath)
	if err != nil {
		if disp.logger != nil {
			disp.logger.Debug("auto-clean skipped", "path", filePath, "error", err.Error())
		}
		return ""
	}
	if !report.Changed {
		return ""
	}
	return cleanNotice(report)
}

func cleanNotice(report clean.Report) string {
	var out strings.Builder
	fmt.Fprintf(&out, "pulsarules removed %d invisible marker(s) from %s:\n",
		len(report.Acted), escape.Path(report.Path))
	for _, acted := range report.Acted {
		verb := "removed"
		if acted.Class == mark.ClassSpace {
			verb = "replaced with a space"
		}
		fmt.Fprintf(&out, "  line %d  U+%04X %s (%s)\n", acted.Line, acted.Rune, acted.Name, verb)
	}
	out.WriteString("The file on disk changed after your write. Re-read it before your next edit.")
	return out.String()
}
