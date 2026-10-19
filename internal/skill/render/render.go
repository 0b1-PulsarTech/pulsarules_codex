package render

import (
	"bytes"
	"fmt"
	"io/fs"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ordered"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type Renderer struct {
	base   *template.Template
	parser *bodyParser
}

func NewRenderer(templates fs.FS) (*Renderer, error) {
	parser := newBodyParser()
	base, err := buildNamespace(templates, parser.funcs)
	if err != nil {
		return nil, err
	}
	return &Renderer{base: base, parser: parser}, nil
}

func (renderer *Renderer) renderBody(idx *knowledge.Index, skill knowledge.Skill) (string, error) {
	return renderer.renderSkill(idx, skill, "SKILL.md.tmpl")
}

func (renderer *Renderer) renderCursorRule(
	idx *knowledge.Index, skill knowledge.Skill,
) (string, error) {
	return renderer.renderSkill(idx, skill, "SKILL.mdc.tmpl")
}

func (renderer *Renderer) renderSkill(
	idx *knowledge.Index, skill knowledge.Skill, tmplName string,
) (string, error) {
	sidecar, err := renderer.parser.sidecar(idx.Body(knowledge.KindSkill, skill.ID))
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
	sources, err := mergeSources(renderer.parser, idx, skill)
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
	if execErr := renderer.base.ExecuteTemplate(&buf, tmplName, doc); execErr != nil {
		return "", fmt.Errorf("render skill %q: %w", skill.ID, execErr)
	}
	return buf.String(), nil
}

func (renderer *Renderer) RenderSkill(
	idx *knowledge.Index,
	skill knowledge.Skill,
	installed []string,
) (string, error) {
	if skill.ID == knowledge.RouterSkillID {
		return renderer.renderRouterDoc(idx, installed)
	}
	return renderer.renderBody(idx, skill)
}

func (renderer *Renderer) RenderSkillCursor(
	idx *knowledge.Index,
	skill knowledge.Skill,
	installed []string,
) (string, error) {
	if skill.ID == knowledge.RouterSkillID {
		return renderer.renderCursorRouter(idx, installed)
	}
	return renderer.renderCursorRule(idx, skill)
}

func (renderer *Renderer) RenderWorkflow(
	idx *knowledge.Index, wf knowledge.Workflow,
) (string, error) {
	doc := workflowDoc{
		ID: wf.ID, Name: wf.Name, Description: wf.Description,
		Steps: wf.Steps,
		Body:  idx.Body(knowledge.KindWorkflow, wf.ID),
	}
	for entry := range composed(idx, wf.ComposesRules, wf.ComposesPatterns) {
		if !entry.Found {
			return "", fmt.Errorf(
				"workflow %q: composed %s %q not found", wf.ID, entry.Label, entry.ID,
			)
		}
		body, bodyErr := renderer.parser.whole(idx.Body(entry.Kind, entry.ID))
		if bodyErr != nil {
			return "", fmt.Errorf("workflow %q: %w", wf.ID, bodyErr)
		}
		item := ref{Name: entry.Name, Body: bodyWithoutH1(body)}
		if entry.Kind == knowledge.KindRule {
			doc.Rules = append(doc.Rules, item)
		} else {
			doc.Patterns = append(doc.Patterns, item)
		}
	}
	var buf bytes.Buffer
	if err := renderer.base.ExecuteTemplate(&buf, "WORKFLOW.md.tmpl", doc); err != nil {
		return "", fmt.Errorf("render workflow %q: %w", wf.ID, err)
	}
	return buf.String(), nil
}

func composedLinters(idx *knowledge.Index, skill knowledge.Skill) []string {
	var linters []string
	for entry := range composed(idx, skill.ComposeRules, nil) {
		linters = append(linters, entry.Linters...)
	}
	// Two composed rules commonly name the same linter.
	return ordered.Dedupe(linters)
}
