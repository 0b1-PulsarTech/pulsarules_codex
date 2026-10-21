package governance

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/overrides"
)

type registered struct {
	id       string
	analyzer pass.Analyzer
}

type StageRunner struct {
	stages map[finding.StageID][]registered
	cfg    *pass.AnalysisConfig
}

func newPipeline(cfg *pass.AnalysisConfig) *StageRunner {
	return &StageRunner{
		stages: make(map[finding.StageID][]registered),
		cfg:    cfg,
	}
}

func (runner *StageRunner) Register(id string, analyzer pass.Analyzer) {
	stage := analyzer.Stage()
	runner.stages[stage] = append(runner.stages[stage], registered{id: id, analyzer: analyzer})
}

func (runner *StageRunner) RunStages(ctx *pass.Context) []finding.Finding {
	for stage := finding.StageContext; stage <= finding.StageOutput; stage++ {
		for _, entry := range runner.stages[stage] {
			if !runner.enabled(entry.id) {
				continue
			}
			findings := entry.analyzer.Analyze(ctx)
			// An in-place analyzer already appended to ctx.Findings; skip its return.
			if _, inPlace := entry.analyzer.(pass.InPlaceAnalyzer); inPlace {
				continue
			}
			ctx.Findings = append(ctx.Findings, runner.overridden(entry.id, findings)...)
		}
	}
	return ctx.Findings
}

func (runner *StageRunner) overridden(id string, findings []finding.Finding) []finding.Finding {
	if runner.cfg == nil {
		return findings
	}
	return overrides.Apply(findings, id, runner.cfg.Overrides)
}

func (runner *StageRunner) enabled(id string) bool {
	if runner.cfg == nil {
		return true
	}
	cfg, ok := runner.cfg.Analyzers[id]
	if !ok {
		return true
	}
	return cfg.Enabled
}
