package validate

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func canonicalSections(idx *knowledge.Index) []string {
	var problems []string
	check := func(kind knowledge.Kind, id string) {
		names, err := render.UnknownSections(idx.Body(kind, id))
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s %q: %v", kind, id, err))
			return
		}
		for _, name := range names {
			problems = append(problems, fmt.Sprintf(
				"%s %q defines non-canonical section %q", kind, id, name,
			))
		}
	}
	for _, rule := range idx.Rules {
		check(knowledge.KindRule, rule.ID)
	}
	for _, pattern := range idx.Patterns {
		check(knowledge.KindPattern, pattern.ID)
	}
	return problems
}
