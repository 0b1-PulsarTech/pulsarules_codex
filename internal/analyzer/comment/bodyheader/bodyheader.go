package bodyheader

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast/testfunc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const (
	ID            = "body-header-comment"
	paramMinLines = "min_lines"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) ID() string { return ID }

func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageAST }

func (analyzer *Analyzer) reporter() finding.Reporter {
	return finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST)
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	const defaultMinLines = 2
	reporter := pass.ResolveReporter(analyzer.reporter(), ctx)
	minLines := ctx.Params(ID).Int(paramMinLines, defaultMinLines)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []finding.Finding {
		if fc.Generated {
			return nil
		}
		settings := check{fset: ctx.ASTCache.FileSet(), reporter: reporter, minLines: minLines}
		return settings.file(fc, file)
	})
}

type check struct {
	fset     *token.FileSet
	reporter finding.Reporter
	minLines int
}

func (settings check) file(fc pass.FileChange, file *ast.File) []finding.Finding {
	var findings []finding.Finding
	for _, site := range bodySites(fc, file) {
		groups := headerGroups(file.Comments, site.Body)
		lines := 0
		for _, group := range groups {
			lines += goast.ProseLines(group)
		}
		if lines < settings.minLines {
			continue
		}
		findings = append(findings, settings.reporter.At(
			fc.Path,
			settings.fset.Position(groups[0].Pos()).Line,
			fmt.Sprintf("%s opens with a %d-line comment block", site.Name, lines),
			"delete it; a removed doc comment is never moved into the body, "+
				"keep only a one-line why at the statement it explains",
		))
	}
	return findings
}

func bodySites(fc pass.FileChange, file *ast.File) []goast.FuncSite {
	var sites []goast.FuncSite
	ast.Inspect(file, func(node ast.Node) bool {
		switch fn := node.(type) {
		case *ast.FuncDecl:
			if testfunc.Covers(fc, file, fn) {
				return false
			}
			if fn.Body != nil {
				sites = append(sites, goast.FuncDeclSite(fn))
			}
		case *ast.FuncLit:
			sites = append(sites, goast.FuncLitSite(fn))
		}
		return true
	})
	return sites
}

func headerGroups(all []*ast.CommentGroup, body *ast.BlockStmt) []*ast.CommentGroup {
	limit := body.Rbrace
	if len(body.List) > 0 {
		limit = body.List[0].Pos()
	}
	var groups []*ast.CommentGroup
	for _, group := range all {
		if group.Pos() > body.Lbrace && group.End() <= limit &&
			!goast.IsDirectiveGroup(group) && !goast.IsSimplificationGroup(group) {
			groups = append(groups, group)
		}
	}
	return groups
}

func Describe() pass.Descriptor {
	return pass.Descriptor{
		ID:     ID,
		Rule:   "effective-go",
		Params: []string{paramMinLines},
	}
}
