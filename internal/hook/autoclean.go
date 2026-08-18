package hook

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/clean"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/mark"
)

// autoCleanEdited removes the context-free carriers from the file just written,
// returning the text that announces it or "" when nothing changed.
// why: the only place a hook writes a project file. The announcement is not
// decoration - the agent's copy is now stale, and an Edit with a stale
// old_string would fail for a reason nothing explains.
func (d *Dispatcher) autoCleanEdited(filePath string) string {
	root := d.resolveProjectDir()
	if root == "" || filePath == "" {
		return ""
	}
	report, err := clean.New(root).CleanFile(filePath)
	if err != nil {
		d.logger.Debug("auto-clean skipped", "path", filePath, "error", err.Error())
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
		len(report.Acted), report.Path)
	for _, acted := range report.Acted {
		verb := "removed"
		if acted.Class == mark.ClassSpace {
			verb = "replaced with a space"
		}
		fmt.Fprintf(&out, "  line %d  U+%04X %s (%s)\n", acted.Line, acted.Rune, acted.Name, verb)
	}
	out.WriteString("The file on disk changed AFTER your write. Re-read it before your next edit.")
	return out.String()
}
