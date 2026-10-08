package pass

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"

func EachChangedFile(
	ctx *Context,
	eligible func(fc FileChange) bool,
	check func(fc FileChange, src []byte) []finding.Finding,
) []finding.Finding {
	if ctx.Sources == nil {
		return nil
	}
	var findings []finding.Finding
	for _, fc := range ctx.ChangedFiles {
		// Changed files arrive from git as whole paths, with no walk to prune them.
		if !eligible(fc) || ctx.Skips.SkipsPath(fc.Path) {
			continue
		}
		src, ok := ctx.Sources.Read(fc.Path)
		if !ok {
			continue
		}
		findings = append(findings, check(fc, src)...)
	}
	return findings
}
