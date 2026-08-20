package target

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/agentswire"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/slicesx"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// noGoplsWarning is reported (not fatal) when gopls is missing, so a missing
// tool never fails an install.
const noGoplsWarning = "gopls not on PATH; skipping gopls MCP + gopls-navigation skill " +
	"(install: go install golang.org/x/tools/gopls@latest)"

// installSkills renders the selected skills to dest, recording installed and
// skipped (unknown) ids in the report, plus a warning for every foreign file
// output.Install backed up rather than overwrote (see output.WriteDoc). It is
// shared by every layout Strategy.
func installSkills(ctx Context, dest string, report *Report) error {
	installed, skipped, backedUp, err := output.Install(
		ctx.Index, ctx.Renderer, dest, ctx.IDs, ctx.RouterFilter,
	)
	// why: output.Install returns the backups it ALREADY made alongside the error, so an id that
	// fails midway through the batch must not swallow the renames the earlier ids performed - the
	// user would see only "install failed" while a file of theirs sat renamed away.
	for _, msg := range backedUp {
		report.Warn("%s", msg)
	}
	if err != nil {
		return fmt.Errorf("render skills to %q: %w", dest, err)
	}
	for _, id := range installed {
		report.Note("installed: %s", filepath.Join(dest, id))
	}
	for _, id := range skipped {
		report.Warn("skipped (unknown skill): %s", id)
	}
	return nil
}

// workflowsForSkills returns the union of all compose_workflows from the given
// skill ids, in insertion order with duplicates removed, so each workflow is
// installed exactly once regardless of how many skills reference it.
func workflowsForSkills(idx *knowledge.Index, skillIDs []string) []string {
	var ids []string
	for _, sid := range skillIDs {
		skill, ok := idx.Skill(sid)
		if !ok {
			continue
		}
		ids = append(ids, skill.ComposeWorkflows...)
	}
	return slicesx.Dedupe(ids)
}

// installWorkflows renders the workflows composed by the installed skills to
// dest, recording installed and skipped ids in the report, plus a warning
// for every foreign file output.InstallWorkflows backed up rather than
// overwrote (see output.WriteDoc).
func installWorkflows(ctx Context, dest string, report *Report) error {
	wfIDs := workflowsForSkills(ctx.Index, ctx.IDs)
	if len(wfIDs) == 0 {
		return nil
	}
	installed, skipped, backedUp, err := output.InstallWorkflows(
		ctx.Index,
		ctx.Renderer,
		dest,
		wfIDs,
	)
	// why: same as installSkills - a backup already made is reported even when a later id fails.
	for _, msg := range backedUp {
		report.Warn("%s", msg)
	}
	if err != nil {
		return fmt.Errorf("render workflows to %q: %w", dest, err)
	}
	for _, id := range installed {
		report.Note("installed workflow: %s", filepath.Join(dest, id))
	}
	for _, id := range skipped {
		report.Warn("skipped workflow (unknown): %s", id)
	}
	return nil
}

// removeSkills deletes every skill directory output.RemoveDocs recognizes
// as ours under dest, recording each removed id and restored backup in the
// report. Shared by every layout Strategy's Uninstall, mirroring
// installSkills; it also cleans up the generated gopls-navigation skill,
// since GenerateGoplsSkill writes it through the same fingerprint.
func removeSkills(dest string, report *Report) error {
	removed, restored, orphaned, err := output.RemoveDocs(dest, "SKILL.md")
	if err != nil {
		return fmt.Errorf("remove skills from %q: %w", dest, err)
	}
	for _, id := range removed {
		report.Note("removed: %s", filepath.Join(dest, id))
	}
	for _, msg := range restored {
		report.Note("%s", msg)
	}
	for _, msg := range orphaned {
		report.Warn("%s", msg)
	}
	return nil
}

// removeWorkflows deletes every workflow directory output.RemoveDocs
// recognizes as ours under dest, recording each removed id in the report,
// plus a note for every backup output.RemoveDocs restored (see
// marker.Backup). Only the claude layout installs workflows.
func removeWorkflows(dest string, report *Report) error {
	removed, restored, orphaned, err := output.RemoveDocs(dest, "WORKFLOW.md")
	if err != nil {
		return fmt.Errorf("remove workflows from %q: %w", dest, err)
	}
	for _, id := range removed {
		report.Note("removed workflow: %s", filepath.Join(dest, id))
	}
	for _, msg := range restored {
		report.Note("%s", msg)
	}
	for _, msg := range orphaned {
		report.Warn("%s", msg)
	}
	return nil
}

// writeAgents renders the root AGENTS.md through agentswire.WriteAgents,
// shared by opencodeTarget and agentsTarget so the file can never diverge
// between them. It notes the write, or warns instead when a user-authored
// AGENTS.md without the ownership marker already occupies the path, so
// Install never silently clobbers a file a user very plausibly owns already.
func writeAgents(ctx Context, report *Report) error {
	path := filepath.Join(ctx.Base, "AGENTS.md")
	wrote, err := agentswire.WriteAgents(ctx.Templates, ctx.Base, ctx.Index, ctx.IDs)
	if err != nil {
		return fmt.Errorf("write agents: %w", err)
	}
	if wrote {
		report.Note("wrote %s", path)
	} else {
		report.Warn("kept existing user-authored %s (not overwritten)", path)
	}
	return nil
}

// removeAgents deletes <base>/AGENTS.md through agentswire.RemoveAgents,
// undoing writeAgents. It is shared by opencodeTarget and agentsTarget so
// both layouts reverse the same file the same way.
func removeAgents(base string, report *Report) error {
	path := filepath.Join(base, "AGENTS.md")
	removed, err := agentswire.RemoveAgents(base)
	if err != nil {
		return fmt.Errorf("remove agents: %w", err)
	}
	if removed {
		report.Note("removed %s", path)
	}
	return nil
}
