package character

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

func eachMarkedFile(
	ctx *pass.Context,
	eligible func(pass.FileChange) bool,
	check func(pass.FileChange, string) []finding.Finding,
) []finding.Finding {
	// EachChangedFile skips testdata paths, so a dirty fixture cannot block its own fix.
	return pass.EachChangedFile(
		ctx, eligible, func(fc pass.FileChange, src []byte) []finding.Finding {
			return check(fc, string(src))
		},
	)
}

func markedExtensions() []string {
	// Markdown too: an invisible carrier is as much a defect in a README as in code.
	return append(language.SourceExtensions(), ".md")
}
