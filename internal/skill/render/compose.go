package render

import (
	"fmt"
	"iter"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ordered"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func (renderer *Renderer) composeWorkflows(
	idx *knowledge.Index, skill knowledge.Skill,
) ([]ref, error) {
	var refs []ref
	for _, id := range skill.ComposeWorkflows {
		workflow, ok := idx.Workflow(id)
		if !ok {
			return nil, fmt.Errorf("skill %q composes unknown workflow %q", skill.ID, id)
		}
		refs = append(refs, ref{
			Name: workflow.Name,
			Body: bodyWithoutH1(idx.Body(knowledge.KindWorkflow, id)),
		})
	}
	return refs, nil
}

func composeReferences(idx *knowledge.Index, skill knowledge.Skill) ([]referenceDoc, error) {
	var ids []string
	for entry := range composed(idx, skill.ComposeRules, skill.ComposePatterns) {
		ids = append(ids, entry.References...)
	}
	ids = ordered.Dedupe(ids)
	docs := make([]referenceDoc, 0, len(ids))
	for _, id := range ids {
		reference, ok := idx.Reference(id)
		if !ok {
			return nil, fmt.Errorf("skill %q cites unknown reference %q", skill.ID, id)
		}
		docs = append(docs, referenceDoc{
			Title: reference.Title, URL: reference.URL, Note: reference.Note,
		})
	}
	return docs, nil
}

func bodyWithoutH1(body string) string {
	_, after, _ := strings.Cut(body, "\n")
	return strings.TrimLeft(after, "\n")
}

func entryID(entry string) string {
	id, _, _ := strings.Cut(entry, "#")
	return id
}

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
