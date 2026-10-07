package gosyntax

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
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
	}
	if fn.Type.Params != nil {
		summary.Params = countParams(fn.Type.Params)
	}
	return summary
}

func cyclomaticComplexity(fn *ast.FuncDecl) int {
	score := 1
	ast.Inspect(fn, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.IfStmt:
			score++
		case *ast.ForStmt:
			score++
		case *ast.RangeStmt:
			score++
		case *ast.CaseClause:
			if typed.List != nil {
				score++
			}
		case *ast.CommClause:
			score++
		case *ast.BinaryExpr:
			if typed.Op == token.LAND || typed.Op == token.LOR {
				score++
			}
		}
		return true
	})
	return score
}

func countParams(fields *ast.FieldList) int {
	total := 0
	for _, field := range fields.List {
		if len(field.Names) > 0 {
			total += len(field.Names)
		} else {
			total++
		}
	}
	return total
}
