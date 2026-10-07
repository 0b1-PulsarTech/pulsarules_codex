package controlflow

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/syntaxfiles"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
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
	return syntaxfiles.Each(ctx, func(fc core.FileChange, file *syntax.File) []core.Finding {
		return analyzer.checkBlocks(fc, file, goElses(ctx, fc, reporter), reporter)
	})
}

func goElses(
	ctx *core.AnalysisContext,
	fc core.FileChange,
	reporter core.Reporter,
) map[int][]core.Finding {
	if ctx.ASTCache == nil {
		return nil
	}
	tree := ctx.ASTCache.Get(fc.Path)
	if tree == nil {
		return nil
	}
	return checkElses(ctx.ASTCache.FileSet(), fc, tree, reporter)
}

func (analyzer *Analyzer) checkBlocks(
	fc core.FileChange,
	file *syntax.File,
	elses map[int][]core.Finding,
	reporter core.Reporter,
) []core.Finding {
	var findings []core.Finding
	for idx, block := range file.Blocks {
		findings = append(findings, elses[idx]...)
		if block.Depth <= analyzer.maxNesting {
			continue
		}
		findings = append(findings, reporter.At(
			fc.Path,
			block.Line,
			fmt.Sprintf("nesting depth %d exceeds max %d", block.Depth, analyzer.maxNesting),
			"extract inner logic into helper functions",
		))
	}
	return findings
}
