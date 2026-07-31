package complexity

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// simplification: magic literal detection skips const-declared values by
// returning false for GenDecl/CONST subtrees. This misses numbers inside
// struct tags and some test fixtures, but those are rare in real code.
func findMagicNumbers(fset *token.FileSet, fc core.FileChange, fn *ast.FuncDecl) []core.Finding {
	var findings []core.Finding
	fired := false

	ast.Inspect(fn, func(n ast.Node) bool {
		// skip const blocks - their literals are intentional
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
		// skip 0/1/-1 (too common to be magic)
		if val == "0" || val == "1" || val == "-1" || val == "0.0" || val == "1.0" {
			return true
		}
		// skip common boundary values
		switch val {
		case "2", "10", "60", "100", "1000", "3600", "86400",
			"255", "256", "1024", "1025":
			return true
		}
		// skip 0o-prefixed octal literals: that syntax is how Go spells a
		// Unix file-permission mode (os.MkdirAll(dir, 0o750)), and the
		// literal form itself is self-documenting. A plain decimal number
		// that happens to be divisible by 8 is NOT exempted by this check.
		if isOctalModeLiteral(val) {
			return true
		}
		if fired {
			return true
		}
		fired = true
		findings = append(findings, complexityInfoReporter.At(
			fc.Path,
			fset.Position(lit.Pos()).Line,
			fmt.Sprintf("magic number %s in %s", val, fn.Name.Name),
			"assign to a named constant",
		))
		return true
	})
	return findings
}

// isOctalModeLiteral reports whether val is the `0o`/`0O`-prefixed octal
// literal form, the form Go requires for a readable file-permission
// constant. It does not match the legacy `0750` octal form or any decimal
// literal, so a decimal number that happens to be divisible by 8 is still
// flagged as a magic number.
func isOctalModeLiteral(val string) bool {
	return strings.HasPrefix(val, "0o") || strings.HasPrefix(val, "0O")
}
