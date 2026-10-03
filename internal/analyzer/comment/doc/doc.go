package doc

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast/testfunc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) ID() string { return "doc-comment" }

func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (analyzer *Analyzer) reporter() pass.Reporter {
	return pass.NewReporter(analyzer.ID(), pass.SeverityWarning, pass.CatAST)
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter(), ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []pass.Finding {
		if fc.Generated {
			return nil
		}
		return checkFile(ctx.ASTCache.FileSet(), fc, file, reporter)
	})
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, file *ast.File, reporter pass.Reporter,
) []pass.Finding {
	var findings []pass.Finding
	for _, decl := range file.Decls {
		for _, site := range docSites(decl, fc, file) {
			if site.group == nil || isExemptDoc(site.group) {
				continue
			}
			findings = append(findings, reporter.At(
				fc.Path,
				fset.Position(site.group.Pos()).Line,
				fmt.Sprintf("%s carries a doc comment", site.symbol),
				goast.HintFor(site.group, site.hint),
			))
		}
	}
	return findings
}

func isExemptDoc(group *ast.CommentGroup) bool {
	return goast.IsDirectiveGroup(group) ||
		goast.IsSimplificationGroup(group) ||
		isDeprecatedOnly(group)
}

func docSites(decl ast.Decl, fc pass.FileChange, file *ast.File) []docSite {
	const funcHint = "delete it; keep a non-obvious why on the line it explains inside the " +
		"body, never as a multi-line block at the top of the body"
	switch node := decl.(type) {
	case *ast.FuncDecl:
		if testfunc.Covers(fc, file, node) {
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
