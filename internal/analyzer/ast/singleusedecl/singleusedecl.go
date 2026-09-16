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

func (a *Analyzer) ID() string { return "single-use-declaration" }

func (a *Analyzer) Stage() core.StageID { return core.StageAST }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if ctx.ASTCache == nil || ctx.Sources == nil {
		return nil
	}
	const (
		paramCheckConst = "check_const"
		paramCheckVar   = "check_var"
	)
	params := ctx.Params(a.ID())
	kinds := declKinds{
		checkConst: params.Bool(paramCheckConst, true),
		checkVar:   params.Bool(paramCheckVar, true),
	}
	if !kinds.any() {
		return nil
	}

	reporter := a.reporter.Resolved(ctx)
	ex := exemptionsFromParams(params)
	var findings []core.Finding
	for _, decl := range a.provider.Declarations(ctx) {
		if !ex.eligible(decl, kinds) {
			continue
		}
		findings = append(findings, checkCandidate(ctx, a.provider, decl, reporter)...)
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
