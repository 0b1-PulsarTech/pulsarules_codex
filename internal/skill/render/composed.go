package render

import (
	"iter"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type member struct {
	Kind                knowledge.Kind
	Label, ID, Name     string
	Linters, References []string
	Found               bool
}

func composed(idx *knowledge.Index, rules, patterns []string) iter.Seq[member] {
	return func(yield func(member) bool) {
		// A dangling id still yields (Found false): each caller words that failure itself.
		for _, entry := range rules {
			id := entryID(entry)
			rule, found := idx.Rule(id)
			item := member{
				Kind: knowledge.KindRule, Label: "rule", ID: id, Name: rule.Name,
				Linters: rule.Linters, References: rule.References, Found: found,
			}
			if !yield(item) {
				return
			}
		}
		for _, entry := range patterns {
			id := entryID(entry)
			pattern, found := idx.Pattern(id)
			item := member{
				Kind: knowledge.KindPattern, Label: "pattern", ID: id, Name: pattern.Name,
				References: pattern.References, Found: found,
			}
			if !yield(item) {
				return
			}
		}
	}
}
