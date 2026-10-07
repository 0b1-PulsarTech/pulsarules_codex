package gosyntax

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func blocks(fset *token.FileSet, file *ast.File) []syntax.Block {
	var found []syntax.Block
	visitor := goast.Visitor{Block: func(block *ast.BlockStmt, depth int) {
		found = append(found, syntax.Block{Line: lineOf(fset, block.Lbrace), Depth: depth})
	}}
	goast.WalkFile(file, visitor)
	return found
}
