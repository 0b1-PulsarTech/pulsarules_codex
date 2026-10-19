package singleuse

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	provider declarationSource
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		provider: goProvider{},
		reporter: finding.NewReporter(
			"single-use-declaration",
			finding.SeverityWarning,
			finding.CatAST,
		),
	}
}

func (analyzer *Analyzer) ID() string { return "single-use-declaration" }

func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageAST }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if ctx.ASTCache == nil || ctx.Sources == nil {
		return nil
	}
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	var findings []finding.Finding
	for _, decl := range analyzer.provider.Declarations(ctx) {
		if !eligible(decl) {
			continue
		}
		findings = append(findings, checkCandidate(ctx, analyzer.provider, decl, reporter)...)
	}
	return findings
}
