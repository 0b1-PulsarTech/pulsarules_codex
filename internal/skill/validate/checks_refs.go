package validate

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func hostComposition(
	kind, owner, phrase string, refs []string, resolve func(string) bool,
) composition {
	return composition{
		refs:    refs,
		resolve: resolve,
		missing: func(ref string) string {
			return fmt.Sprintf("%s %q %s %q", kind, owner, phrase, ref)
		},
	}
}

func crossReferences(idx *knowledge.Index) []string {
	comps := make([]composition, 0, len(idx.Rules)+len(idx.Patterns))
	for _, rule := range idx.Rules {
		comps = append(comps, hostComposition(
			"rule", rule.ID, "depends on unknown rule", rule.Dependencies, idx.RuleExists,
		))
	}
	for _, pattern := range idx.Patterns {
		comps = append(comps, hostComposition(
			"pattern", pattern.ID, "depends on unknown rule", pattern.Dependencies, idx.RuleExists,
		))
	}
	for _, pattern := range idx.Patterns {
		comps = append(comps, hostComposition(
			"pattern", pattern.ID, "composes unknown pattern", pattern.Composes, idx.PatternExists,
		))
	}
	for _, rule := range idx.Rules {
		comps = append(comps, hostComposition(
			"rule", rule.ID, "cites unknown reference", rule.References, idx.ReferenceExists,
		))
	}
	for _, pattern := range idx.Patterns {
		comps = append(comps, hostComposition(
			"pattern", pattern.ID, "cites unknown reference", pattern.References,
			idx.ReferenceExists,
		))
	}
	return runCompositions(comps)
}
