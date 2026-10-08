package singleuse

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	provider DeclarationProvider
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
	const (
		paramCheckConst = "check_const"
		paramCheckVar   = "check_var"
	)
	params := ctx.Params(analyzer.ID())
	kinds := declKinds{
		checkConst: params.Bool(paramCheckConst, true),
		checkVar:   params.Bool(paramCheckVar, true),
	}
	if !kinds.any() {
		return nil
	}

	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	ex := exemptionsFromParams(params)
	var findings []finding.Finding
	for _, decl := range analyzer.provider.Declarations(ctx) {
		if !ex.eligible(decl, kinds) {
			continue
		}
		findings = append(findings, checkCandidate(ctx, analyzer.provider, decl, reporter)...)
	}
	return findings
}

type declKinds struct {
	checkConst bool
	checkVar   bool
}

func (kinds declKinds) any() bool { return kinds.checkConst || kinds.checkVar }

func (kinds declKinds) allows(kind DeclKind) bool {
	switch kind {
	case DeclConst:
		return kinds.checkConst
	case DeclVar:
		return kinds.checkVar
	default:
		return false
	}
}
