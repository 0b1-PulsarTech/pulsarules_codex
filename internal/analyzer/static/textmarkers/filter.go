package textmarkers

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func eachMarkedFile(
	ctx *core.AnalysisContext,
	analyzerID string,
	check func(core.FileChange, string) []core.Finding,
) []core.Finding {
	scanned := append(core.DefaultSourceExtensions(), ".md")
	eligible := core.ExtensionFilter(ctx, analyzerID, scanned)
	return core.EachChangedFile(ctx, eligible, func(
		fc core.FileChange, src []byte,
	) []core.Finding {
		return check(fc, string(src))
	})
}
