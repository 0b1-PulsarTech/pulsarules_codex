package render

import (
	"fmt"
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
