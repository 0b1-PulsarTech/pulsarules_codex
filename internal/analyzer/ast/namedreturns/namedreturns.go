package namedreturns

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct {
	reporter core.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: core.NewReporter("named-returns", core.SeverityWarning, core.CatAST),
	}
}

func (a *Analyzer) ID() string          { return "named-returns" }
func (a *Analyzer) Stage() core.StageID { return core.StageAST }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	return core.RunPerGoFile(ctx, func(fc core.FileChange, f *ast.File) []core.Finding {
		return checkFile(ctx.ASTCache.FileSet(), fc, f, a.reporter)
	})
}

func checkFile(
	fset *token.FileSet, fc core.FileChange, f *ast.File, reporter core.Reporter,
) []core.Finding {
	var findings []core.Finding

	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			findings = append(
				findings,
				checkResults(fset, fc, node.Name.Name, node.Pos(), node.Type.Results, reporter)...,
			)
		case *ast.InterfaceType:
			for _, field := range node.Methods.List {
				funcType, ok := field.Type.(*ast.FuncType)
				if !ok || len(field.Names) == 0 {
					continue
				}
				findings = append(
					findings,
					checkResults(
						fset,
						fc,
						field.Names[0].Name,
						field.Pos(),
						funcType.Results,
						reporter,
					)...,
				)
			}
		}
		return true
	})

	return findings
}

func checkResults(
	fset *token.FileSet, fc core.FileChange, name string, pos token.Pos, results *ast.FieldList,
	reporter core.Reporter,
) []core.Finding {
	dup, found := duplicateResultType(results)
	if !found {
		return nil
	}
	return []core.Finding{reporter.At(
		fc.Path,
		fset.Position(pos).Line,
		fmt.Sprintf(
			"%s returns more than one unnamed %s, so only the order tells them apart",
			name,
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
