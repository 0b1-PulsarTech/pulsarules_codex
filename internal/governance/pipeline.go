package governance

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

type StageRunner struct {
	stages map[pass.StageID][]pass.Analyzer
	cfg    *pass.AnalysisConfig
}

func newPipeline(cfg *pass.AnalysisConfig) *StageRunner {
	return &StageRunner{
		stages: make(map[pass.StageID][]pass.Analyzer),
		cfg:    cfg,
	}
}

func (r *StageRunner) Register(a pass.Analyzer) {
	r.stages[a.Stage()] = append(r.stages[a.Stage()], a)
}

func (r *StageRunner) RunStages(ctx *pass.Context) []pass.Finding {
	for stage := pass.StageContext; stage <= pass.StageOutput; stage++ {
		analyzers := r.stages[stage]
		for _, a := range analyzers {
			if !r.enabled(a) {
				continue
			}
			findings := a.Analyze(ctx)
			ctx.Findings = append(ctx.Findings, findings...)
		}
	}
	return ctx.Findings
}

func (r *StageRunner) enabled(a pass.Analyzer) bool {
	if r.cfg == nil {
		return true
	}
	cfg, ok := r.cfg.Analyzers[a.ID()]
	if !ok {
		return true
	}
	return cfg.Enabled
}
