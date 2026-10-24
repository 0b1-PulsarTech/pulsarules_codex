package controlflow

import (
	"fmt"
	"go/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

const (
	ID              = "control-flow"
	paramForbidElse = "forbid_else"
)

type Analyzer struct {
	maxNesting   int
	reporter     finding.Reporter
	elseReporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	const defaultMaxNesting = 4
	return &Analyzer{
		maxNesting:   defaultMaxNesting,
		reporter:     finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST),
		elseReporter: finding.NewReporter(ID+"/else", finding.SeverityWarning, finding.CatAST),
	}
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	forbid := ctx.Params(ID).Bool(paramForbidElse, false)
	return ctx.EachSyntaxFile(func(fc pass.FileChange, file *syntax.File) []finding.Finding {
		tree := goTree(ctx, fc)
		var byBlock map[int][]finding.Finding
		if tree != nil && !forbid {
			byBlock = checkElses(ctx.ASTCache.FileSet(), fc, tree, analyzer.elseReporter)
		}
		findings := analyzer.checkBlocks(fc, file, byBlock, analyzer.reporter)
		if tree != nil && forbid {
			findings = append(
				findings,
				forbidGoElses(ctx.ASTCache.FileSet(), fc, tree, analyzer.elseReporter)...,
			)
		}
		if forbid {
			return append(findings, forbidSyntaxElses(fc, file, analyzer.elseReporter)...)
		}
		return append(findings, exitSyntaxElses(fc, file, analyzer.elseReporter)...)
	})
}

func goTree(ctx *pass.Context, fc pass.FileChange) *ast.File {
	if ctx.ASTCache == nil {
		return nil
	}
	return ctx.ASTCache.Get(fc.Path)
}

func (analyzer *Analyzer) checkBlocks(
	fc pass.FileChange,
	file *syntax.File,
	elses map[int][]finding.Finding,
	reporter finding.Reporter,
) []finding.Finding {
	var findings []finding.Finding
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

func Describe() pass.Descriptor {
	return pass.Descriptor{ID: ID, Rule: "code-smells", Params: []string{paramForbidElse}}
}
