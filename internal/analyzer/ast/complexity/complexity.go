package complexity

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct {
	maxComplexity int
	maxFuncLines  int
	maxParams     int
	warnReporter  core.Reporter
	infoReporter  core.Reporter
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
		warnReporter:  core.NewReporter("complexity", core.SeverityWarning, core.CatAST),
		infoReporter:  core.NewReporter("complexity", core.SeverityInfo, core.CatAST),
	}
}

func (analyzer *Analyzer) ID() string          { return "complexity" }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageAST }

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	th := analyzer.resolveThresholds(ctx)
	findings := make([]core.Finding, 0, len(ctx.ChangedFiles))
	for _, fc := range ctx.ChangedFiles {
		findings = append(findings, th.checkChanged(ctx, fc)...)
	}
	return findings
}

func (analyzer *Analyzer) resolveThresholds(ctx *core.AnalysisContext) thresholds {
	params := ctx.Params(analyzer.ID())
	return thresholds{
		maxComplexity: params.Int("max_complexity", analyzer.maxComplexity),
		maxFuncLines:  params.Int("max_func_lines", analyzer.maxFuncLines),
		maxParams:     params.Int("max_params", analyzer.maxParams),
		warnReporter:  analyzer.warnReporter.Resolved(ctx),
		infoReporter:  analyzer.infoReporter.Resolved(ctx),
	}
}
