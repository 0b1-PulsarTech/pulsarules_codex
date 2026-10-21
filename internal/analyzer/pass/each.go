package pass

import (
	"go/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func (ctx *Context) EachSource(
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

func (ctx *Context) EachGoFile(
	check func(fc FileChange, file *ast.File) []finding.Finding,
) []finding.Finding {
	var findings []finding.Finding
	for fc, file := range ctx.ChangedGoASTs() {
		findings = append(findings, check(fc, file)...)
	}
	return findings
}

func (ctx *Context) EachSyntaxFile(
	check func(fc FileChange, file *syntax.File) []finding.Finding,
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
