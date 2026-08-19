package timediscipline

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct {
	reporter core.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: core.NewReporter("time-discipline", core.SeverityWarning, core.CatAST),
	}
}

func (a *Analyzer) ID() string          { return "time-discipline" }
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
		case *ast.CallExpr:
			if isTimeSleepCall(node) {
				findings = append(findings, reporter.At(
					fc.Path,
					fset.Position(node.Pos()).Line,
					"time.Sleep call; pace production code on a ticker/context.Done "+
						"and drive test timing with testing/synctest, never a real sleep",
					"replace with a ticker/context wait (production) or synctest.Test (tests)",
				))
			}
		case *ast.StructType:
			// Interface method signatures are also *ast.Field nodes; walking
			// only *ast.StructType excludes an interface's Now() method.
			for _, field := range node.Fields.List {
				name, ok := clockSeamFieldName(field)
				if !ok {
					continue
				}
				findings = append(findings, reporter.At(
					fc.Path,
					fset.Position(field.Pos()).Line,
					"field "+name+" is a now func() time.Time clock-injection seam",
					"drive test timing with testing/synctest instead of seaming time "+
						"with a func() time.Time field; richer code takes an injected Clock",
				))
			}
		}
		return true
	})

	return findings
}

func isTimeSleepCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Sleep" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "time"
}

func clockSeamFieldName(field *ast.Field) (string, bool) {
	if len(field.Names) == 0 {
		return "", false
	}
	fn, ok := field.Type.(*ast.FuncType)
	if !ok || fn.Params == nil || len(fn.Params.List) != 0 {
		return "", false
	}
	if fn.Results == nil || len(fn.Results.List) != 1 || len(fn.Results.List[0].Names) != 0 {
		return "", false
	}
	if !isTimeTimeExpr(fn.Results.List[0].Type) {
		return "", false
	}
	return field.Names[0].Name, true
}

func isTimeTimeExpr(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Time" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "time"
}
