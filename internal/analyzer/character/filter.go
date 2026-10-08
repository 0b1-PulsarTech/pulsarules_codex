package character

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func eachMarkedFile(
	ctx *pass.Context,
	analyzerID string,
	check func(pass.FileChange, string) []finding.Finding,
) []finding.Finding {
	scanned := append(pass.DefaultSourceExtensions(), ".md")
	eligible := pass.ExtensionFilter(ctx, analyzerID, scanned)
	return pass.EachChangedFile(ctx, eligible, func(
		fc pass.FileChange, src []byte,
	) []finding.Finding {
		return check(fc, string(src))
	})
}
