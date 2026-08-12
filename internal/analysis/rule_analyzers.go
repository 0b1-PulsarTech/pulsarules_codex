package analysis

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func RuleAnalyzersCheck(idx *knowledge.Index) []string {
	known := make(map[string]bool, len(analyzerSpecs()))
	for _, id := range RegisteredAnalyzerIDs() {
		known[id] = true
	}

	problems := make([]string, 0, len(idx.Rules))
	for _, rule := range idx.Rules {
		for _, ref := range rule.Analyzers {
			if !known[ref] {
				problems = append(
					problems,
					fmt.Sprintf("rule %q declares unknown analyzer %q", rule.ID, ref),
				)
			}
		}
	}
	return problems
}
