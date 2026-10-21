package clock

import (
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const ID = "time-discipline"

type Analyzer struct {
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST),
	}
}

func (analyzer *Analyzer) ID() string             { return ID }
func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageAST }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []finding.Finding {
		return checkFile(ctx.ASTCache.FileSet(), fc, file, reporter)
	})
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, file *ast.File, reporter finding.Reporter,
) []finding.Finding {
	var findings []finding.Finding

	bubbles := synctestBubbles(file)
	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CallExpr:
			// A sleep in a synctest bubble advances the fake clock, so flagging it
			// would ask the author to replace correct code.
			if goast.PkgCall(file, typed, "time", "Sleep") && !insideBubble(bubbles, typed.Pos()) {
				findings = append(findings, reporter.At(
					fc.Path,
					fset.Position(typed.Pos()).Line,
					"time.Sleep call; pace production code on a ticker/context.Done "+
						"and drive test timing with testing/synctest, never a real sleep",
					"replace with a ticker/context wait (production) or synctest.Test (tests)",
				))
			}
		case *ast.StructType:
			// Interface method signatures are also *ast.Field nodes; walking
			// only *ast.StructType excludes an interface's Now() method.
			for _, field := range typed.Fields.List {
				name, ok := clockSeamFieldName(file, field)
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
		if !ok || !goast.PkgCall(file, call, "testing/synctest", "Test", "Run") {
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

func insideBubble(bubbles [][2]token.Pos, pos token.Pos) bool {
	for _, bubble := range bubbles {
		if pos >= bubble[0] && pos < bubble[1] {
			return true
		}
	}
	return false
}

func clockSeamFieldName(file *ast.File, field *ast.Field) (string, bool) {
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
	if !goast.PkgRef(file, fn.Results.List[0].Type, "time", "Time") {
		return "", false
	}
	return field.Names[0].Name, true
}

func Describe() pass.Descriptor {
	return pass.Descriptor{ID: ID, Rule: "concurrency"}
}
