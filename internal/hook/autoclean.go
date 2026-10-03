package hook

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible/cleaner"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

func (d *Dispatcher) autoCleanEdited(filePath string) string {
	root := d.resolveProjectDir()
	if root == "" || filePath == "" {
		return ""
	}
	report, err := cleaner.New(root).CleanFile(filePath)
	if err != nil {
		d.logger.Debug("auto-clean skipped", "path", filePath, "error", err.Error())
		return ""
	}
	if !report.Changed {
		return ""
	}
	return cleanNotice(report)
}

func cleanNotice(report cleaner.Report) string {
	var out strings.Builder
	fmt.Fprintf(&out, "pulsarules removed %d invisible marker(s) from %s:\n",
		len(report.Acted), printable.Path(report.Path))
	for _, acted := range report.Acted {
		verb := "removed"
		if acted.Class == invisible.ClassSpace {
			verb = "replaced with a space"
		}
		fmt.Fprintf(&out, "  line %d  U+%04X %s (%s)\n", acted.Line, acted.Rune, acted.Name, verb)
	}
	out.WriteString("The file on disk changed after your write. Re-read it before your next edit.")
	return out.String()
}
