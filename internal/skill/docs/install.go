package docs

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func Install(
	idx *knowledge.Index,
	rnd *render.Renderer,
	dest string,
	ids []string,
	routerFilter []string,
) (installed, skipped []string, err error) {
	return installDocs(dest, "SKILL.md", ids, func(id string) (string, bool, error) {
		skill, ok := idx.Skill(id)
		if !ok {
			return "", false, nil
		}
		body, renderErr := rnd.RenderSkill(idx, skill, routerFilter)
		if renderErr != nil {
			return "", true, fmt.Errorf("render %q: %w", id, renderErr)
		}
		return body, true, nil
	})
}

func InstallWorkflows(
	idx *knowledge.Index,
	rnd *render.Renderer,
	dest string,
	wfIDs []string,
) (installed, skipped []string, err error) {
	return installDocs(dest, "WORKFLOW.md", wfIDs, func(id string) (string, bool, error) {
		wf, ok := idx.Workflow(id)
		if !ok {
			return "", false, nil
		}
		body, renderErr := rnd.RenderWorkflow(idx, wf)
		if renderErr != nil {
			return "", true, fmt.Errorf("render workflow %q: %w", id, renderErr)
		}
		return body, true, nil
	})
}

func installDocs(
	dest, docName string,
	ids []string,
	render func(id string) (body string, ok bool, err error),
) (installed, skipped []string, err error) {
	for _, id := range ids {
		body, ok, renderErr := render(id)
		if renderErr != nil {
			// The render callbacks already wrap their errors with the id; do not wrap again.
			return installed, skipped, renderErr
		}
		if !ok {
			skipped = append(skipped, id)
			continue
		}
		if writeErr := Write(filepath.Join(dest, id), docName, body); writeErr != nil {
			return installed, skipped, fmt.Errorf("install %q: %w", id, writeErr)
		}
		installed = append(installed, id)
	}
	return installed, skipped, nil
}
