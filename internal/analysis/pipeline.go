package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// StageRunner holds the analyzers grouped by stage. Each stage runs after the
// previous one, and its analyzers receive the accumulated context. Adding a
// new rule is one file (implementing core.Analyzer) plus one line in boot.go.
//
// why: cfg used to live wrapped in a second, otherwise-empty
// *core.AnalysisContext built solely to carry it - a shadow of the real
// context RunStages(ctx) evaluates analyzers against, and nothing kept the
// two in sync. Storing the config directly removes that second context.
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

// RunStages executes all enabled analyzers in stage order, accumulating
// findings in the context. It returns the aggregated findings.
//
// why: a core.InPlaceAnalyzer (ruleinjection, output) already transforms
// ctx.Findings itself; appending its Analyze return value on top would
// double whatever it returned. Checking the marker interface, rather than
// trusting every such analyzer to return nil, makes that contract explicit
// instead of an unenforced convention.
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
