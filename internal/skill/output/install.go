package output

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type InstallResult struct {
	Installed []string
	Skipped   []string
	BackedUp  []string
}

type InstallRequest struct {
	IDs          []string
	RouterFilter []string
}

func Install(
	idx *knowledge.Index,
	rnd *render.Renderer,
	dest string,
	req InstallRequest,
) (InstallResult, error) {
	return installDocs(dest, "SKILL.md", req.IDs, func(id string) (string, bool, error) {
		skill, ok := idx.Skill(id)
		if !ok {
			return "", false, nil
		}
		body, renderErr := rnd.RenderSkill(idx, skill, req.RouterFilter)
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
) (InstallResult, error) {
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
) (InstallResult, error) {
	var result InstallResult
	for _, id := range ids {
		body, ok, renderErr := render(id)
		if renderErr != nil {
			// The render callbacks already wrap their errors with the id; do not wrap again.
			return result, renderErr
		}
		if !ok {
			result.Skipped = append(result.Skipped, id)
			continue
		}
		docBackups, writeErr := WriteDoc(filepath.Join(dest, id), docName, body)
		if writeErr != nil {
			return result, fmt.Errorf("install %q: %w", id, writeErr)
		}
		result.BackedUp = append(result.BackedUp, docBackups...)
		result.Installed = append(result.Installed, id)
	}
	return result, nil
}
