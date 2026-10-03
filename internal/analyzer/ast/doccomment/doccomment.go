package doccomment

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/directive"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/gotest"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) ID() string { return "doc-comment" }

func (analyzer *Analyzer) Stage() core.StageID { return core.StageAST }

func (analyzer *Analyzer) reporter() core.Reporter {
	return core.NewReporter(analyzer.ID(), core.SeverityWarning, core.CatAST)
}

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporter := analyzer.reporter().Resolved(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, file *ast.File) []core.Finding {
		if fc.Generated {
			return nil
		}
		return checkFile(ctx.ASTCache.FileSet(), fc, file, reporter)
	})
}

func checkFile(
	fset *token.FileSet, fc core.FileChange, file *ast.File, reporter core.Reporter,
) []core.Finding {
	var findings []core.Finding
	for _, decl := range file.Decls {
		for _, site := range docSites(decl, fc, file) {
			if site.group == nil || isExemptDoc(site.group) {
				continue
			}
			findings = append(findings, reporter.At(
				fc.Path,
				fset.Position(site.group.Pos()).Line,
				fmt.Sprintf("%s carries a doc comment", site.symbol),
				directive.HintFor(site.group, site.hint),
			))
		}
	}
	return findings
}

func isExemptDoc(group *ast.CommentGroup) bool {
	return directive.IsDirectiveGroup(group) ||
		directive.IsSimplificationGroup(group) ||
		isDeprecatedOnly(group)
}

func docSites(decl ast.Decl, fc core.FileChange, file *ast.File) []docSite {
	const funcHint = "delete it; keep a non-obvious why on the line it explains inside the " +
		"body, never as a multi-line block at the top of the body"
	switch node := decl.(type) {
	case *ast.FuncDecl:
		if gotest.CoversFunc(fc, file, node) {
			return nil
		}
		return []docSite{{symbol: node.Name.Name, group: node.Doc, hint: funcHint}}
	case *ast.GenDecl:
		if node.Tok != token.TYPE {
			return nil
		}
		return typeSites(node)
	}
	return nil
}
