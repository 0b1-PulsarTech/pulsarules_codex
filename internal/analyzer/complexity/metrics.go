package complexity

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
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
	reporter finding.Reporter,
) []finding.Finding {
	const flagArgumentsRule = "flag-arguments"
	if fn.Type.Params == nil {
		return nil
	}
	var findings []finding.Finding
	for _, field := range fn.Type.Params.List {
		if !isBoolType(field.Type) {
			continue
		}
		for _, id := range field.Names {
			// Telling a default value from a mode switch needs data-flow analysis.
			if isValueArgName(id.Name) {
				continue
			}
			finding := reporter.At(
				fc.Path,
				fset.Position(id.NamePos).Line,
				fmt.Sprintf("flag argument %q in %s", id.Name, fn.Name.Name),
				"split into two functions or use a typed enum",
			)
			finding.RuleID = flagArgumentsRule
			findings = append(findings, finding)
		}
	}
	return findings
}

func isBoolType(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == "bool"
}
