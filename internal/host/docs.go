package host

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/docs"
)

type docKind struct {
	noun      string
	docName   string
	installed string
	skipped   string
	removed   string
}

func skillKind() docKind {
	return docKind{
		noun:      "skills",
		docName:   "SKILL.md",
		installed: "installed: %s",
		skipped:   "skipped (unknown skill): %s",
		removed:   "removed: %s",
	}
}

func workflowKind() docKind {
	return docKind{
		noun:      "workflows",
		docName:   "WORKFLOW.md",
		installed: "installed workflow: %s",
		skipped:   "skipped workflow (unknown): %s",
		removed:   "removed workflow: %s",
	}
}

func installSkills(ctx Context, dest string, report *Report) error {
	result, err := docs.Install(
		ctx.Index, ctx.Renderer, dest,
		docs.InstallRequest{IDs: ctx.IDs, RouterFilter: ctx.RouterFilter},
	)
	return reportInstalled(skillKind(), dest, result, err, report)
}

func installWorkflows(ctx Context, dest string, report *Report) error {
	wfIDs := workflowsForSkills(ctx.Index, ctx.IDs)
	if len(wfIDs) == 0 {
		return nil
	}
	result, err := docs.InstallWorkflows(ctx.Index, ctx.Renderer, dest, wfIDs)
	return reportInstalled(workflowKind(), dest, result, err, report)
}

func reportInstalled(
	kind docKind, dest string, result docs.InstallResult, err error, report *Report,
) error {
	// Backups made before a later id failed still reach the report.
	report.Merge(ownedfile.Ledger{BackedUp: result.BackedUp}.Report())
	if err != nil {
		return fmt.Errorf("render %s to %q: %w", kind.noun, dest, err)
	}
	for _, id := range result.Installed {
		report.Note(kind.installed, filepath.Join(dest, id))
	}
	for _, id := range result.Skipped {
		report.Warn(kind.skipped, id)
	}
	return nil
}

func removeSkills(dest string, report *Report) error {
	return removeKind(skillKind(), dest, report)
}

func removeWorkflows(dest string, report *Report) error {
	return removeKind(workflowKind(), dest, report)
}

func removeKind(kind docKind, dest string, report *Report) error {
	result, err := docs.Remove(dest, kind.docName)
	if err != nil {
		return fmt.Errorf("remove %s from %q: %w", kind.noun, dest, err)
	}
	for _, id := range result.Removed {
		report.Note(kind.removed, filepath.Join(dest, id))
	}
	report.Merge(ownedfile.Ledger{Restored: result.Restored, Orphaned: result.Orphaned}.Report())
	return nil
}
