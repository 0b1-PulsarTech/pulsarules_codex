package render

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// composeWorkflows transcludes a skill's composed workflows whole (workflows are
// not section-merged).
func (r *Renderer) composeWorkflows(idx *knowledge.Index, skill knowledge.Skill) ([]ref, error) {
	var refs []ref
	for _, id := range skill.ComposeWorkflows {
		workflow, ok := idx.Workflow(id)
		if !ok {
			return nil, fmt.Errorf("skill %q composes unknown workflow %q", skill.ID, id)
		}
		refs = append(refs, ref{
			Name: workflow.Name,
			Body: bodyWithoutH1(idx.Body("workflows", id)),
		})
	}
	return refs, nil
}

// composeReferences gathers the external references cited by a skill's composed
// rules and patterns, de-duplicated in first-seen order, resolved to their data.
func composeReferences(idx *knowledge.Index, skill knowledge.Skill) ([]referenceDoc, error) {
	var ids []string
	seen := map[string]bool{}
	for _, entry := range skill.ComposeRules {
		id, _, _ := strings.Cut(entry, "#")
		if rule, ok := idx.Rule(id); ok {
			ids = appendUnseen(ids, seen, rule.References)
		}
	}
	for _, entry := range skill.ComposePatterns {
		id, _, _ := strings.Cut(entry, "#")
		if pattern, ok := idx.Pattern(id); ok {
			ids = appendUnseen(ids, seen, pattern.References)
		}
	}
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

// appendUnseen appends each id not already in seen to ids, marking it seen.
func appendUnseen(ids []string, seen map[string]bool, add []string) []string {
	for _, id := range add {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

// bodyWithoutH1 drops the leading `# Name` line so a transcluded body renders
// cleanly under its own ### sub-heading.
func bodyWithoutH1(body string) string {
	_, after, _ := strings.Cut(body, "\n")
	return strings.TrimLeft(after, "\n")
}
