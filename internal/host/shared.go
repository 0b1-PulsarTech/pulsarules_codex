package host

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/agents"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/claude"
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
	return writeAgentsIn(ctx, ctx.Base, report)
}

func writeAgentsIn(ctx Context, dir string, report *Report) error {
	outcome, ledger, err := agents.Write(ctx.Templates, dir, ctx.Index, ctx.IDs)
	report.Merge(ledger.Report())
	if err != nil {
		return fmt.Errorf("write agents: %w", err)
	}
	report.Merge(outcome.ReportWrite(filepath.Join(dir, agents.FileName), "wrote %s"))
	return nil
}

func removeAgents(base string, report *Report) error {
	path := filepath.Join(base, agents.FileName)
	outcome, ledger, err := ownedfile.RemoveOwned(path)
	report.Merge(ledger.Report())
	if err != nil {
		return fmt.Errorf("remove agents: %w", err)
	}
	if outcome == ownedfile.Removed {
		report.Note("removed %s", path)
	}
	return nil
}

func warnBlock(report *Report, heading, block string) {
	// Each line is its own warning because the report prints one escaped line at a time.
	report.Warn("%s", heading)
	for line := range strings.SplitSeq(strings.TrimRight(block, "\n"), "\n") {
		report.Warn("    %s", line)
	}
}

func warnUnparseable(report *Report, err error) bool {
	if !errors.Is(err, jsondoc.ErrUnparseable) {
		return false
	}
	report.Warn("%v", err)
	return true
}

func sameDir(first, second string) bool {
	// File identity, so a symlinked path to the same directory is the same directory.
	if first == "" || second == "" {
		return false
	}
	firstInfo, firstErr := os.Stat(first)
	secondInfo, secondErr := os.Stat(second)
	return firstErr == nil && secondErr == nil && os.SameFile(firstInfo, secondInfo)
}

func warnShadowedSkills(ctx Context, claudeDir string, report *Report) {
	// Advisory: an unreadable or odd entry counts as not ours and never fails the install.
	if ctx.Global || ctx.UserClaudeDir == "" || sameDir(ctx.UserClaudeDir, claudeDir) {
		return
	}
	userSkills := filepath.Join(ctx.UserClaudeDir, claude.SkillsSubdir)
	var installed, shadowed []string
	unchecked := 0
	for _, id := range ordered.Dedupe(append(slices.Clone(ctx.IDs), "gopls-navigation")) {
		if _, err := os.Stat(
			filepath.Join(claudeDir, claude.SkillsSubdir, id, "SKILL.md"),
		); err != nil {
			continue
		}
		installed = append(installed, id)
		_, ours, err := ownedfile.Check(filepath.Join(userSkills, id, "SKILL.md"))
		switch {
		case err != nil:
			unchecked++
		case ours:
			shadowed = append(shadowed, id)
		}
	}
	if unchecked > 0 {
		report.Warn("could not check %d skills in %s for shadowing; skipped", unchecked, userSkills)
	}
	if len(shadowed) > 0 {
		report.Warn("%s", shadowWarning(len(installed), shadowed, userSkills))
	}
}

func shadowWarning(installed int, shadowed []string, userSkills string) string {
	// Claude Code prefers a personal skill over a project one with the same name.
	const advice = "take precedence over this project's copies " +
		"(remove them with uninstall --global, or accept the global versions)"
	if len(shadowed) == installed {
		return fmt.Sprintf("all %d installed skills are also installed in %s and %s",
			installed, userSkills, advice)
	}
	const maxListed = 10
	listed := shadowed[:min(len(shadowed), maxListed)]
	text := strings.Join(listed, ", ")
	if more := len(shadowed) - len(listed); more > 0 {
		text += fmt.Sprintf(" +%d more", more)
	}
	return fmt.Sprintf("%d of %d installed skills also installed in %s %s: %s",
		len(shadowed), installed, userSkills, advice, text)
}
