package host

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/agents"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ordered"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/docs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

const noGoplsWarning = "gopls not on PATH; skipping gopls MCP + gopls-navigation skill " +
	"(install: go install golang.org/x/tools/gopls@latest)"

func installSkills(ctx Context, dest string, report *Report) error {
	installed, skipped, backedUp, err := docs.Install(
		ctx.Index, ctx.Renderer, dest, ctx.IDs, ctx.RouterFilter,
	)
	// Install returns the backups it already made alongside the error; report them
	// even when a later id fails.
	for _, msg := range backedUp {
		report.warn("%s", msg)
	}
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

func workflowsForSkills(idx *knowledge.Index, skillIDs []string) []string {
	var ids []string
	for _, sid := range skillIDs {
		skill, ok := idx.Skill(sid)
		if !ok {
			continue
		}
		ids = append(ids, skill.ComposeWorkflows...)
	}
	return ordered.Dedupe(ids)
}

func installWorkflows(ctx Context, dest string, report *Report) error {
	wfIDs := workflowsForSkills(ctx.Index, ctx.IDs)
	if len(wfIDs) == 0 {
		return nil
	}
	installed, skipped, backedUp, err := docs.InstallWorkflows(
		ctx.Index,
		ctx.Renderer,
		dest,
		wfIDs,
	)
	for _, msg := range backedUp {
		report.warn("%s", msg)
	}
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

func removeSkills(dest string, report *Report) error {
	removed, restored, err := docs.Remove(dest, "SKILL.md")
	if err != nil {
		return fmt.Errorf("remove skills from %q: %w", dest, err)
	}
	for _, id := range removed {
		report.note("removed: %s", filepath.Join(dest, id))
	}
	for _, msg := range restored {
		report.note("%s", msg)
	}
	return nil
}

func removeWorkflows(dest string, report *Report) error {
	removed, restored, err := docs.Remove(dest, "WORKFLOW.md")
	if err != nil {
		return fmt.Errorf("remove workflows from %q: %w", dest, err)
	}
	for _, id := range removed {
		report.note("removed workflow: %s", filepath.Join(dest, id))
	}
	for _, msg := range restored {
		report.note("%s", msg)
	}
	return nil
}

func writeAgents(ctx Context, report *Report) error {
	path := filepath.Join(ctx.Base, "AGENTS.md")
	wrote, err := agents.Write(ctx.Templates, ctx.Base, ctx.Index, ctx.IDs)
	if err != nil {
		return fmt.Errorf("write agents: %w", err)
	}
	if wrote {
		report.note("wrote %s", path)
	} else {
		report.warn("kept existing user-authored %s (not overwritten)", path)
	}
	return nil
}

func removeAgents(base string, report *Report) error {
	path := filepath.Join(base, "AGENTS.md")
	removed, err := agents.RemoveAgents(base)
	if err != nil {
		return fmt.Errorf("remove agents: %w", err)
	}
	if removed {
		report.note("removed %s", path)
	}
	return nil
}
