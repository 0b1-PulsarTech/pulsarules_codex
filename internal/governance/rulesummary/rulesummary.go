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
		// A nil index is accepted and leaves the findings untouched.
		if index == nil {
			return findings
		}
		rules := newResolver(index, defaults)
		for idx := range findings {
			hit := &findings[idx]
			if hit.RuleSummary != "" {
				continue
			}
			if hit.RuleID == "" {
				hit.RuleID = rules.resolve(hit.AnalyzerID, hit.File)
			}
			if hit.RuleID != "" {
				hit.RuleSummary = index.Summary(knowledge.KindRule, hit.RuleID)
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
