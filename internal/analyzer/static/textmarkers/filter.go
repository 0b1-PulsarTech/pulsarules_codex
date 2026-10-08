package textmarkers

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func eachMarkedFile(
	ctx *core.AnalysisContext,
	eligible func(core.FileChange) bool,
	check func(core.FileChange, string) []core.Finding,
) []core.Finding {
	// EachChangedFile skips testdata paths, so a dirty fixture cannot block its own fix.
	return core.EachChangedFile(
		ctx, eligible, func(fc core.FileChange, src []byte) []core.Finding {
			return check(fc, string(src))
		},
	)
}

func markedExtensions() []string {
	// Markdown too: an invisible carrier is as much a defect in a README as in code.
	return append(core.DefaultSourceExtensions(), ".md")
}
