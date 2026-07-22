package complexity

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// simplification: magic literal detection skips const-declared values by returning false
// for GenDecl/CONST subtrees, missing struct-tag and fixture numbers. Upgrade path: walk
// into those subtrees instead, if those literals start producing real false negatives.
func findMagicNumbers(
	fset *token.FileSet,
	fc core.FileChange,
	fn *ast.FuncDecl,
	reporter core.Reporter,
) []core.Finding {
	var findings []core.Finding
	fired := false

	ast.Inspect(fn, func(node ast.Node) bool {
		// Skip const blocks: their literals are intentional.
		if gen, ok := node.(*ast.GenDecl); ok && gen.Tok == token.CONST {
			return false
		}
		lit, ok := node.(*ast.BasicLit)
		if !ok || (lit.Kind != token.INT && lit.Kind != token.FLOAT) {
			return true
		}
		if isSkippableLiteralValue(lit.Value) || fired {
			return true
		}
		fired = true
		findings = append(findings, reporter.At(
			fc.Path,
			fset.Position(lit.Pos()).Line,
			fmt.Sprintf("magic number %s in %s", lit.Value, fn.Name.Name),
			"assign to a named constant",
		))
		return true
	})
	return findings
}

func isSkippableLiteralValue(val string) bool {
	switch val {
	case "0", "1", "-1", "0.0", "1.0",
		"2", "10", "60", "100", "1000", "3600", "86400",
		"255", "256", "1024", "1025":
		return true
	}
	return isOctalModeLiteral(val)
}

func isOctalModeLiteral(val string) bool {
	return strings.HasPrefix(val, "0o") || strings.HasPrefix(val, "0O")
}
