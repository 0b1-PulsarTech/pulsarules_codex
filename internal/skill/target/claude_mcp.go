package target

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/mcpwire"
)

// goplsInstructionsTimeout bounds the `gopls mcp -instructions` call.
const goplsInstructionsTimeout = 30 * time.Second

// wireClaudeMCP merges the gopls server into the project's .mcp.json and
// regenerates the gopls-navigation skill. It is a no-op (with a warning) when
// gopls is not on PATH, so a missing tool never fails an install.
func wireClaudeMCP(templates fs.FS, repoDir, skillsDir string, report *Report) error {
	if !mcpwire.GoplsOnPath() {
		report.Warn(noGoplsWarning)
		return nil
	}
	if err := mcpwire.WriteMCP(templates, repoDir); err != nil {
		return fmt.Errorf("write mcp: %w", err)
	}
	if err := generateGoplsSkill(templates, skillsDir, report); err != nil {
		return fmt.Errorf("generate gopls skill: %w", err)
	}
	report.Note(
		"wired gopls MCP into %s; generated gopls-navigation skill",
		filepath.Join(repoDir, ".mcp.json"),
	)
	return nil
}

// unwireClaudeMCP removes the gopls server entry from .mcp.json, warning
// instead of failing when the file exists but does not parse. It notes the
// removal only when mcpwire.RemoveMCP actually changed something, so the
// report never claims to have unwired gopls from a file that was absent, or
// present but carrying no gopls entry.
func unwireClaudeMCP(repoDir string, report *Report) error {
	changed, err := mcpwire.RemoveMCP(repoDir)
	if err != nil {
		if errors.Is(err, fsx.ErrUnparseableJSON) {
			report.Warn("%v", err)
			return nil
		}
		return fmt.Errorf("remove mcp: %w", err)
	}
	if changed {
		report.Note("unwired gopls from %s", filepath.Join(repoDir, ".mcp.json"))
	}
	return nil
}

// generateGoplsSkill runs `gopls mcp -instructions` and writes the
// gopls-navigation skill into skillsDir, warning for every foreign file it
// backs up rather than overwrites (see output.WriteDoc).
// why: sharing opencodeTarget's signature discarded that list; both already
// hold a *Report, so sharing it just silently renamed a hand-written skill.
func generateGoplsSkill(templates fs.FS, skillsDir string, report *Report) error {
	ctx, cancel := context.WithTimeout(context.Background(), goplsInstructionsTimeout)
	defer cancel()
	instructions, err := mcpwire.GoplsInstructions(ctx)
	if err != nil {
		return fmt.Errorf("get gopls instructions: %w", err)
	}
	backedUp, genErr := mcpwire.GenerateGoplsSkill(templates, skillsDir, instructions)
	for _, msg := range backedUp {
		report.Warn("%s", msg)
	}
	if genErr != nil {
		return fmt.Errorf("generate gopls skill: %w", genErr)
	}
	return nil
}
