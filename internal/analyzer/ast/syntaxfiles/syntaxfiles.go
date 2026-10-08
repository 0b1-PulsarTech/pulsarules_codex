package syntaxfiles

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func Each(
	ctx *pass.Context,
	check func(fc pass.FileChange, file *syntax.File) []finding.Finding,
) []finding.Finding {
	var findings []finding.Finding
	for _, fc := range ctx.ChangedFiles {
		if ctx.Skips.SkipsPath(fc.Path) {
			continue
		}
		file, ok := ctx.Syntax.Get(fc.Path)
		if !ok || file.ParseError != "" {
			continue
		}
		findings = append(findings, check(fc, file)...)
	}
	return findings
}
