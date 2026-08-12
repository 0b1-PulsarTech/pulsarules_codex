package timediscipline

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// Analyzer reports two ways a codebase fakes deterministic time instead of
// injecting it: a direct time.Sleep call (production code should pace on a
// ticker/context, tests should drive testing/synctest), and a struct field
// shaped like a clock-injection seam (a "now func() time.Time" field added
// only to make time testable).
type Analyzer struct{}

var timeDisciplineReporter = core.NewReporter("time-discipline", core.SeverityWarning, core.CatAST)

// NewAnalyzer creates a time-discipline analyzer.
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) ID() string   { return "time-discipline" }
func (a *Analyzer) Name() string { return "Time discipline" }
func (a *Analyzer) Description() string {
	return "Reports time.Sleep calls and now func() time.Time clock-injection fields"
}
func (a *Analyzer) Stage() core.StageID     { return core.StageAST }
func (a *Analyzer) Category() core.Category { return core.CatAST }
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{NeedsAST: true}
}

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.ASTCache == nil {
		return nil
	}

	fset := ctx.ASTCache.FileSet()
	var findings []core.Finding
	for fc, f := range ctx.ChangedGoASTs() {
		findings = append(findings, checkFile(fset, fc, f)...)
	}
	return findings
}

func checkFile(fset *token.FileSet, fc core.FileChange, f *ast.File) []core.Finding {
	var findings []core.Finding

	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			if isTimeSleepCall(node) {
				findings = append(findings, timeDisciplineReporter.At(
					fc.Path,
					fset.Position(node.Pos()).Line,
					"time.Sleep call; pace production code on a ticker/context.Done "+
						"and drive test timing with testing/synctest, never a real sleep",
					"replace with a ticker/context wait (production) or synctest.Test (tests)",
				))
			}
		case *ast.StructType:
			// Interface method signatures are also *ast.Field nodes, so the
			// seam check only walks a struct's own field list - an
			// interface's Now() time.Time method is the port the seam should
			// be replaced with, not the seam itself.
			for _, field := range node.Fields.List {
				name, ok := clockSeamFieldName(field)
				if !ok {
					continue
				}
				findings = append(findings, timeDisciplineReporter.At(
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

// isTimeSleepCall reports whether call is time.Sleep(...), matched
// syntactically on the selector (time.Sleep) rather than through go/types -
// the same identifier-name heuristic markStdlibBitSizeArg in the complexity
// analyzer already uses for strconv.
func isTimeSleepCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Sleep" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "time"
}

// clockSeamFieldName reports whether field is shaped exactly like the
// clock-injection seam the testing rule forbids: a named field of type
// func() time.Time, with no parameters. An anonymous/embedded field (no
// Names) is not a seam candidate - there is nothing to call it by.
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

// isTimeTimeExpr reports whether expr is the qualified identifier time.Time.
func isTimeTimeExpr(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Time" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "time"
}
