package contextdiscipline

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
		reporter: core.NewReporter("context-discipline", core.SeverityWarning, core.CatAST),
	}
}

func (analyzer *Analyzer) ID() string          { return "context-discipline" }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageAST }

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporter := analyzer.reporter.Resolved(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, file *ast.File) []core.Finding {
		return checkFile(ctx.ASTCache.FileSet(), fc, file, reporter)
	})
}

type fileCheck struct {
	fset             *token.FileSet
	fc               core.FileChange
	comments         []*ast.CommentGroup
	codeLines        map[int]bool
	tickerFieldNames map[string]bool
	reporter         core.Reporter
}

func checkFile(
	fset *token.FileSet, fc core.FileChange, file *ast.File, reporter core.Reporter,
) []core.Finding {
	check := fileCheck{
		fset:      fset,
		fc:        fc,
		comments:  file.Comments,
		codeLines: linesWithCode(fset, file),
		// A field's type is the same across every function that reads it, so names are file-wide.
		tickerFieldNames: collectTickerFieldNames(file),
		reporter:         reporter,
	}
	var findings []core.Finding
	ast.Inspect(file, func(node ast.Node) bool {
		params, body := funcSignature(node)
		if body == nil {
			return true
		}
		findings = append(findings, checkFuncBody(check, params, body)...)
		return false
	})
	return findings
}

func funcSignature(target ast.Node) (*ast.FieldList, *ast.BlockStmt) {
	switch node := target.(type) {
	case *ast.FuncDecl:
		return node.Type.Params, node.Body
	case *ast.FuncLit:
		return node.Type.Params, node.Body
	default:
		return nil, nil
	}
}

func checkFuncBody(check fileCheck, params *ast.FieldList, body *ast.BlockStmt) []core.Finding {
	if body == nil {
		return nil
	}
	scope := collectChannelScope(params, body, check.tickerFieldNames)
	labels := collectLoopLabels(body)

	var findings []core.Finding
	ast.Inspect(body, func(node ast.Node) bool {
		if funcLit, ok := node.(*ast.FuncLit); ok {
			findings = append(findings, checkFuncBody(check, funcLit.Type.Params, funcLit.Body)...)
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
		if shape, ok := tickerLoopShape(node, scope); ok {
			stmt, _ := node.(ast.Stmt)
			findings = append(findings, checkTickerLoop(check, shape, scope, labels[stmt])...)
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
	return isDoneReceiveNode(funcLit.Body.List[0], nil)
}
