package complexity

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func isValueArgName(name string) bool {
	switch name {
	case "fallback", "def", "defaultValue":
		return true
	}
	return false
}

func checkFlagArguments(
	fset *token.FileSet,
	fc pass.FileChange,
	fn *ast.FuncDecl,
	reporter pass.Reporter,
) []pass.Finding {
	if fn.Type.Params == nil {
		return nil
	}
	var findings []pass.Finding
	for _, field := range fn.Type.Params.List {
		if !isBoolType(field.Type) {
			continue
		}
		for _, id := range field.Names {
			// Telling a default value from a mode switch needs data-flow analysis.
			if isValueArgName(id.Name) {
				continue
			}
			findings = append(findings, reporter.At(
				fc.Path,
				fset.Position(id.NamePos).Line,
				fmt.Sprintf("flag argument %q in %s", id.Name, fn.Name.Name),
				"split into two functions or use a typed enum",
			))
		}
	}
	return findings
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

func isBoolType(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == "bool"
}

func countParams(fl *ast.FieldList) int {
	count := 0
	for _, field := range fl.List {
		if len(field.Names) > 0 {
			count += len(field.Names)
		} else {
			count++
		}
	}
	return count
}
