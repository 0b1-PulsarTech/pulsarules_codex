package host

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/agents"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/docs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

const noGoplsWarning = "gopls not on PATH; skipping gopls MCP + gopls-navigation skill " +
	"(install: go install golang.org/x/tools/gopls@latest)"

func installSkills(ctx Context, dest string, report *Report) error {
	installed, skipped, backedUp, err := docs.Install(
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
	for _, msg := range backedUp {
		report.warn("%s", msg)
	}
	return nil
}

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
	if err != nil {
		return fmt.Errorf("render workflows to %q: %w", dest, err)
	}
	for _, id := range installed {
		report.note("installed workflow: %s", filepath.Join(dest, id))
	}
	for _, id := range skipped {
		report.warn("skipped workflow (unknown): %s", id)
	}
	for _, msg := range backedUp {
		report.warn("%s", msg)
	}
	return nil
}

func writeAgents(ctx Context, report *Report) error {
	path := filepath.Join(ctx.Base, "AGENTS.md")
	wrote, err := agents.Write(ctx.Templates, ctx.Base, ctx.Index)
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
