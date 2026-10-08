package overrides

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/glob"
)

type rule struct {
	analyzers map[string]pass.OverrideConfig
	files     glob.Set
}

func Apply(findings []finding.Finding, owner string, rules []pass.Override) []finding.Finding {
	if len(rules) == 0 {
		return findings
	}
	return applyRules(findings, owner, prepare(rules))
}

func applyRules(findings []finding.Finding, owner string, prepared []*rule) []finding.Finding {
	kept := make([]finding.Finding, 0, len(findings))
	for _, finding := range findings {
		if resolved, keep := resolve(finding, owner, prepared); keep {
			kept = append(kept, resolved)
		}
	}
	return kept
}

func prepare(rules []pass.Override) []*rule {
	prepared := make([]*rule, len(rules))
	for idx, source := range rules {
		prepared[idx] = &rule{
			analyzers: source.Analyzers,
			files:     glob.CompileSet(source.Files),
		}
	}
	return prepared
}

func resolve(finding finding.Finding, owner string, rules []*rule) (finding.Finding, bool) {
	enabled := true
	for _, item := range rules {
		override, found := lookup(item.analyzers, finding.AnalyzerID, owner)
		if !found || !item.matches(finding.File) {
			continue
		}
		if override.Enabled != nil {
			enabled = *override.Enabled
		}
		if severity, ok := parseSeverity(override.Severity); ok {
			finding.Severity = severity
		}
	}
	return finding, enabled
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

func (item *rule) matches(file string) bool {
	if !item.files.IsOpen() && file == "" {
		return false
	}
	return item.files.Match(file)
}

func parseSeverity(name string) (finding.Severity, bool) {
	switch name {
	case "error":
		return finding.SeverityError, true
	case "warning":
		return finding.SeverityWarning, true
	case "info":
		return finding.SeverityInfo, true
	}
	return finding.SeverityInfo, false
}
