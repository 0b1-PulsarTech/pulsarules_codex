package textmarkers

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func eachMarkedFile(
	ctx *core.AnalysisContext,
	check func(core.FileChange, string) []core.Finding,
) []core.Finding {
	scanned := map[string]bool{".go": true, ".md": true}
	eligible := func(fc core.FileChange) bool { return scanned[strings.ToLower(fc.Extension)] }
	return core.EachChangedFile(ctx, eligible, func(
		fc core.FileChange, src []byte,
	) []core.Finding {
		return check(fc, string(src))
	})
}
