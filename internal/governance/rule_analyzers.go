package governance

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func RuleAnalyzersCheck(idx *knowledge.Index) []string {
	ids := IDs()
	known := make(map[string]bool, len(ids))
	for _, id := range ids {
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
