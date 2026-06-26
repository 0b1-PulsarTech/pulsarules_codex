package target

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// noGoplsWarning is reported (not fatal) when gopls is missing, so a missing
// tool never fails an install.
const noGoplsWarning = "gopls not on PATH; skipping gopls MCP + gopls-navigation skill " +
	"(install: go install golang.org/x/tools/gopls@latest)"

// installSkills renders the selected skills to dest, recording installed and
// skipped (unknown) ids in the report. It is shared by every layout Strategy.
func installSkills(ctx Context, dest string, report *Report) error {
	installed, skipped, err := output.Install(
		ctx.Index, ctx.Renderer, dest, ctx.IDs, ctx.RouterFilter,
	)
	if err != nil {
		return fmt.Errorf("render skills to %q: %w", dest, err)
	}
	for _, id := range installed {
		report.note("installed: %s", filepath.Join(dest, id))
	}
	for _, id := range skipped {
		report.warn("skipped (unknown skill): %s", id)
	}
	return nil
}

// workflowsForSkills returns the union of all compose_workflows from the given
// skill ids, in insertion order with duplicates removed, so each workflow is
// installed exactly once regardless of how many skills reference it.
func workflowsForSkills(idx *knowledge.Index, skillIDs []string) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, sid := range skillIDs {
		skill, ok := idx.Skill(sid)
		if !ok {
			continue
		}
		for _, wid := range skill.ComposeWorkflows {
			if !seen[wid] {
				seen[wid] = true
				ids = append(ids, wid)
			}
		}
	}
	return ids
}

// installWorkflows renders the workflows composed by the installed skills to
// dest, recording installed and skipped ids in the report.
func installWorkflows(ctx Context, dest string, report *Report) error {
	wfIDs := workflowsForSkills(ctx.Index, ctx.IDs)
	if len(wfIDs) == 0 {
		return nil
	}
	installed, skipped, err := output.InstallWorkflows(ctx.Index, ctx.Renderer, dest, wfIDs)
	if err != nil {
		return fmt.Errorf("render workflows to %q: %w", dest, err)
	}
	for _, id := range installed {
		report.note("installed workflow: %s", filepath.Join(dest, id))
	}
	for _, id := range skipped {
		report.warn("skipped workflow (unknown): %s", id)
	}
	return nil
}
