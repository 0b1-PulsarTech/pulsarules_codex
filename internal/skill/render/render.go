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

// Render produces one skill as SKILL.md, merging its composed rules/patterns
// section-by-section (or rendering them per-source when the skill opts out of
// merging).
func (r *Renderer) Render(idx *knowledge.Index, skill knowledge.Skill) (string, error) {
	return r.renderSkill(idx, skill, "SKILL.md.tmpl")
}

// RenderCursorRule produces one skill as a Cursor .mdc rule: the same body
// Render produces, wrapped in Cursor's description/globs/alwaysApply
// frontmatter (the "mdcFrontmatter" partial) instead of SKILL.md's
// name/description block, so a skill's content never diverges between
// hosts - only the frontmatter adapts to the target.
func (r *Renderer) RenderCursorRule(idx *knowledge.Index, skill knowledge.Skill) (string, error) {
	return r.renderSkill(idx, skill, "SKILL.mdc.tmpl")
}

// renderSkill is the shared body behind Render and RenderCursorRule, which
// differ only in which top-level template renders the shared skillDoc.
func (r *Renderer) renderSkill(
	idx *knowledge.Index, skill knowledge.Skill, tmplName string,
) (string, error) {
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
	if doc.Workflows, err = r.composeWorkflows(idx, skill); err != nil {
		return "", err
	}
	if doc.References, err = composeReferences(idx, skill); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if execErr := r.base.ExecuteTemplate(&buf, tmplName, doc); execErr != nil {
		return "", fmt.Errorf("render skill %q: %w", skill.ID, execErr)
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

// RenderSkillCursor is RenderSkill's Cursor counterpart: it dispatches to
// RenderCursorRouter for project-router and RenderCursorRule for all others.
func (r *Renderer) RenderSkillCursor(
	idx *knowledge.Index,
	skill knowledge.Skill,
	installed []string,
) (string, error) {
	if skill.ID == "project-router" {
		return r.RenderCursorRouter(idx, installed)
	}
	return r.RenderCursorRule(idx, skill)
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

// composedLinters gathers the linters of a skill's composed rules for the
// footer, de-duplicated in first-seen order via appendUnseen (compose.go):
// two composed rules commonly name the same linter.
func composedLinters(idx *knowledge.Index, skill knowledge.Skill) []string {
	var linters []string
	seen := map[string]bool{}
	for _, entry := range skill.ComposeRules {
		id, _, _ := strings.Cut(entry, "#")
		if rule, ok := idx.Rule(id); ok {
			linters = appendUnseen(linters, seen, rule.Linters)
		}
	}
	return linters
}
