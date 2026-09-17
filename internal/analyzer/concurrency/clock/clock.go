package clock

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	reporter pass.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: pass.NewReporter("time-discipline", pass.SeverityWarning, pass.CatAST),
	}
}

func (a *Analyzer) ID() string          { return "time-discipline" }
func (a *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	reporter := pass.ResolveReporter(a.reporter, ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, f *ast.File) []pass.Finding {
		return checkFile(ctx.ASTCache.FileSet(), fc, f, reporter)
	})
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, f *ast.File, reporter pass.Reporter,
) []pass.Finding {
	var findings []pass.Finding

	bubbles := synctestBubbles(f)
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			// A sleep in a synctest bubble advances the fake clock, so flagging it
			// would ask the author to replace correct code.
			if isTimeSleepCall(node) && !insideBubble(bubbles, node.Pos()) {
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

func synctestBubbles(file *ast.File) [][2]token.Pos {
	var bubbles [][2]token.Pos
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || !isSynctestRunner(call) {
			return true
		}
		for _, arg := range call.Args {
			if lit, isLit := arg.(*ast.FuncLit); isLit {
				bubbles = append(bubbles, [2]token.Pos{lit.Pos(), lit.End()})
			}
		}
		return true
	})
	return bubbles
}

func isSynctestRunner(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return false
	}
	if sel.Sel.Name != "Test" && sel.Sel.Name != "Run" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "synctest"
}

func insideBubble(bubbles [][2]token.Pos, pos token.Pos) bool {
	for _, bubble := range bubbles {
		if pos >= bubble[0] && pos < bubble[1] {
			return true
		}
	}
	return false
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
