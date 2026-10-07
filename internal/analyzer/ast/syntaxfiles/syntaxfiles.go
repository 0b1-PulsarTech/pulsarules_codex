package syntaxfiles

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

func Each(
	ctx *core.AnalysisContext,
	check func(fc core.FileChange, file *syntax.File) []core.Finding,
) []core.Finding {
	var findings []core.Finding
	for _, fc := range ctx.ChangedFiles {
		file, ok := ctx.Syntax.Get(fc.Path)
		if !ok || file.ParseError != "" {
			continue
		}
		findings = append(findings, check(fc, file)...)
	}
	return findings
}
