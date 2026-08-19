package controlflow

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// Analyzer reports redundant else branches (return/panic/loop)
// and deep nesting in Go functions.
type Analyzer struct {
	maxNesting int
}

var controlFlowReporter = core.NewReporter("control-flow", core.SeverityWarning, core.CatAST)

// defaultMaxNesting is the deepest if/for nesting a function may reach
// before this analyzer reports it.
const defaultMaxNesting = 4

func NewAnalyzer() *Analyzer {
	return &Analyzer{maxNesting: defaultMaxNesting}
}

func (a *Analyzer) ID() string          { return "control-flow" }
func (a *Analyzer) Stage() core.StageID { return core.StageAST }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporter := controlFlowReporter.Resolved(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, f *ast.File) []core.Finding {
		return a.checkFile(ctx.ASTCache.FileSet(), fc, f, reporter)
	})
}

func (a *Analyzer) checkFile(
	fset *token.FileSet,
	fc core.FileChange,
	f *ast.File,
	reporter core.Reporter,
) []core.Finding {
	var findings []core.Finding

	emit := func(pos token.Pos, msg, suggestion string) {
		findings = append(
			findings,
			reporter.At(fc.Path, fset.Position(pos).Line, msg, suggestion),
		)
	}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		a.walkBlock(fn.Body, 0, emit)
	}

	return findings
}

func (a *Analyzer) walkBlock(
	block *ast.BlockStmt,
	depth int,
	emit func(token.Pos, string, string),
) {
	if block == nil {
		return
	}

	if depth > a.maxNesting {
		emit(
			block.Lbrace,
			fmt.Sprintf("nesting depth %d exceeds max %d", depth, a.maxNesting),
			"extract inner logic into helper functions",
		)
	}

	for _, stmt := range block.List {
		switch s := stmt.(type) {
		case *ast.IfStmt:
			a.walkIf(s, depth, emit)
		case *ast.ForStmt:
			a.walkBlock(s.Body, depth+1, emit)
		case *ast.RangeStmt:
			a.walkBlock(s.Body, depth+1, emit)
		case *ast.SwitchStmt:
			a.walkBlock(s.Body, depth+1, emit)
		case *ast.TypeSwitchStmt:
			a.walkBlock(s.Body, depth+1, emit)
		case *ast.SelectStmt:
			a.walkBlock(s.Body, depth+1, emit)
		case *ast.BlockStmt:
			a.walkBlock(s, depth+1, emit)
		}
	}
}

func (a *Analyzer) walkIf(
	s *ast.IfStmt,
	depth int,
	emit func(token.Pos, string, string),
) {
	if s.Body != nil {
		a.walkBlock(s.Body, depth+1, emit)
	}

	el, ok := s.Else.(*ast.BlockStmt)
	if !ok {
		// else if: walk the inner if
		if elseif, ok := s.Else.(*ast.IfStmt); ok {
			a.walkIf(elseif, depth, emit)
		}
		return
	}

	// matched a plain else { ... }
	a.checkElseBlock(el, emit)

	// still walk nested constructs inside the else block
	a.walkBlock(el, depth+1, emit)
}

func (a *Analyzer) checkElseBlock(
	block *ast.BlockStmt,
	emit func(token.Pos, string, string),
) {
	for _, stmt := range block.List {
		switch s := stmt.(type) {
		case *ast.ReturnStmt:
			emit(
				s.Return,
				"else block contains return; use early return instead",
				"remove else and return directly",
			)
			return
		case *ast.ExprStmt:
			if call, ok := s.X.(*ast.CallExpr); ok {
				if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
					emit(
						s.Pos(),
						"else block contains panic; panic directly instead",
						"remove else and panic directly",
					)
					return
				}
			}
		case *ast.ForStmt, *ast.RangeStmt:
			emit(
				s.Pos(),
				"else block contains loop; restructure to avoid else",
				"move loop out of else or invert the condition",
			)
			return
		case *ast.BranchStmt:
			emit(
				s.Pos(),
				fmt.Sprintf("else block contains %s statement", s.Tok),
				"restructure to avoid branch inside else",
			)
			return
		}
	}
}
