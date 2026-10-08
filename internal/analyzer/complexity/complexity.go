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

func (analyzer *Analyzer) ID() string             { return "complexity" }
func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageAST }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	th := analyzer.resolveThresholds(ctx)
	findings := make([]finding.Finding, 0, len(ctx.ChangedFiles))
	for _, fc := range ctx.ChangedFiles {
		findings = append(findings, th.checkChanged(ctx, fc)...)
	}
	return findings
}

func (analyzer *Analyzer) resolveThresholds(ctx *pass.Context) thresholds {
	params := ctx.Params(analyzer.ID())
	return thresholds{
		maxComplexity: params.Int("max_complexity", analyzer.maxComplexity),
		maxFuncLines:  params.Int("max_func_lines", analyzer.maxFuncLines),
		maxParams:     params.Int("max_params", analyzer.maxParams),
		warnReporter:  pass.ResolveReporter(analyzer.warnReporter, ctx),
		infoReporter:  pass.ResolveReporter(analyzer.infoReporter, ctx),
	}
}
