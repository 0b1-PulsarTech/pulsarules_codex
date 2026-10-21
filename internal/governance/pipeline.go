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

type Pipeline struct {
	analyzers []registered
	cfg       *pass.AnalysisConfig
}

func NewPipeline(cfg *pass.AnalysisConfig) *Pipeline {
	return &Pipeline{cfg: cfg}
}

func (pipeline *Pipeline) Register(id string, analyzer pass.Analyzer) {
	pipeline.analyzers = append(pipeline.analyzers, registered{id: id, analyzer: analyzer})
}

func (pipeline *Pipeline) Run(ctx *pass.Context) []finding.Finding {
	for _, entry := range pipeline.analyzers {
		if !pipeline.enabled(entry.id) {
			continue
		}
		findings := entry.analyzer.Analyze(ctx)
		// An in-place analyzer already appended to ctx.Findings; skip its return.
		if _, inPlace := entry.analyzer.(pass.InPlaceAnalyzer); inPlace {
			continue
		}
		ctx.Findings = append(ctx.Findings, pipeline.overridden(entry.id, findings)...)
	}
	return ctx.Findings
}

func (pipeline *Pipeline) overridden(id string, findings []finding.Finding) []finding.Finding {
	if pipeline.cfg == nil {
		return findings
	}
	return overrides.Apply(findings, id, pipeline.cfg.Overrides)
}

func (pipeline *Pipeline) enabled(id string) bool {
	if pipeline.cfg == nil {
		return true
	}
	cfg, ok := pipeline.cfg.Analyzers[id]
	if !ok {
		return true
	}
	return cfg.Enabled
}
