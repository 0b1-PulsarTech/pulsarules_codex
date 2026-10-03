package loopvari

import (
	"go/ast"
	"go/token"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const loopVarName = "i"

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) ID() string          { return "loop-var-i" }
func (a *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	reporter := pass.ResolveReporter(
		pass.NewReporter(a.ID(), pass.SeverityWarning, pass.CatAST),
		ctx,
	)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, f *ast.File) []pass.Finding {
		if fc.Generated {
			return nil
		}
		return checkFile(ctx.ASTCache.FileSet(), fc, f, reporter)
	})
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, f *ast.File, reporter pass.Reporter,
) []pass.Finding {
	var findings []pass.Finding
	pkgLevel := packageShadows(f)
	for _, decl := range f.Decls {
		ast.Inspect(decl, func(n ast.Node) bool {
			if loopEscapes(n, pkgLevel, decl) {
				findings = append(findings, reporter.At(
					fc.Path,
					fset.Position(n.Pos()).Line,
					"loop variable i is used for more than indexing",
					"name the variable after what it counts or holds, such as row, attempt or "+
						"offset",
				))
			}
			return true
		})
	}
	return findings
}

func loopEscapes(n ast.Node, pkgLevel map[string]bool, decl ast.Decl) bool {
	switch loop := n.(type) {
	case *ast.RangeStmt:
		return definesLoopVar(loop.Key, loop.Tok) &&
			newScanner(declShadows(pkgLevel, decl, loop)).scanNode(loop.Body)
	case *ast.ForStmt:
		return forDefinesLoopVar(loop) && scanFor(loop, declShadows(pkgLevel, decl, loop))
	}
	return false
}

func definesLoopVar(key ast.Expr, tok token.Token) bool {
	ident, ok := key.(*ast.Ident)
	return ok && tok == token.DEFINE && ident.Name == loopVarName
}

func forDefinesLoopVar(loop *ast.ForStmt) bool {
	assign, ok := loop.Init.(*ast.AssignStmt)
	if !ok || assign.Tok != token.DEFINE {
		return false
	}
	return hasLoopVar(assign.Lhs)
}

func hasLoopVar(exprs []ast.Expr) bool {
	for _, expr := range exprs {
		if ident, ok := expr.(*ast.Ident); ok && ident.Name == loopVarName {
			return true
		}
	}
	return false
}

func scanFor(loop *ast.ForStmt, shadowed map[string]bool) bool {
	scan := newScanner(shadowed)
	scan.inCond = true
	if loop.Cond != nil && scan.scanNode(loop.Cond) {
		return true
	}
	scan.inCond = false
	if slices.ContainsFunc(uncountedPost(loop.Post), scan.scanNode) {
		return true
	}
	return scan.scanNode(loop.Body)
}

func uncountedPost(post ast.Stmt) []ast.Node {
	switch step := post.(type) {
	case nil:
		return nil
	case *ast.IncDecStmt:
		if isLoopIdent(step.X) {
			return nil
		}
	case *ast.AssignStmt:
		return uncountedAssign(step)
	}
	return []ast.Node{post}
}

func uncountedAssign(step *ast.AssignStmt) []ast.Node {
	if step.Tok == token.DEFINE || len(step.Lhs) != len(step.Rhs) {
		return []ast.Node{step}
	}
	var rest []ast.Node
	for pos, lhs := range step.Lhs {
		if isLoopIdent(lhs) && isCounterValue(step.Tok, step.Rhs[pos]) {
			continue
		}
		rest = append(rest, lhs, step.Rhs[pos])
	}
	return rest
}
