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
	idx *knowledge.Index, skill knowledge.Skill,
) (string, error) {
	if skill.ID == "project-router" {
		return renderer.RenderRouter(idx)
	}
	return renderer.Render(idx, skill)
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
