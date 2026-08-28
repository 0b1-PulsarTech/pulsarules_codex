package loopexit

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
		reporter: pass.NewReporter("context-discipline", pass.SeverityWarning, pass.CatAST),
	}
}

func (a *Analyzer) ID() string          { return "context-discipline" }
func (a *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	reporter := pass.ResolveReporter(a.reporter, ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, f *ast.File) []pass.Finding {
		return checkFile(ctx.ASTCache.FileSet(), fc, f, reporter)
	})
}

type fileCheck struct {
	fset     *token.FileSet
	fc       pass.FileChange
	reporter pass.Reporter
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, file *ast.File, reporter pass.Reporter,
) []pass.Finding {
	check := fileCheck{
		fset:     fset,
		fc:       fc,
		reporter: reporter,
	}
	var findings []pass.Finding
	ast.Inspect(file, func(node ast.Node) bool {
		body := funcBody(node)
		if body == nil {
			return true
		}
		findings = append(findings, checkFuncBody(check, body)...)
		return false
	})
	return findings
}

func funcBody(target ast.Node) *ast.BlockStmt {
	switch node := target.(type) {
	case *ast.FuncDecl:
		return node.Body
	case *ast.FuncLit:
		return node.Body
	default:
		return nil
	}
}

func checkFuncBody(check fileCheck, body *ast.BlockStmt) []pass.Finding {
	if body == nil {
		return nil
	}
	var findings []pass.Finding
	ast.Inspect(body, func(node ast.Node) bool {
		if funcLit, ok := node.(*ast.FuncLit); ok {
			findings = append(findings, checkFuncBody(check, funcLit.Body)...)
			return false
		}
		if goStmt, ok := node.(*ast.GoStmt); ok && isFireAndForgetDone(goStmt) {
			findings = append(findings, check.reporter.At(
				check.fc.Path,
				check.fset.Position(goStmt.Pos()).Line,
				"goroutine whose body starts by waiting on <-ctx.Done() is the "+
					"fire-and-forget shape concurrency.md #1 forbids; use "+
					"context.AfterFunc(ctx, cleanup), which returns a stop function",
				"replace the goroutine with context.AfterFunc",
			))
		}
		return true
	})
	return findings
}

func isFireAndForgetDone(goStmt *ast.GoStmt) bool {
	funcLit, ok := goStmt.Call.Fun.(*ast.FuncLit)
	if !ok || funcLit.Body == nil || len(funcLit.Body.List) == 0 {
		return false
	}
	return isDoneReceive(funcLit.Body.List[0])
}

func isDoneReceive(node ast.Node) bool {
	exprStmt, ok := node.(*ast.ExprStmt)
	if !ok {
		return false
	}
	unary, ok := exprStmt.X.(*ast.UnaryExpr)
	if !ok || unary.Op != token.ARROW {
		return false
	}
	call, ok := unary.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel != nil && sel.Sel.Name == "Done"
}
