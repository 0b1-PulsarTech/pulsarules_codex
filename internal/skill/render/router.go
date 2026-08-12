package render

import (
	"bytes"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// RenderRouter produces the project-router skill as router.md, filtered to
// installed skills; an empty installed list renders the full router.
func (r *Renderer) RenderRouter(idx *knowledge.Index, installed []string) (string, error) {
	return r.renderRouter(idx, installed, "router.md.tmpl")
}

// RenderCursorRouter produces the project-router skill as a Cursor .mdc rule,
// the RenderCursorRule counterpart for the router's own richer doc shape
// (baseline/dispatch/order tables SKILL.mdc.tmpl's skillDoc has no room for).
func (r *Renderer) RenderCursorRouter(idx *knowledge.Index, installed []string) (string, error) {
	return r.renderRouter(idx, installed, "router.mdc.tmpl")
}

// renderRouter is the shared body behind RenderRouter and RenderCursorRouter.
func (r *Renderer) renderRouter(
	idx *knowledge.Index, installed []string, tmplName string,
) (string, error) {
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
	if err := r.base.ExecuteTemplate(&buf, tmplName, doc); err != nil {
		return "", fmt.Errorf("render router: %w", err)
	}
	return buf.String(), nil
}
