package overrides

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func Apply(findings []finding.Finding, owner string, rules []pass.Override) []finding.Finding {
	if len(rules) == 0 {
		return findings
	}
	return applyRules(findings, owner, rules)
}

func applyRules(findings []finding.Finding, owner string, rules []pass.Override) []finding.Finding {
	kept := make([]finding.Finding, 0, len(findings))
	for _, finding := range findings {
		if resolved, keep := resolve(finding, owner, rules); keep {
			kept = append(kept, resolved)
		}
	}
	return kept
}

func resolve(
	findingItem finding.Finding,
	owner string,
	rules []pass.Override,
) (finding.Finding, bool) {
	enabled := true
	for _, item := range rules {
		override, found := lookup(item.Analyzers, findingItem.AnalyzerID, owner)
		if !found || !item.Covers(findingItem.File) {
			continue
		}
		if override.Enabled != nil {
			enabled = *override.Enabled
		}
		if severity, err := finding.ParseSeverity(override.Severity); err == nil {
			findingItem.Severity = severity
		}
	}
	return findingItem, enabled
}

func lookup(
	analyzers map[string]pass.OverrideConfig, id, owner string,
) (pass.OverrideConfig, bool) {
	if base, _, hasSub := strings.Cut(id, "/"); hasSub {
		if override, found := analyzers[id]; found {
			return override, true
		}
		if override, found := analyzers[base]; found {
			return override, true
		}
	}
	// A bare finding ID names its analyzer only through the owner, so a delegated key that
	// spells another analyzer's finding ID cannot capture it.
	override, found := analyzers[owner]
	return override, found
}
