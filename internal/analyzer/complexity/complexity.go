package complexity

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	maxComplexity int
	maxFuncLines  int
	maxParams     int
	warnReporter  finding.Reporter
	infoReporter  finding.Reporter
}

const (
	defaultMaxComplexity = 15
	defaultMaxFuncLines  = 80
	defaultMaxParams     = 5
)

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		maxComplexity: defaultMaxComplexity,
		maxFuncLines:  defaultMaxFuncLines,
		maxParams:     defaultMaxParams,
		warnReporter:  finding.NewReporter("complexity", finding.SeverityWarning, finding.CatAST),
		infoReporter:  finding.NewReporter("complexity", finding.SeverityInfo, finding.CatAST),
	}
}

func (a *Analyzer) ID() string             { return "complexity" }
func (a *Analyzer) Stage() finding.StageID { return finding.StageAST }
func (a *Analyzer) Needs() finding.Requirements {
	return finding.Requirements{NeedsAST: true}
}

func (a *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if ctx.ASTCache == nil {
		return nil
	}
	th := a.resolveThresholds(ctx)
	fset := ctx.ASTCache.FileSet()
	var findings []finding.Finding
	for fc, f := range ctx.ChangedGoASTs() {
		findings = append(findings, th.checkFile(fset, fc, f)...)
	}
	return findings
}

func (a *Analyzer) resolveThresholds(ctx *pass.Context) thresholds {
	params := ctx.Params(a.ID())
	return thresholds{
		maxComplexity: params.Int("max_complexity", a.maxComplexity),
		maxFuncLines:  params.Int("max_func_lines", a.maxFuncLines),
		maxParams:     params.Int("max_params", a.maxParams),
		warnReporter:  a.warnReporter,
		infoReporter:  a.infoReporter,
	}
}
