package character

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

func eachMarkedFile(
	ctx *pass.Context,
	analyzerID string,
	check func(pass.FileChange, string) []pass.Finding,
) []pass.Finding {
	scanned := append(pass.DefaultSourceExtensions(), ".md")
	eligible := pass.ExtensionFilter(ctx, analyzerID, scanned)
	return pass.EachChangedFile(ctx, eligible, func(
		fc pass.FileChange, src []byte,
	) []pass.Finding {
		return check(fc, string(src))
	})
}
