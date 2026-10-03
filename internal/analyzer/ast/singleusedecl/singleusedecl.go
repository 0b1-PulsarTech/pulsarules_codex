package singleusedecl

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct {
	provider core.DeclarationProvider
	reporter core.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		provider: goProvider{},
		reporter: core.NewReporter("single-use-declaration", core.SeverityWarning, core.CatAST),
	}
}

func (analyzer *Analyzer) ID() string { return "single-use-declaration" }

func (analyzer *Analyzer) Stage() core.StageID { return core.StageAST }

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
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

	reporter := analyzer.reporter.Resolved(ctx)
	ex := exemptionsFromParams(params)
	var findings []core.Finding
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

func (kinds declKinds) allows(kind core.DeclKind) bool {
	switch kind {
	case core.DeclConst:
		return kinds.checkConst
	case core.DeclVar:
		return kinds.checkVar
	default:
		return false
	}
}
