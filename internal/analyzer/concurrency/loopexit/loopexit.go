package loopexit

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const ID = "context-discipline"

type Analyzer struct {
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST),
	}
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	return ctx.EachGoFile(func(fc pass.FileChange, file *ast.File) []finding.Finding {
		return checkFile(ctx.ASTCache.FileSet(), fc, file, analyzer.reporter)
	})
}

type fileCheck struct {
	file             *ast.File
	fset             *token.FileSet
	fc               pass.FileChange
	comments         []*ast.CommentGroup
	codeLines        map[int]bool
	tickerFieldNames map[string]bool
	reporter         finding.Reporter
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, file *ast.File, reporter finding.Reporter,
) []finding.Finding {
	check := fileCheck{
		file:      file,
		fset:      fset,
		fc:        fc,
		comments:  file.Comments,
		codeLines: linesWithCode(fset, file),
		// A field's type is the same across every function that reads it, so names are file-wide.
		tickerFieldNames: collectTickerFieldNames(file),
		reporter:         reporter,
	}
	var findings []finding.Finding
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

func checkFuncBody(check fileCheck, params *ast.FieldList, body *ast.BlockStmt) []finding.Finding {
	if body == nil {
		return nil
	}
	scope := collectChannelScope(check.file, params, body, check.tickerFieldNames)
	labels := collectLoopLabels(body)

	var findings []finding.Finding
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

func Describe() pass.Descriptor {
	return pass.Descriptor{ID: ID, Rule: "concurrency"}
}
