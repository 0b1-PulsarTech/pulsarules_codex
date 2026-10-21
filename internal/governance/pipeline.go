package governance

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/overrides"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
)

type pipeline struct {
	analyzers []registry.Built
	cfg       *pass.AnalysisConfig
}

func newPipeline(cfg *pass.AnalysisConfig, analyzers []registry.Built) *pipeline {
	return &pipeline{analyzers: analyzers, cfg: cfg}
}

func (pipe *pipeline) run(ctx *pass.Context) []finding.Finding {
	for _, entry := range pipe.analyzers {
		if !pipe.enabled(entry.ID) {
			continue
		}
		findings := entry.Analyzer.Analyze(ctx)
		// An in-place analyzer already appended to ctx.Findings; skip its return.
		if _, inPlace := entry.Analyzer.(pass.InPlaceAnalyzer); inPlace {
			continue
		}
		ctx.Findings = append(ctx.Findings, pipe.overridden(entry.ID, findings)...)
	}
	return ctx.Findings
}

func (pipe *pipeline) overridden(id string, findings []finding.Finding) []finding.Finding {
	if pipe.cfg == nil {
		return findings
	}
	return overrides.Apply(findings, id, pipe.cfg.Overrides)
}

func (pipe *pipeline) enabled(id string) bool {
	if pipe.cfg == nil {
		return true
	}
	cfg, ok := pipe.cfg.Analyzers[id]
	if !ok {
		return true
	}
	return cfg.Enabled
}
