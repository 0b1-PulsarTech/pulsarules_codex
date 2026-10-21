package governance

import (
	"cmp"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/packagevar"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/overrides"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/rulesummary"
)

type transform func([]finding.Finding) []finding.Finding

type pipeline struct {
	analyzers  []registry.Built
	cfg        *pass.AnalysisConfig
	transforms []transform
}

func newPipeline(
	cfg *pass.AnalysisConfig, analyzers []registry.Built, transforms []transform,
) *pipeline {
	return &pipeline{analyzers: analyzers, cfg: cfg, transforms: transforms}
}

func (pipe *pipeline) run(ctx *pass.Context, seed []finding.Finding) []finding.Finding {
	findings := slices.Clone(seed)
	for _, entry := range pipe.analyzers {
		if !pipe.enabled(entry.ID) {
			continue
		}
		reported := entry.Analyzer.Analyze(ctx)
		findings = append(findings, pipe.overridden(entry.ID, reported)...)
	}
	for _, apply := range pipe.transforms {
		findings = apply(findings)
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

func (sess *Session) transforms() []transform {
	return []transform{
		packagevar.DropSingleUseDuplicates,
		rulesummary.Attach(sess.index, registry.Rules()),
		orderFindings,
	}
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
