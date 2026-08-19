package character

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func eachMarkedFile(
	ctx *pass.Context,
	check func(pass.FileChange, string) []pass.Finding,
) []pass.Finding {
	scanned := map[string]bool{".go": true, ".md": true}
	eligible := func(fc pass.FileChange) bool { return scanned[strings.ToLower(fc.Extension)] }
	return pass.EachChangedFile(ctx, eligible, func(
		fc pass.FileChange, src []byte,
	) []pass.Finding {
		return check(fc, string(src))
	})
}
