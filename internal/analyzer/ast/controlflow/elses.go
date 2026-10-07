package controlflow

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/gosyntax"
)

func checkElses(
	fset *token.FileSet,
	fc core.FileChange,
	file *ast.File,
	reporter core.Reporter,
) map[int][]core.Finding {
	findings := map[int][]core.Finding{}
	blockIdx := 0
	emit := func(pos token.Pos, msg, suggestion string) {
		findings[blockIdx] = append(
			findings[blockIdx],
			reporter.At(fc.Path, fset.Position(pos).Line, msg, suggestion),
		)
	}
	// An else finding fires just before its block is entered, so blockIdx is that block's index.
	gosyntax.WalkFile(file, gosyntax.Visitor{
		Block: func(*ast.BlockStmt, int) { blockIdx++ },
		Else:  func(block *ast.BlockStmt) { checkElseBlock(block, emit) },
	})
	return findings
}

func checkElseBlock(
	block *ast.BlockStmt,
	emit func(token.Pos, string, string),
) {
	for _, stmt := range block.List {
		switch typed := stmt.(type) {
		case *ast.ReturnStmt:
			emit(
				typed.Return,
				"else block contains return; use early return instead",
				"remove else and return directly",
			)
			return
		case *ast.ExprStmt:
			if call, ok := typed.X.(*ast.CallExpr); ok {
				if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
					emit(
						typed.Pos(),
						"else block contains panic; panic directly instead",
						"remove else and panic directly",
					)
					return
				}
			}
		case *ast.ForStmt, *ast.RangeStmt:
			emit(
				typed.Pos(),
				"else block contains loop; restructure to avoid else",
				"move loop out of else or invert the condition",
			)
			return
		case *ast.BranchStmt:
			emit(
				typed.Pos(),
				fmt.Sprintf("else block contains %s statement", typed.Tok),
				"restructure to avoid branch inside else",
			)
			return
		}
	}
}
