package complexity

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

func (th thresholds) checkChanged(ctx *core.AnalysisContext, fc core.FileChange) []core.Finding {
	var neutral []syntax.Function
	if file, ok := ctx.Syntax.Get(fc.Path); ok && file.ParseError == "" {
		neutral = file.Functions
	}
	var tree *ast.File
	if fc.Extension == ".go" && ctx.ASTCache != nil {
		tree = ctx.ASTCache.Get(fc.Path)
	}
	if tree == nil {
		return th.checkFile(fc, &syntax.File{Functions: neutral})
	}
	decls := bodyDecls(tree)
	fset := ctx.ASTCache.FileSet()
	// Zipping needs each summarised function to pair with its declaration; otherwise
	// keep the two passes.
	if !pairedByLine(fset, neutral, decls) {
		findings := th.checkFile(fc, &syntax.File{Functions: neutral})
		return append(findings, th.checkGoFile(fset, fc, tree)...)
	}
	var findings []core.Finding
	for idx, fn := range neutral {
		findings = append(findings, th.checkFunction(fc, fn)...)
		findings = append(findings, th.checkGoFunc(fset, fc, decls[idx])...)
	}
	return findings
}

func pairedByLine(fset *token.FileSet, neutral []syntax.Function, decls []*ast.FuncDecl) bool {
	if len(decls) != len(neutral) {
		return false
	}
	for idx, fn := range neutral {
		if fn.Line != fset.Position(decls[idx].Pos()).Line {
			return false
		}
	}
	return true
}
