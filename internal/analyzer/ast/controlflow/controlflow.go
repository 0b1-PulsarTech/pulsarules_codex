package controlflow

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct {
	maxNesting int
	reporter   core.Reporter
}

func NewAnalyzer() *Analyzer {
	const defaultMaxNesting = 4
	return &Analyzer{
		maxNesting: defaultMaxNesting,
		reporter:   core.NewReporter("control-flow", core.SeverityWarning, core.CatAST),
	}
}

func (analyzer *Analyzer) ID() string          { return "control-flow" }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageAST }

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporter := analyzer.reporter.Resolved(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, file *ast.File) []core.Finding {
		return analyzer.checkFile(ctx.ASTCache.FileSet(), fc, file, reporter)
	})
}

func (analyzer *Analyzer) checkFile(
	fset *token.FileSet,
	fc core.FileChange,
	file *ast.File,
	reporter core.Reporter,
) []core.Finding {
	var findings []core.Finding

	emit := func(pos token.Pos, msg, suggestion string) {
		findings = append(
			findings,
			reporter.At(fc.Path, fset.Position(pos).Line, msg, suggestion),
		)
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		analyzer.walkBlock(fn.Body, 0, emit)
	}

	return findings
}

func (analyzer *Analyzer) walkBlock(
	block *ast.BlockStmt,
	depth int,
	emit func(token.Pos, string, string),
) {
	if block == nil {
		return
	}

	if depth > analyzer.maxNesting {
		emit(
			block.Lbrace,
			fmt.Sprintf("nesting depth %d exceeds max %d", depth, analyzer.maxNesting),
			"extract inner logic into helper functions",
		)
	}

	for _, stmt := range block.List {
		switch typed := stmt.(type) {
		case *ast.IfStmt:
			analyzer.walkIf(typed, depth, emit)
		case *ast.ForStmt:
			analyzer.walkBlock(typed.Body, depth+1, emit)
		case *ast.RangeStmt:
			analyzer.walkBlock(typed.Body, depth+1, emit)
		case *ast.SwitchStmt:
			analyzer.walkBlock(typed.Body, depth+1, emit)
		case *ast.TypeSwitchStmt:
			analyzer.walkBlock(typed.Body, depth+1, emit)
		case *ast.SelectStmt:
			analyzer.walkBlock(typed.Body, depth+1, emit)
		case *ast.BlockStmt:
			analyzer.walkBlock(typed, depth+1, emit)
		}
	}
}

func (analyzer *Analyzer) walkIf(
	ifStmt *ast.IfStmt,
	depth int,
	emit func(token.Pos, string, string),
) {
	if ifStmt.Body != nil {
		analyzer.walkBlock(ifStmt.Body, depth+1, emit)
	}

	el, ok := ifStmt.Else.(*ast.BlockStmt)
	if !ok {
		if elseif, ok := ifStmt.Else.(*ast.IfStmt); ok {
			analyzer.walkIf(elseif, depth, emit)
		}
		return
	}

	analyzer.checkElseBlock(el, emit)

	analyzer.walkBlock(el, depth+1, emit)
}

func (analyzer *Analyzer) checkElseBlock(
	block *ast.BlockStmt,
	emit func(token.Pos, string, string),
) {
	for _, stmt := range block.List {
		switch typed := stmt.(type) {
		case *ast.ReturnStmt:
			emit(
				typed.Return,
				"else block contains return; use early return instead",
				"remove else and return directly",
			)
			return
		case *ast.ExprStmt:
			if call, ok := typed.X.(*ast.CallExpr); ok {
				if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
					emit(
						typed.Pos(),
						"else block contains panic; panic directly instead",
						"remove else and panic directly",
					)
					return
				}
			}
		case *ast.ForStmt, *ast.RangeStmt:
			emit(
				typed.Pos(),
				"else block contains loop; restructure to avoid else",
				"move loop out of else or invert the condition",
			)
			return
		case *ast.BranchStmt:
			emit(
				typed.Pos(),
				fmt.Sprintf("else block contains %s statement", typed.Tok),
				"restructure to avoid branch inside else",
			)
			return
		}
	}
}
