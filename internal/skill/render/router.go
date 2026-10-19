package render

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func (renderer *Renderer) renderRouterDoc(
	idx *knowledge.Index,
	installed []string,
) (string, error) {
	return renderer.renderRouter(idx, installed, "router.md.tmpl")
}

func (renderer *Renderer) renderCursorRouter(
	idx *knowledge.Index, installed []string,
) (string, error) {
	return renderer.renderRouter(idx, installed, "router.mdc.tmpl")
}

func (renderer *Renderer) renderRouter(
	idx *knowledge.Index, installed []string, tmplName string,
) (string, error) {
	router, ok := idx.Skill("project-router")
	if !ok {
		return "", errors.New("missing project-router skill")
	}
	keep := installFilter(installed)
	doc := routerDoc{
		ID: router.ID, Name: router.Name, Description: router.Description,
		Baseline:        filterBaseline(idx.Router.Baseline, keep),
		Dispatch:        filterDispatch(idx.Router.Dispatch, keep),
		Order:           filterOrder(idx.Router.Order, keep),
		ShowTestCallout: keep("integration-tests"),
	}
	var buf bytes.Buffer
	if err := renderer.base.ExecuteTemplate(&buf, tmplName, doc); err != nil {
		return "", fmt.Errorf("render router: %w", err)
	}
	return buf.String(), nil
}
