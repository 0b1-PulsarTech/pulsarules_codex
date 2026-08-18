package output

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// Install renders each selected skill in-memory and writes it to
// <dest>/<id>/SKILL.md plus a sibling .gitignore (see WriteDoc). Unknown ids
// are skipped and reported. routerFilter trims the rendered project-router to
// those skills (empty = the full router). backedUp carries a ready-to-print
// message for every foreign file WriteDoc backed up rather than overwrote.
func Install(
	idx *knowledge.Index,
	rnd *render.Renderer,
	dest string,
	ids []string,
	routerFilter []string,
) (installed, skipped, backedUp []string, err error) {
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

// InstallWorkflows renders each workflow whose id is in wfIDs and writes it to
// <dest>/<id>/WORKFLOW.md plus a sibling .gitignore (see WriteDoc). Unknown
// ids are skipped and reported. backedUp carries a ready-to-print message for
// every foreign file WriteDoc backed up rather than overwrote.
func InstallWorkflows(
	idx *knowledge.Index,
	rnd *render.Renderer,
	dest string,
	wfIDs []string,
) (installed, skipped, backedUp []string, err error) {
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

// installDocs writes render(id)'s body to <dest>/<id>/<docName> for every id
// render reports known; ids render reports unknown are skipped and returned
// separately. render is expected to wrap its own render errors with the id's
// context. It is the shared body behind Install and InstallWorkflows, which
// differ only in the doc filename and how a body is rendered.
func installDocs(
	dest, docName string,
	ids []string,
	render func(id string) (body string, ok bool, err error),
) (installed, skipped, backedUp []string, err error) {
	for _, id := range ids {
		body, ok, renderErr := render(id)
		if renderErr != nil {
			return installed, skipped, backedUp, renderErr
		}
		if !ok {
			skipped = append(skipped, id)
			continue
		}
		docBackups, writeErr := WriteDoc(filepath.Join(dest, id), docName, body)
		if writeErr != nil {
			return installed, skipped, backedUp, fmt.Errorf("install %q: %w", id, writeErr)
		}
		backedUp = append(backedUp, docBackups...)
		installed = append(installed, id)
	}
	return installed, skipped, backedUp, nil
}
