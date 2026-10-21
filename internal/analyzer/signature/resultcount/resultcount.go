package resultcount

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const (
	ID              = "too-many-results"
	paramMaxResults = "max_results"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	const defaultMaxResults = 3
	limit := ctx.Params(ID).Int(paramMaxResults, defaultMaxResults)
	reporter := pass.ResolveReporter(
		finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST),
		ctx,
	)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []finding.Finding {
		if fc.Generated {
			return nil
		}
		return rule{reporter: reporter, limit: limit}.check(ctx.ASTCache.FileSet(), fc, file)
	})
}

type rule struct {
	reporter finding.Reporter
	limit    int
}

func (rule rule) check(fset *token.FileSet, fc pass.FileChange, file *ast.File) []finding.Finding {
	var findings []finding.Finding
	ast.Inspect(file, func(node ast.Node) bool {
		for _, site := range sitesOf(node) {
			count := goast.FieldCount(site.Results)
			if count <= rule.limit {
				continue
			}
			findings = append(findings, rule.reporter.At(
				fc.Path,
				fset.Position(site.Pos).Line,
				fmt.Sprintf(
					"%s returns %d results, more than the limit of %d",
					site.Name,
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

func sitesOf(decl ast.Node) []goast.FuncSite {
	switch node := decl.(type) {
	case *ast.FuncDecl:
		return []goast.FuncSite{goast.FuncDeclSite(node)}
	case *ast.FuncLit:
		return []goast.FuncSite{goast.FuncLitSite(node)}
	case *ast.TypeSpec:
		funcType, ok := node.Type.(*ast.FuncType)
		if !ok {
			return nil
		}
		return []goast.FuncSite{
			{Name: node.Name.Name, Pos: node.Pos(), Results: funcType.Results},
		}
	case *ast.InterfaceType:
		return goast.InterfaceMethodSites(node)
	}
	return nil
}

func Describe() pass.Descriptor {
	return pass.Descriptor{
		ID:     ID,
		Rule:   "effective-go",
		Params: []string{paramMaxResults},
	}
}
