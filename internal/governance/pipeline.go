package governance

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/overrides"
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

func (runner *StageRunner) Register(analyzer pass.Analyzer) {
	runner.stages[analyzer.Stage()] = append(runner.stages[analyzer.Stage()], analyzer)
}

func (runner *StageRunner) RunStages(ctx *pass.Context) []finding.Finding {
	for stage := finding.StageContext; stage <= finding.StageOutput; stage++ {
		analyzers := runner.stages[stage]
		for _, analyzer := range analyzers {
			if !runner.enabled(analyzer) {
				continue
			}
			findings := analyzer.Analyze(ctx)
			// An in-place analyzer already appended to ctx.Findings; skip its return.
			if _, inPlace := analyzer.(pass.InPlaceAnalyzer); inPlace {
				continue
			}
			ctx.Findings = append(ctx.Findings, runner.overridden(analyzer, findings)...)
		}
	}
	return ctx.Findings
}

func (runner *StageRunner) overridden(
	analyzer pass.Analyzer,
	findings []finding.Finding,
) []finding.Finding {
	if runner.cfg == nil {
		return findings
	}
	return overrides.Apply(findings, analyzer.ID(), runner.cfg.Overrides)
}

func (runner *StageRunner) enabled(analyzer pass.Analyzer) bool {
	if runner.cfg == nil {
		return true
	}
	cfg, ok := runner.cfg.Analyzers[analyzer.ID()]
	if !ok {
		return true
	}
	return cfg.Enabled
}
