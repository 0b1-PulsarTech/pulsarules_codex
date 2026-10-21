package governance

import (
	"cmp"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/overrides"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
)

type Transform func([]finding.Finding) []finding.Finding

type pipeline struct {
	analyzers  []registry.Built
	cfg        *pass.AnalysisConfig
	transforms []Transform
}

func newPipeline(
	cfg *pass.AnalysisConfig, analyzers []registry.Built, transforms []Transform,
) *pipeline {
	return &pipeline{analyzers: analyzers, cfg: cfg, transforms: transforms}
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
	findings := ctx.Findings
	for _, transform := range pipe.transforms {
		findings = transform(findings)
	}
	return findings
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

func orderFindings(findings []finding.Finding) []finding.Finding {
	slices.SortStableFunc(findings, func(left, right finding.Finding) int {
		return cmp.Or(
			cmp.Compare(right.Severity, left.Severity),
			strings.Compare(left.File, right.File),
			cmp.Compare(left.Line, right.Line),
		)
	})
	return slices.CompactFunc(findings, func(left, right finding.Finding) bool {
		return left.AnalyzerID == right.AnalyzerID &&
			left.File == right.File &&
			left.Line == right.Line &&
			left.Message == right.Message
	})
}
