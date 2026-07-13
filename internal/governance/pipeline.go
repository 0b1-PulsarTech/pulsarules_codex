package governance

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type StageRunner struct {
	stages map[finding.StageID][]pass.Analyzer
	cfg    *pass.AnalysisConfig
}

func newPipeline(cfg *pass.AnalysisConfig) *StageRunner {
	return &StageRunner{
		stages: make(map[finding.StageID][]pass.Analyzer),
		cfg:    cfg,
	}
}

func (r *StageRunner) Register(a pass.Analyzer) {
	r.stages[a.Stage()] = append(r.stages[a.Stage()], a)
}

func (r *StageRunner) RunStages(ctx *pass.Context) []finding.Finding {
	for stage := finding.StageContext; stage <= finding.StageOutput; stage++ {
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
