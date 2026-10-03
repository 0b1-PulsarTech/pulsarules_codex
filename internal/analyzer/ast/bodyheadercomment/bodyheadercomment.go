package bodyheadercomment

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

func (analyzer *Analyzer) ID() string { return "body-header-comment" }

func (analyzer *Analyzer) Stage() core.StageID { return core.StageAST }

func (analyzer *Analyzer) reporter() core.Reporter {
	return core.NewReporter(analyzer.ID(), core.SeverityWarning, core.CatAST)
}

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	const defaultMinLines = 2
	reporter := analyzer.reporter().Resolved(ctx)
	minLines := ctx.Params(analyzer.ID()).Int("min_lines", defaultMinLines)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, file *ast.File) []core.Finding {
		if fc.Generated {
			return nil
		}
		settings := check{fset: ctx.ASTCache.FileSet(), reporter: reporter, minLines: minLines}
		return settings.file(fc, file)
	})
}

type bodySite struct {
	name string
	body *ast.BlockStmt
}

type check struct {
	fset     *token.FileSet
	reporter core.Reporter
	minLines int
}

func (settings check) file(fc core.FileChange, file *ast.File) []core.Finding {
	var findings []core.Finding
	for _, site := range bodySites(fc, file) {
		groups := headerGroups(file.Comments, site.body)
		lines := 0
		for _, group := range groups {
			lines += directive.ProseLines(group)
		}
		if lines < settings.minLines {
			continue
		}
		findings = append(findings, settings.reporter.At(
			fc.Path,
			settings.fset.Position(groups[0].Pos()).Line,
			fmt.Sprintf("%s opens with a %d-line comment block", site.name, lines),
			"delete it; a removed doc comment is never moved into the body, "+
				"keep only a one-line why at the statement it explains",
		))
	}
	return findings
}

func bodySites(fc core.FileChange, file *ast.File) []bodySite {
	var sites []bodySite
	ast.Inspect(file, func(node ast.Node) bool {
		switch fn := node.(type) {
		case *ast.FuncDecl:
			if gotest.CoversFunc(fc, file, fn) {
				return false
			}
			if fn.Body != nil {
				sites = append(sites, bodySite{name: fn.Name.Name, body: fn.Body})
			}
		case *ast.FuncLit:
			sites = append(sites, bodySite{name: "function literal", body: fn.Body})
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
			!directive.IsDirectiveGroup(group) && !directive.IsSimplificationGroup(group) {
			groups = append(groups, group)
		}
	}
	return groups
}
