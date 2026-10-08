package overrides

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/pathglob"
)

type rule struct {
	analyzers map[string]core.OverrideConfig
	files     pathglob.Set
}

func Apply(findings []core.Finding, owner string, rules []core.Override) []core.Finding {
	if len(rules) == 0 {
		return findings
	}
	return applyRules(findings, owner, prepare(rules))
}

func applyRules(findings []core.Finding, owner string, prepared []*rule) []core.Finding {
	kept := make([]core.Finding, 0, len(findings))
	for _, finding := range findings {
		if resolved, keep := resolve(finding, owner, prepared); keep {
			kept = append(kept, resolved)
		}
	}
	return kept
}

func prepare(rules []core.Override) []*rule {
	prepared := make([]*rule, len(rules))
	for idx, source := range rules {
		prepared[idx] = &rule{
			analyzers: source.Analyzers,
			files:     pathglob.CompileSet(source.Files),
		}
	}
	return prepared
}

func resolve(finding core.Finding, owner string, rules []*rule) (core.Finding, bool) {
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
	analyzers map[string]core.OverrideConfig, id, owner string,
) (core.OverrideConfig, bool) {
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

func parseSeverity(name string) (core.Severity, bool) {
	switch name {
	case "error":
		return core.SeverityError, true
	case "warning":
		return core.SeverityWarning, true
	case "info":
		return core.SeverityInfo, true
	}
	return core.SeverityInfo, false
}
