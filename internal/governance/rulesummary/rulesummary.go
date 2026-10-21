package rulesummary

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func Attach(
	index *knowledge.Index, defaults map[string]string,
) func([]finding.Finding) []finding.Finding {
	return func(findings []finding.Finding) []finding.Finding {
		// A nil index is accepted and skips injection.
		if index == nil {
			return findings
		}
		rules := newResolver(index, defaults)
		for idx := range findings {
			finding := &findings[idx]
			if finding.RuleSummary != "" {
				continue
			}
			if finding.RuleID == "" {
				finding.RuleID = rules.resolve(finding.AnalyzerID, finding.File)
			}
			if finding.RuleID != "" {
				finding.RuleSummary = index.Summary(knowledge.KindRule, finding.RuleID)
			}
		}
		return findings
	}
}

func lookupRule(rules map[string]string, analyzerID string) string {
	if rid, ok := rules[analyzerID]; ok {
		return rid
	}
	// Sub-IDs like "golangci-lint/gocyclo" come from the delegated linter.
	if strings.HasPrefix(analyzerID, "golangci-lint/") {
		return rules["golangci-lint"]
	}
	return ""
}
