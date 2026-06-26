package render

import (
	"bytes"
	"fmt"
	"io/fs"
	"strings"
	"text/template"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// Renderer renders skills from the index and the embedded templates.
type Renderer struct {
	base *template.Template
}

// NewRenderer builds the shared template namespace from the templates filesystem.
func NewRenderer(templates fs.FS) (*Renderer, error) {
	base, err := buildNamespace(templates)
	if err != nil {
		return nil, err
	}
	return &Renderer{base: base}, nil
}

// Render produces one skill, merging its composed rules/patterns section-by-section
// (or rendering them per-source when the skill opts out of merging).
func (r *Renderer) Render(idx *knowledge.Index, skill knowledge.Skill) (string, error) {
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
	sources, err := r.mergeSources(idx, skill)
	if err != nil {
		return "", err
	}
	if skill.Merged() {
		doc.Merged = mergeSections(sources)
	} else {
		doc.Sources = sourceDocs(sources)
	}
	if doc.Workflows, err = r.composeWorkflows(idx, skill); err != nil {
		return "", err
	}
	if doc.References, err = composeReferences(idx, skill); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if execErr := r.base.ExecuteTemplate(&buf, "SKILL.md.tmpl", doc); execErr != nil {
		return "", fmt.Errorf("render skill %q: %w", skill.ID, execErr)
	}
	return buf.String(), nil
}

// RenderRouter produces the project-router skill filtered to installed skills;
// an empty installed list renders the full router.
func (r *Renderer) RenderRouter(idx *knowledge.Index, installed []string) (string, error) {
	router, ok := idx.Skill("project-router")
	if !ok {
		return "", fmt.Errorf("missing project-router skill")
	}
	keep := installFilter(installed)
	doc := routerDoc{
		ID: router.ID, Name: router.Name, Description: router.Description,
		Baseline:        filterBaseline(idx.Router.Baseline, keep),
		Dispatch:        filterDispatch(idx.Router.Dispatch, keep),
		Order:           filterOrder(idx.Router.Order, keep),
		ShowTestCallout: keep("integration-tests"),
	}
	for _, skill := range idx.SkillsOrdered() {
		if skill.ID == "project-router" || !keep(skill.ID) {
			continue
		}
		doc.AvailableSkills = append(doc.AvailableSkills, skillSummary{
			ID: skill.ID, Description: knowledge.FirstSentence(skill.Description),
		})
	}
	var buf bytes.Buffer
	if err := r.base.ExecuteTemplate(&buf, "router.md.tmpl", doc); err != nil {
		return "", fmt.Errorf("render router: %w", err)
	}
	return buf.String(), nil
}

// RenderSkill dispatches to RenderRouter for project-router; renders normally for all others.
func (r *Renderer) RenderSkill(
	idx *knowledge.Index,
	skill knowledge.Skill,
	installed []string,
) (string, error) {
	if skill.ID == "project-router" {
		return r.RenderRouter(idx, installed)
	}
	return r.Render(idx, skill)
}

// RenderWorkflow renders one workflow into a WORKFLOW.md string.
func (r *Renderer) RenderWorkflow(idx *knowledge.Index, wf knowledge.Workflow) (string, error) {
	doc := workflowDoc{
		ID: wf.ID, Name: wf.Name, Description: wf.Description,
		Steps: wf.Steps,
		Body:  idx.Body("workflows", wf.ID),
	}
	for _, entry := range wf.ComposesRules {
		id, _, _ := strings.Cut(entry, "#")
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
		id, _, _ := strings.Cut(entry, "#")
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
	if err := r.base.ExecuteTemplate(&buf, "WORKFLOW.md.tmpl", doc); err != nil {
		return "", fmt.Errorf("render workflow %q: %w", wf.ID, err)
	}
	return buf.String(), nil
}

// composedLinters gathers the linters of a skill's composed rules for the footer.
func composedLinters(idx *knowledge.Index, skill knowledge.Skill) []string {
	var linters []string
	for _, entry := range skill.ComposeRules {
		id, _, _ := strings.Cut(entry, "#")
		if rule, ok := idx.Rule(id); ok {
			linters = append(linters, rule.Linters...)
		}
	}
	return linters
}
