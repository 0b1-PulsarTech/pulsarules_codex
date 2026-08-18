package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// StageRunner holds the analyzers grouped by stage. Each stage runs after the
// previous one, and its analyzers receive the accumulated context. Adding a
// new rule is one file (implementing core.Analyzer) plus one line in boot.go.
type StageRunner struct {
	stages  map[core.StageID][]core.Analyzer
	context *core.AnalysisContext
}

// NewStageRunner creates an empty stage runner with the given config.
// Analyzers are registered via Register.
func NewStageRunner(cfg *core.AnalysisConfig) *StageRunner {
	return &StageRunner{
		stages:  make(map[core.StageID][]core.Analyzer),
		context: &core.AnalysisContext{Config: cfg},
	}
}

func (r *StageRunner) Register(a core.Analyzer) {
	r.stages[a.Stage()] = append(r.stages[a.Stage()], a)
}

// RunStages executes all enabled analyzers in stage order, accumulating
// findings in the context. It returns the aggregated findings.
func (r *StageRunner) RunStages(ctx *core.AnalysisContext) []core.Finding {
	for stage := core.StageContext; stage <= core.StageOutput; stage++ {
		analyzers := r.stages[stage]
		for _, a := range analyzers {
			if !r.enabled(a) {
				continue
			}
			if !r.requirementsMet(a, ctx) {
				continue
			}
			findings := a.Analyze(ctx)
			ctx.Findings = append(ctx.Findings, findings...)
		}
	}
	return ctx.Findings
}

func (r *StageRunner) enabled(a core.Analyzer) bool {
	if r.context == nil || r.context.Config == nil {
		return true
	}
	cfg, ok := r.context.Config.Analyzers[a.ID()]
	if !ok {
		return true
	}
	return cfg.Enabled
}

func (r *StageRunner) requirementsMet(a core.Analyzer, ctx *core.AnalysisContext) bool {
	req := a.Needs()
	if req.NeedsAST && ctx.ChangedFiles == nil {
		return false
	}
	if req.NeedsGitHistory && len(ctx.GitHistory) == 0 {
		return false
	}
	return true
}
