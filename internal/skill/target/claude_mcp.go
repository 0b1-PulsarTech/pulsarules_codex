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

func wireClaudeMCP(templates fs.FS, repoDir, skillsDir string, report *Report) error {
	if !mcpwire.GoplsOnPath() {
		report.warn(noGoplsWarning)
		return nil
	}
	if err := mcpwire.WriteMCP(templates, repoDir); err != nil {
		return fmt.Errorf("write mcp: %w", err)
	}
	if err := generateGoplsSkill(templates, skillsDir, report); err != nil {
		return fmt.Errorf("generate gopls skill: %w", err)
	}
	report.note(
		"wired gopls MCP into %s; generated gopls-navigation skill",
		filepath.Join(repoDir, ".mcp.json"),
	)
	return nil
}

func unwireClaudeMCP(repoDir string, report *Report) error {
	changed, err := mcpwire.RemoveMCP(repoDir)
	if err != nil {
		if errors.Is(err, fsx.ErrUnparseableJSON) {
			report.warn("%v", err)
			return nil
		}
		return fmt.Errorf("remove mcp: %w", err)
	}
	if changed {
		report.note("unwired gopls from %s", filepath.Join(repoDir, ".mcp.json"))
	}
	return nil
}

func generateGoplsSkill(templates fs.FS, skillsDir string, report *Report) error {
	const goplsInstructionsTimeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), goplsInstructionsTimeout)
	defer cancel()
	instructions, err := mcpwire.GoplsInstructions(ctx)
	if err != nil {
		return fmt.Errorf("get gopls instructions: %w", err)
	}
	backedUp, genErr := mcpwire.GenerateGoplsSkill(templates, skillsDir, instructions)
	// Backed-up foreign files must reach the report, or a hand-written skill is renamed silently.
	for _, msg := range backedUp {
		report.warn("%s", msg)
	}
	if genErr != nil {
		return fmt.Errorf("generate gopls skill: %w", genErr)
	}
	return nil
}
