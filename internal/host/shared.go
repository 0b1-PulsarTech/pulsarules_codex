package host

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/agents"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ordered"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

const noGoplsWarning = "gopls not on PATH; skipping gopls MCP + gopls-navigation skill " +
	"(install: go install golang.org/x/tools/gopls@latest)"

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

func writeAgents(ctx Context, report *Report) error {
	outcome, err := agents.Write(ctx.Templates, ctx.Base, ctx.Index, ctx.IDs)
	if err != nil {
		return fmt.Errorf("write agents: %w", err)
	}
	report.Merge(outcome.ReportWrite(filepath.Join(ctx.Base, agents.FileName), "wrote %s"))
	return nil
}

func removeAgents(base string, report *Report) error {
	path := filepath.Join(base, agents.FileName)
	outcome, err := ownedfile.RemoveOwned(path)
	if err != nil {
		return fmt.Errorf("remove agents: %w", err)
	}
	if outcome == ownedfile.Removed {
		report.Note("removed %s", path)
	}
	return nil
}

func warnUnparseable(report *Report, err error) bool {
	if !errors.Is(err, jsondoc.ErrUnparseable) {
		return false
	}
	report.Warn("%v", err)
	return true
}
