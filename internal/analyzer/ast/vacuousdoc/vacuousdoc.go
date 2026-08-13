package vacuousdoc

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: finding.NewReporter("vacuous-doc", finding.SeverityWarning, finding.CatAST),
	}
}

func (a *Analyzer) ID() string { return "vacuous-doc" }

func (a *Analyzer) Stage() finding.StageID { return finding.StageAST }

func (a *Analyzer) Needs() finding.Requirements {
	return finding.Requirements{NeedsAST: true}
}

func (a *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, f *ast.File) []finding.Finding {
		return checkFile(ctx.ASTCache.FileSet(), fc, f, a.reporter)
	})
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, f *ast.File, reporter finding.Reporter,
) []finding.Finding {
	var findings []finding.Finding
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Doc == nil || !returnsOnlyLiteralOrField(fn) || earnsItsPlace(fn.Doc) {
			continue
		}
		findings = append(findings, reporter.At(
			fc.Path,
			fset.Position(fn.Doc.Pos()).Line,
			fmt.Sprintf("doc comment on %s restates a one-line accessor", fn.Name.Name),
			"delete it, or replace it with the constraint the signature cannot show",
		))
	}
	return findings
}

func earnsItsPlace(doc *ast.CommentGroup) bool {
	text := strings.ToLower(doc.Text())
	if strings.Contains(text, "why:") || strings.Contains(text, "simplification:") {
		return true
	}
	return sentenceCount(text) > 1
}

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
