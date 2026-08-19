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

func (r *StageRunner) Register(a core.Analyzer) {
	r.stages[a.Stage()] = append(r.stages[a.Stage()], a)
}

func (r *StageRunner) RunStages(ctx *core.AnalysisContext) []core.Finding {
	for stage := core.StageContext; stage <= core.StageOutput; stage++ {
		analyzers := r.stages[stage]
		for _, a := range analyzers {
			if !r.enabled(a) {
				continue
			}
			findings := a.Analyze(ctx)
			// An in-place analyzer already appended to ctx.Findings; skip its return.
			if _, inPlace := a.(core.InPlaceAnalyzer); inPlace {
				continue
			}
			ctx.Findings = append(ctx.Findings, findings...)
		}
	}
	return ctx.Findings
}

func (r *StageRunner) enabled(a core.Analyzer) bool {
	if r.cfg == nil {
		return true
	}
	cfg, ok := r.cfg.Analyzers[a.ID()]
	if !ok {
		return true
	}
	return cfg.Enabled
}
