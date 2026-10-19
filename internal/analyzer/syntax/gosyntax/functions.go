package gosyntax

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func lineOf(fset *token.FileSet, pos token.Pos) int {
	return fset.Position(pos).Line
}

func functions(fset *token.FileSet, file *ast.File) []syntax.Function {
	var found []syntax.Function
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		found = append(found, summarizeFunc(fset, fn))
	}
	return found
}

func summarizeFunc(fset *token.FileSet, fn *ast.FuncDecl) syntax.Function {
	kind := syntax.FuncFunction
	if fn.Recv != nil {
		kind = syntax.FuncMethod
	}
	summary := syntax.Function{
		Name:       fn.Name.Name,
		Kind:       kind,
		Line:       lineOf(fset, fn.Pos()),
		BodyStart:  lineOf(fset, fn.Body.Lbrace),
		BodyEnd:    lineOf(fset, fn.Body.Rbrace),
		Complexity: cyclomaticComplexity(fn),
		Params:     goast.FieldCount(fn.Type.Params),
	}
	return summary
}
