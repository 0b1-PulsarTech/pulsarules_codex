package complexity

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const ID = "complexity"

const (
	paramMaxComplexity = "max_complexity"
	paramMaxFuncLines  = "max_func_lines"
	paramMaxParams     = "max_params"
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
		warnReporter:  finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST),
		infoReporter:  finding.NewReporter(ID, finding.SeverityInfo, finding.CatAST),
	}
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	th := analyzer.resolveThresholds(ctx)
	findings := make([]finding.Finding, 0, len(ctx.ChangedFiles))
	for _, fc := range ctx.ChangedFiles {
		findings = append(findings, th.checkChanged(ctx, fc)...)
	}
	return findings
}

func (analyzer *Analyzer) resolveThresholds(ctx *pass.Context) thresholds {
	params := ctx.Params(ID)
	return thresholds{
		maxComplexity: params.Int(paramMaxComplexity, analyzer.maxComplexity),
		maxFuncLines:  params.Int(paramMaxFuncLines, analyzer.maxFuncLines),
		maxParams:     params.Int(paramMaxParams, analyzer.maxParams),
		warnReporter:  pass.ResolveReporter(analyzer.warnReporter, ctx),
		infoReporter:  pass.ResolveReporter(analyzer.infoReporter, ctx),
	}
}

func Describe() pass.Descriptor {
	return pass.Descriptor{
		ID:     ID,
		Rule:   "code-smells",
		Params: []string{paramMaxComplexity, paramMaxFuncLines, paramMaxParams},
	}
}
