package validate

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// referencesResolve reports any rule or pattern that cites an unknown reference id.
func referencesResolve(idx *knowledge.Index) []string {
	var problems []string
	for _, rule := range idx.Rules {
		problems = append(problems, runCompositions([]composition{{
			refs:    rule.References,
			resolve: idx.ReferenceExists,
			missing: func(ref string) string {
				return fmt.Sprintf("rule %q cites unknown reference %q", rule.ID, ref)
			},
		}})...)
	}
	for _, pattern := range idx.Patterns {
		problems = append(problems, runCompositions([]composition{{
			refs:    pattern.References,
			resolve: idx.ReferenceExists,
			missing: func(ref string) string {
				return fmt.Sprintf("pattern %q cites unknown reference %q", pattern.ID, ref)
			},
		}})...)
	}
	return problems
}
