package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type StageRunner struct {
	stages map[core.StageID][]core.Analyzer
	cfg    *core.AnalysisConfig
}

func NewStageRunner(cfg *core.AnalysisConfig) *StageRunner {
	return &StageRunner{
		stages: make(map[core.StageID][]core.Analyzer),
		cfg:    cfg,
	}
}

func (runner *StageRunner) Register(analyzer core.Analyzer) {
	runner.stages[analyzer.Stage()] = append(runner.stages[analyzer.Stage()], analyzer)
}

func (runner *StageRunner) RunStages(ctx *core.AnalysisContext) []core.Finding {
	for stage := core.StageContext; stage <= core.StageOutput; stage++ {
		analyzers := runner.stages[stage]
		for _, analyzer := range analyzers {
			if !runner.enabled(analyzer) {
				continue
			}
			findings := analyzer.Analyze(ctx)
			// An in-place analyzer already appended to ctx.Findings; skip its return.
			if _, inPlace := analyzer.(core.InPlaceAnalyzer); inPlace {
				continue
			}
			ctx.Findings = append(ctx.Findings, findings...)
		}
	}
	return ctx.Findings
}

func (runner *StageRunner) enabled(analyzer core.Analyzer) bool {
	if runner.cfg == nil {
		return true
	}
	cfg, ok := runner.cfg.Analyzers[analyzer.ID()]
	if !ok {
		return true
	}
	return cfg.Enabled
}
