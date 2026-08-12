package host

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/mcp"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

func wireClaudeMCP(templates fs.FS, repoDir, skillsDir string, report *Report) error {
	if !mcp.GoplsOnPath() {
		report.warn(noGoplsWarning)
		return nil
	}
	if err := mcp.Write(templates, repoDir); err != nil {
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
	mcpPath := filepath.Join(repoDir, ".mcp.json")
	if _, statErr := os.Stat(mcpPath); errors.Is(statErr, fs.ErrNotExist) {
		return nil
	}
	if err := mcp.Remove(repoDir); err != nil {
		if errors.Is(err, jsondoc.ErrUnparseableJSON) {
			report.warn("%v", err)
			return nil
		}
		return fmt.Errorf("remove mcp: %w", err)
	}
	report.note("unwired gopls from %s", mcpPath)
	return nil
}

func generateGoplsSkill(templates fs.FS, skillsDir string, report *Report) error {
	const goplsInstructionsTimeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), goplsInstructionsTimeout)
	defer cancel()
	instructions, err := mcp.GoplsInstructions(ctx)
	if err != nil {
		return fmt.Errorf("get gopls instructions: %w", err)
	}
	backedUp, genErr := mcp.GenerateGoplsSkill(templates, skillsDir, instructions)
	if genErr != nil {
		return fmt.Errorf("generate gopls skill: %w", genErr)
	}
	for _, msg := range backedUp {
		report.warn("%s", msg)
	}
	return nil
}
