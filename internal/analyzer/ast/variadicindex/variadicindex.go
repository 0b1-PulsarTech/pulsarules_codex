package variadicindex

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) ID() string          { return "variadic-index" }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageAST }

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporter := core.NewReporter(analyzer.ID(), core.SeverityWarning, core.CatAST).Resolved(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, file *ast.File) []core.Finding {
		if fc.Generated {
			return nil
		}
		return checkFile(ctx.ASTCache.FileSet(), fc, file, reporter)
	})
}

func checkFile(
	fset *token.FileSet, fc core.FileChange, file *ast.File, reporter core.Reporter,
) []core.Finding {
	var findings []core.Finding
	report := func(index *ast.IndexExpr, name string) {
		findings = append(findings, reporter.At(
			fc.Path,
			fset.Position(index.Pos()).Line,
			fmt.Sprintf("variadic parameter %s is read by position", name),
			"take a named options struct, or range over the values instead of indexing them",
		))
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.FuncDecl:
			scanFunc(typed.Type, typed.Body, report)
		case *ast.FuncLit:
			scanFunc(typed.Type, typed.Body, report)
		}
		return true
	})
	return findings
}

func scanFunc(fn *ast.FuncType, body *ast.BlockStmt, report func(*ast.IndexExpr, string)) {
	name := variadicName(fn)
	if name == "" || body == nil {
		return
	}
	// The top-level block is the parameter scope: redeclaring there is illegal Go,
	// so only nested blocks can shadow the variadic name.
	walker := scope{name: name, report: report}
	walker.topStmts(body.List)
}

func variadicName(fn *ast.FuncType) string {
	params := fn.Params.List
	if len(params) == 0 {
		return ""
	}
	last := params[len(params)-1]
	if _, ok := last.Type.(*ast.Ellipsis); !ok || len(last.Names) == 0 {
		return ""
	}
	return last.Names[len(last.Names)-1].Name
}
