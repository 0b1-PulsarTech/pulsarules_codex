package complexity

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

// simplification: magic literal detection skips const-declared values by returning false
// for GenDecl/CONST subtrees, missing struct-tag and fixture numbers. Upgrade path: walk
// into those subtrees instead, if those literals start producing real false negatives.
func findMagicNumbers(
	fset *token.FileSet,
	fc pass.FileChange,
	fn *ast.FuncDecl,
	reporter finding.Reporter,
) []finding.Finding {
	var findings []finding.Finding
	fired := false

	ast.Inspect(fn, func(n ast.Node) bool {
		// Skip const blocks: their literals are intentional.
		if gen, ok := n.(*ast.GenDecl); ok && gen.Tok == token.CONST {
			return false
		}
		lit, ok := n.(*ast.BasicLit)
		if !ok {
			return true
		}
		if lit.Kind != token.INT && lit.Kind != token.FLOAT {
			return true
		}
		val := lit.Value
		if val == "0" || val == "1" || val == "-1" || val == "0.0" || val == "1.0" {
			return true
		}
		switch val {
		case "2", "10", "60", "100", "1000", "3600", "86400",
			"255", "256", "1024", "1025":
			return true
		}
		if isOctalModeLiteral(val) {
			return true
		}
		if fired {
			return true
		}
		fired = true
		findings = append(findings, reporter.At(
			fc.Path,
			fset.Position(lit.Pos()).Line,
			fmt.Sprintf("magic number %s in %s", val, fn.Name.Name),
			"assign to a named constant",
		))
		return true
	})
	return findings
}

func isOctalModeLiteral(val string) bool {
	return strings.HasPrefix(val, "0o") || strings.HasPrefix(val, "0O")
}
