package vacuousdoc

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// Analyzer reports a doc comment on a function whose whole body is one return
// of a literal or a field.
//
// why: at that shape the signature already says it. A branch, loop or composed
// call is out of scope - there a comment may carry a precondition.
type Analyzer struct{}

var vacuousDocReporter = core.NewReporter(
	"vacuous-doc",
	core.SeverityWarning,
	core.CatAST,
)

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) ID() string   { return "vacuous-doc" }
func (a *Analyzer) Name() string { return "Vacuous doc comment" }

func (a *Analyzer) Description() string {
	return "Reports a doc comment on a function that only returns a literal or a field"
}
func (a *Analyzer) Stage() core.StageID     { return core.StageAST }
func (a *Analyzer) Category() core.Category { return core.CatAST }

// Needs asks for nothing: the check reads the AST the cache already holds.
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{NeedsAST: true}
}

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	return core.RunPerGoFile(ctx, func(fc core.FileChange, f *ast.File) []core.Finding {
		return checkFile(ctx.ASTCache.FileSet(), fc, f)
	})
}

func checkFile(fset *token.FileSet, fc core.FileChange, f *ast.File) []core.Finding {
	var findings []core.Finding
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Doc == nil || !returnsOnlyLiteralOrField(fn) || earnsItsPlace(fn.Doc) {
			continue
		}
		findings = append(findings, vacuousDocReporter.At(
			fc.Path,
			fset.Position(fn.Doc.Pos()).Line,
			fmt.Sprintf("doc comment on %s restates a one-line accessor", fn.Name.Name),
			"delete it, or replace it with the constraint the signature cannot show",
		))
	}
	return findings
}

// why: the shape alone produced false positives on the two comments a human
// review had deliberately kept - a rationale marker, and a second sentence
// carrying a cross-analyzer invariant. Both say something the signature cannot,
// so the shape check needs this second gate before it accuses anything.
func earnsItsPlace(doc *ast.CommentGroup) bool {
	text := strings.ToLower(doc.Text())
	if strings.Contains(text, "why:") || strings.Contains(text, "simplification:") {
		return true
	}
	return sentenceCount(text) > 1
}

// sentenceCount counts terminators, ignoring the abbreviations that carry a dot
// mid-sentence and would otherwise read as an extra one.
func sentenceCount(text string) int {
	for _, abbrev := range []string{"e.g.", "i.e.", "etc.", "vs."} {
		text = strings.ReplaceAll(text, abbrev, "")
	}
	trimmed := strings.TrimSpace(text)
	count := strings.Count(text, ". ") + strings.Count(trimmed, ".\n")
	if strings.HasSuffix(trimmed, ".") {
		count++
	}
	return count
}

// returnsOnlyLiteralOrField reports whether fn's body is exactly one return of
// values that carry no logic of their own.
func returnsOnlyLiteralOrField(fn *ast.FuncDecl) bool {
	if fn.Body == nil || len(fn.Body.List) != 1 {
		return false
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) == 0 {
		return false
	}
	for _, result := range ret.Results {
		if !isPlainValue(result) {
			return false
		}
	}
	return true
}

// why: a selector is only plain when its base is a bare identifier - c.allowed
// counts, pkg.Fn().Field does not, because the call is behaviour a comment may
// legitimately describe.
func isPlainValue(expr ast.Expr) bool {
	switch value := expr.(type) {
	case *ast.BasicLit, *ast.Ident:
		return true
	case *ast.SelectorExpr:
		_, ok := value.X.(*ast.Ident)
		return ok
	case *ast.UnaryExpr:
		return isPlainValue(value.X)
	default:
		return false
	}
}
