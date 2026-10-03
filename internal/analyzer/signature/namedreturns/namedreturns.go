package namedreturns

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	reporter pass.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: pass.NewReporter("named-returns", pass.SeverityWarning, pass.CatAST),
	}
}

func (analyzer *Analyzer) ID() string          { return "named-returns" }
func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []pass.Finding {
		return checkFile(ctx.ASTCache.FileSet(), fc, file, reporter)
	})
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, file *ast.File, reporter pass.Reporter,
) []pass.Finding {
	var findings []pass.Finding

	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.FuncDecl:
			site := declSite{name: typed.Name.Name, pos: typed.Pos(), results: typed.Type.Results}
			findings = append(findings, checkResults(fset, fc, site, reporter)...)
		case *ast.InterfaceType:
			for _, field := range typed.Methods.List {
				funcType, ok := field.Type.(*ast.FuncType)
				if !ok || len(field.Names) == 0 {
					// Embedded interface: no method name of its own to report.
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
	fset *token.FileSet, fc pass.FileChange, site declSite, reporter pass.Reporter,
) []pass.Finding {
	dup, found := duplicateResultType(site.results)
	if !found {
		return nil
	}
	return []pass.Finding{reporter.At(
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
