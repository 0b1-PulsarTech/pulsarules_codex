package singleuse

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

type Analyzer struct {
	provider DeclarationProvider
	reporter pass.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		provider: goProvider{},
		reporter: pass.NewReporter("single-use-declaration", pass.SeverityWarning, pass.CatAST),
	}
}

func (a *Analyzer) ID() string { return "single-use-declaration" }

func (a *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
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

	reporter := pass.ResolveReporter(a.reporter, ctx)
	ex := exemptionsFromParams(params)
	var findings []pass.Finding
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
