package toomanyresults

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) ID() string          { return "too-many-results" }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageAST }

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	const defaultMaxResults = 3
	limit := ctx.Params(analyzer.ID()).Int("max_results", defaultMaxResults)
	reporter := core.NewReporter(analyzer.ID(), core.SeverityWarning, core.CatAST).Resolved(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, file *ast.File) []core.Finding {
		if fc.Generated {
			return nil
		}
		return rule{reporter: reporter, limit: limit}.check(ctx.ASTCache.FileSet(), fc, file)
	})
}

type declSite struct {
	name    string
	pos     token.Pos
	results *ast.FieldList
}

type rule struct {
	reporter core.Reporter
	limit    int
}

func (rule rule) check(fset *token.FileSet, fc core.FileChange, file *ast.File) []core.Finding {
	var findings []core.Finding
	ast.Inspect(file, func(node ast.Node) bool {
		for _, site := range sitesOf(node) {
			count := countResults(site.results)
			if count <= rule.limit {
				continue
			}
			findings = append(findings, rule.reporter.At(
				fc.Path,
				fset.Position(site.pos).Line,
				fmt.Sprintf(
					"%s returns %d results, more than the limit of %d",
					site.name,
					count,
					rule.limit,
				),
				"return a named result of an anonymous struct, or a named struct type "+
					"when it is reused",
			))
		}
		return true
	})
	return findings
}

func sitesOf(decl ast.Node) []declSite {
	switch node := decl.(type) {
	case *ast.FuncDecl:
		return []declSite{{name: node.Name.Name, pos: node.Pos(), results: node.Type.Results}}
	case *ast.FuncLit:
		return []declSite{{name: "function literal", pos: node.Pos(), results: node.Type.Results}}
	case *ast.TypeSpec:
		funcType, ok := node.Type.(*ast.FuncType)
		if !ok {
			return nil
		}
		return []declSite{{name: node.Name.Name, pos: node.Pos(), results: funcType.Results}}
	case *ast.InterfaceType:
		var sites []declSite
		for _, field := range node.Methods.List {
			funcType, ok := field.Type.(*ast.FuncType)
			if !ok || len(field.Names) == 0 {
				continue
			}
			sites = append(sites, declSite{
				name: field.Names[0].Name, pos: field.Pos(), results: funcType.Results,
			})
		}
		return sites
	}
	return nil
}

func countResults(results *ast.FieldList) int {
	if results == nil {
		return 0
	}
	total := 0
	for _, field := range results.List {
		total += max(1, len(field.Names))
	}
	return total
}
