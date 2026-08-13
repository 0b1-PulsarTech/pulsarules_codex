package namedreturns

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: finding.NewReporter("named-returns", finding.SeverityWarning, finding.CatAST),
	}
}

func (a *Analyzer) ID() string             { return "named-returns" }
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

	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			site := declSite{name: node.Name.Name, pos: node.Pos(), results: node.Type.Results}
			findings = append(findings, checkResults(fset, fc, site, reporter)...)
		case *ast.InterfaceType:
			for _, field := range node.Methods.List {
				funcType, ok := field.Type.(*ast.FuncType)
				if !ok || len(field.Names) == 0 {
					continue
				}
				site := declSite{
					name:    field.Names[0].Name,
					pos:     field.Pos(),
					results: funcType.Results,
				}
				findings = append(findings, checkResults(fset, fc, site, reporter)...)
			}
		}
		return true
	})

	return findings
}

type declSite struct {
	name    string
	pos     token.Pos
	results *ast.FieldList
}

func checkResults(
	fset *token.FileSet, fc pass.FileChange, site declSite, reporter finding.Reporter,
) []finding.Finding {
	dup, found := duplicateResultType(site.results)
	if !found {
		return nil
	}
	return []finding.Finding{reporter.At(
		fc.Path,
		fset.Position(site.pos).Line,
		fmt.Sprintf(
			"%s returns more than one unnamed %s, so only the order tells them apart",
			site.name,
			dup,
		),
		"name the results so the signature says which is which",
	)}
}

func duplicateResultType(results *ast.FieldList) (string, bool) {
	if results == nil || len(results.List) < 2 {
		return "", false
	}

	seen := make(map[string]bool, len(results.List))
	for _, field := range results.List {
		// Naming even one result names them all, so a named signature is left alone.
		if len(field.Names) > 0 {
			return "", false
		}
		// types.ExprString reads the type off the syntax alone, so no package
		// has to be loaded to compare two result types.
		name := types.ExprString(field.Type)
		if seen[name] {
			return name, true
		}
		seen[name] = true
	}
	return "", false
}
