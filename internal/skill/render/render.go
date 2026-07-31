package render

import (
	"bytes"
	"fmt"
	"io/fs"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type Renderer struct {
	base *template.Template
}

func NewRenderer(templates fs.FS) (*Renderer, error) {
	base, err := buildNamespace(templates)
	if err != nil {
		return nil, err
	}
	return &Renderer{base: base}, nil
}

func (renderer *Renderer) Render(idx *knowledge.Index, skill knowledge.Skill) (string, error) {
	sidecar, err := renderSidecar(idx.Body("skills", skill.ID))
	if err != nil {
		return "", fmt.Errorf("skill %q sidecar: %w", skill.ID, err)
	}
	doc := skillDoc{
		ID: skill.ID, Name: skill.Name, Description: skill.Description,
		Triggers: skill.Triggers, AlwaysLoad: skill.AlwaysLoad, Order: skill.Order,
		ComposeSkills: skill.ComposeSkills,
		Sidecar:       sidecar,
		Linters:       composedLinters(idx, skill),
	}
	sources, err := mergeSources(idx, skill)
	if err != nil {
		return "", err
	}
	if skill.Merged() {
		doc.Merged = mergeSections(sources)
	} else {
		doc.Sources = sourceDocs(sources)
	}
	if doc.Workflows, err = renderer.composeWorkflows(idx, skill); err != nil {
		return "", err
	}
	if doc.References, err = composeReferences(idx, skill); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if execErr := renderer.base.ExecuteTemplate(&buf, "SKILL.md.tmpl", doc); execErr != nil {
		return "", fmt.Errorf("render skill %q: %w", skill.ID, execErr)
	}
	return buf.String(), nil
}

func (renderer *Renderer) RenderSkill(
	idx *knowledge.Index,
	skill knowledge.Skill,
	installed []string,
) (string, error) {
	if skill.ID == "project-router" {
		return renderer.RenderRouter(idx, installed)
	}
	return renderer.Render(idx, skill)
}

func (renderer *Renderer) RenderWorkflow(
	idx *knowledge.Index, wf knowledge.Workflow,
) (string, error) {
	doc := workflowDoc{
		ID: wf.ID, Name: wf.Name, Description: wf.Description,
		Steps: wf.Steps,
		Body:  idx.Body("workflows", wf.ID),
	}
	for _, entry := range wf.ComposesRules {
		id := entryID(entry)
		rule, ok := idx.Rule(id)
		if !ok {
			return "", fmt.Errorf("workflow %q: composed rule %q not found", wf.ID, id)
		}
		body, bodyErr := wholeBody(idx.Body("rules", id))
		if bodyErr != nil {
			return "", fmt.Errorf("workflow %q: %w", wf.ID, bodyErr)
		}
		doc.Rules = append(doc.Rules, ref{Name: rule.Name, Body: bodyWithoutH1(body)})
	}
	for _, entry := range wf.ComposesPatterns {
		id := entryID(entry)
		pat, ok := idx.Pattern(id)
		if !ok {
			return "", fmt.Errorf("workflow %q: composed pattern %q not found", wf.ID, id)
		}
		body, bodyErr := wholeBody(idx.Body("patterns", id))
		if bodyErr != nil {
			return "", fmt.Errorf("workflow %q: %w", wf.ID, bodyErr)
		}
		doc.Patterns = append(doc.Patterns, ref{Name: pat.Name, Body: bodyWithoutH1(body)})
	}
	var buf bytes.Buffer
	if err := renderer.base.ExecuteTemplate(&buf, "WORKFLOW.md.tmpl", doc); err != nil {
		return "", fmt.Errorf("render workflow %q: %w", wf.ID, err)
	}
	return buf.String(), nil
}

func composedLinters(idx *knowledge.Index, skill knowledge.Skill) []string {
	var linters []string
	seen := map[string]bool{}
	for _, entry := range skill.ComposeRules {
		id := entryID(entry)
		// Two composed rules commonly name the same linter, so appendUnseen de-duplicates.
		if rule, ok := idx.Rule(id); ok {
			linters = appendUnseen(linters, seen, rule.Linters)
		}
	}
	return linters
}
