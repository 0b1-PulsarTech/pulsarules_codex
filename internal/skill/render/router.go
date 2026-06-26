package render

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func (renderer *Renderer) RenderRouter(idx *knowledge.Index) (string, error) {
	router, ok := idx.Skill("project-router")
	if !ok {
		return "", errors.New("missing project-router skill")
	}
	doc := routerDoc{
		ID: router.ID, Name: router.Name, Description: router.Description,
		Baseline: idx.Router.Baseline,
		Dispatch: idx.Router.Dispatch,
		Order:    idx.Router.Order,
	}
	for _, skill := range idx.SkillsOrdered() {
		if skill.ID == "project-router" {
			continue
		}
		doc.AvailableSkills = append(doc.AvailableSkills, skillSummary{
			ID: skill.ID, Description: knowledge.FirstSentence(skill.Description),
		})
	}
	var buf bytes.Buffer
	if err := renderer.base.ExecuteTemplate(&buf, "router.md.tmpl", doc); err != nil {
		return "", fmt.Errorf("render router: %w", err)
	}
	return buf.String(), nil
}
