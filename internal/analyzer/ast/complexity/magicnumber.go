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
// Upgrade path: revisit if struct-tag or fixture literals start producing
// real false negatives; walk into those subtrees instead of skipping them.
func findMagicNumbers(
	fset *token.FileSet,
	fc core.FileChange,
	fn *ast.FuncDecl,
	reporter core.Reporter,
) []core.Finding {
	var findings []core.Finding
	fired := false
	// exemptPos records literals whose ROLE, not their value, makes them
	// self-documenting (e.g. a stdlib bit-size argument). ast.Inspect visits
	// a *ast.CallExpr before it descends into that call's argument nodes, so
	// markStdlibBitSizeArg always runs before the literal itself is visited.
	exemptPos := map[token.Pos]bool{}

	ast.Inspect(fn, func(n ast.Node) bool {
		// skip const blocks - their literals are intentional
		if gen, ok := n.(*ast.GenDecl); ok && gen.Tok == token.CONST {
			return false
		}
		if call, ok := n.(*ast.CallExpr); ok {
			markStdlibBitSizeArg(call, exemptPos)
			return true
		}
		lit, ok := n.(*ast.BasicLit)
		if !ok || (lit.Kind != token.INT && lit.Kind != token.FLOAT) {
			return true
		}
		if exemptPos[lit.Pos()] || isSkippableLiteralValue(lit.Value) || fired {
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

// isSkippableLiteralValue reports whether val is a literal this analyzer
// treats as self-documenting rather than magic: 0/1/-1 (too common to be
// magic), a common boundary value, or a 0o-prefixed octal file-mode literal
// (the form Go requires for a readable permission constant - a plain decimal
// divisible by 8 is NOT covered by this).
func isSkippableLiteralValue(val string) bool {
	switch val {
	case "0", "1", "-1", "0.0", "1.0",
		"2", "10", "60", "100", "1000", "3600", "86400",
		"255", "256", "1024", "1025":
		return true
	}
	return isOctalModeLiteral(val)
}

// isOctalModeLiteral reports whether val is the `0o`/`0O`-prefixed octal
// literal form, the form Go requires for a readable file-permission
// constant. It does not match the legacy `0750` octal form or any decimal
// literal, so a decimal number that happens to be divisible by 8 is still
// flagged as a magic number.
func isOctalModeLiteral(val string) bool {
	return strings.HasPrefix(val, "0o") || strings.HasPrefix(val, "0O")
}

// strconvBitSizeFuncs are the strconv parse functions whose final parameter
// is the integer/float bit size being parsed into (e.g. 32, 64).
var strconvBitSizeFuncs = map[string]bool{
	"ParseInt":   true,
	"ParseUint":  true,
	"ParseFloat": true,
}

// markStdlibBitSizeArg exempts call's last argument for
// strconv.ParseInt/ParseUint/ParseFloat: its value is mandated by the parse
// target's width, not a code choice, so naming it a constant would obscure
// the call. It is POSITION-aware, not value-aware - only that argument slot
// is exempt; the same literal alone in a business expression still flags.
func markStdlibBitSizeArg(call *ast.CallExpr, exempt map[token.Pos]bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "strconv" || !strconvBitSizeFuncs[sel.Sel.Name] {
		return
	}
	if len(call.Args) == 0 {
		return
	}
	if lit, ok := call.Args[len(call.Args)-1].(*ast.BasicLit); ok {
		exempt[lit.Pos()] = true
	}
}
