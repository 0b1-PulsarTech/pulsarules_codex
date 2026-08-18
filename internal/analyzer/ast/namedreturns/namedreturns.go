package namedreturns

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// Analyzer reports the one case where a missing result name is objectively a
// defect rather than a judgement call: two or more unnamed results of the same
// type, where nothing but the order tells a caller which is which.
type Analyzer struct{}

var namedReturnsReporter = core.NewReporter(
	"named-returns",
	core.SeverityWarning,
	core.CatAST,
)

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) ID() string   { return "named-returns" }
func (a *Analyzer) Name() string { return "Named results" }
func (a *Analyzer) Description() string {
	return "Reports two or more unnamed results of the same type, where only the order tells them apart"
}
func (a *Analyzer) Stage() core.StageID     { return core.StageAST }
func (a *Analyzer) Category() core.Category { return core.CatAST }
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{NeedsAST: true}
}

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	return core.RunPerGoFile(ctx, func(fc core.FileChange, f *ast.File) []core.Finding {
		return checkFile(ctx.ASTCache.FileSet(), fc, f)
	})
}

func checkFile(fset *token.FileSet, fc core.FileChange, f *ast.File) []core.Finding {
	var findings []core.Finding

	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			findings = append(
				findings,
				checkResults(fset, fc, node.Name.Name, node.Pos(), node.Type.Results)...,
			)
		case *ast.InterfaceType:
			for _, field := range node.Methods.List {
				funcType, ok := field.Type.(*ast.FuncType)
				if !ok || len(field.Names) == 0 {
					continue // embedded interface: no method name of its own to report
				}
				findings = append(
					findings,
					checkResults(fset, fc, field.Names[0].Name, field.Pos(), funcType.Results)...,
				)
			}
		}
		return true
	})

	return findings
}

// why: shared by both the FuncDecl and InterfaceType branches above.
func checkResults(
	fset *token.FileSet, fc core.FileChange, name string, pos token.Pos, results *ast.FieldList,
) []core.Finding {
	dup, found := duplicateResultType(results)
	if !found {
		return nil
	}
	return []core.Finding{namedReturnsReporter.At(
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

// duplicateResultType reports a type appearing twice among UNNAMED results.
// Naming even one result names them all in Go, so a signature that already
// carries a name is left alone.
func duplicateResultType(results *ast.FieldList) (string, bool) {
	if results == nil || len(results.List) < 2 {
		return "", false
	}

	seen := make(map[string]bool, len(results.List))
	for _, field := range results.List {
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
