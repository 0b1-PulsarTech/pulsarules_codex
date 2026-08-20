package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// StageRunner runs registered analyzers grouped by stage.
//
// why: cfg once lived in a second, empty *core.AnalysisContext built only
// to carry it and prone to drift from the real context RunStages
// evaluates; storing cfg directly removes that duplicate.
type StageRunner struct {
	stages map[core.StageID][]core.Analyzer
	cfg    *core.AnalysisConfig
}

// NewStageRunner creates an empty stage runner with the given config.
// Analyzers are registered via Register.
func NewStageRunner(cfg *core.AnalysisConfig) *StageRunner {
	return &StageRunner{
		stages: make(map[core.StageID][]core.Analyzer),
		cfg:    cfg,
	}
}

func (r *StageRunner) Register(a core.Analyzer) {
	r.stages[a.Stage()] = append(r.stages[a.Stage()], a)
}

// RunStages executes enabled analyzers in stage order, returning findings.
//
// why: a core.InPlaceAnalyzer already mutates ctx.Findings itself;
// appending its return on top would double it, so the marker check makes
// that contract explicit instead of an unenforced convention.
func (r *StageRunner) RunStages(ctx *core.AnalysisContext) []core.Finding {
	for stage := core.StageContext; stage <= core.StageOutput; stage++ {
		analyzers := r.stages[stage]
		for _, a := range analyzers {
			if !r.enabled(a) {
				continue
			}
			findings := a.Analyze(ctx)
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
