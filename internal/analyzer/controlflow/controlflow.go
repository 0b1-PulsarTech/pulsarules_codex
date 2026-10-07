package controlflow

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/syntaxfiles"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

type Analyzer struct {
	maxNesting int
	reporter   pass.Reporter
}

func NewAnalyzer() *Analyzer {
	const defaultMaxNesting = 4
	return &Analyzer{
		maxNesting: defaultMaxNesting,
		reporter:   pass.NewReporter("control-flow", pass.SeverityWarning, pass.CatAST),
	}
}

func (analyzer *Analyzer) ID() string          { return "control-flow" }
func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	return syntaxfiles.Each(ctx, func(fc pass.FileChange, file *syntax.File) []pass.Finding {
		return analyzer.checkBlocks(fc, file, goElses(ctx, fc, reporter), reporter)
	})
}

func goElses(
	ctx *pass.Context,
	fc pass.FileChange,
	reporter pass.Reporter,
) map[int][]pass.Finding {
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
	fc pass.FileChange,
	file *syntax.File,
	elses map[int][]pass.Finding,
	reporter pass.Reporter,
) []pass.Finding {
	var findings []pass.Finding
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
