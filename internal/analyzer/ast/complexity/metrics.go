package complexity

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// valueArgNames are bool parameter names that name a returned DEFAULT
// VALUE, not a mode switch (e.g. ParamSet.Bool(key string, fallback bool)).
// Telling the two apart needs data-flow analysis this AST-only pass lacks,
// so a bool param named in this small set is assumed a value and unreported.
var valueArgNames = map[string]bool{
	"fallback":     true,
	"def":          true,
	"defaultValue": true,
}

// checkFlagArguments reports bool parameters that select between two
// behaviors. It excludes bool parameters named for a default VALUE (see
// valueArgNames) rather than a behavior switch.
func checkFlagArguments(
	fset *token.FileSet,
	fc core.FileChange,
	fn *ast.FuncDecl,
	reporter core.Reporter,
) []core.Finding {
	if fn.Type.Params == nil {
		return nil
	}
	var findings []core.Finding
	for _, p := range fn.Type.Params.List {
		if !isBoolType(p.Type) {
			continue
		}
		for _, id := range p.Names {
			if valueArgNames[id.Name] {
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
	c := 1
	ast.Inspect(fn, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.IfStmt:
			c++
		case *ast.ForStmt:
			c++
		case *ast.RangeStmt:
			c++
		case *ast.CaseClause:
			if v.List != nil {
				c++
			}
		case *ast.CommClause:
			c++
		case *ast.BinaryExpr:
			if v.Op == token.LAND || v.Op == token.LOR {
				c++
			}
		}
		return true
	})
	return c
}

func isBoolType(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == "bool"
}

func countParams(fl *ast.FieldList) int {
	n := 0
	for _, p := range fl.List {
		if len(p.Names) > 0 {
			n += len(p.Names)
		} else {
			n++
		}
	}
	return n
}
