package textmarkers

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// scanned is the file set both analyzers read.
var scanned = map[string]bool{".go": true, ".md": true}

// eachMarkedFile calls check for every eligible changed file's source text.
// The testdata skip core.EachChangedFile applies is what keeps a
// deliberately dirty fixture from tripping an error-severity finding that
// would block the very commit fixing it.
func eachMarkedFile(
	ctx *core.AnalysisContext,
	check func(core.FileChange, string) []core.Finding,
) []core.Finding {
	eligible := func(fc core.FileChange) bool { return scanned[strings.ToLower(fc.Extension)] }
	return core.EachChangedFile(ctx, eligible, func(fc core.FileChange, src []byte) []core.Finding {
		return check(fc, string(src))
	})
}
