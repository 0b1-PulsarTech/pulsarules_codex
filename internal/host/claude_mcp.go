package host

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/mcp"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

func wireClaudeMCP(ctx Context, skillsDir string, report *Report) error {
	wired, skipped, err := mcp.Write(ctx.Base, ctx.Languages)
	if err != nil {
		return fmt.Errorf("write mcp: %w", err)
	}
	for _, name := range skipped {
		if name == "gopls" {
			// The gopls warning in goplsSkill carries richer install guidance.
			continue
		}
		report.Warn("%s not on PATH; skipping its MCP server", name)
	}
	if _, err = goplsSkill(ctx, skillsDir, report); err != nil {
		return fmt.Errorf("generate gopls skill: %w", err)
	}
	if len(wired) > 0 {
		report.Note(
			"wired MCP servers (%s) into %s",
			strings.Join(wired, ", "), filepath.Join(ctx.Base, ".mcp.json"),
		)
	}
	return nil
}

func unwireClaudeMCP(repoDir string, report *Report) error {
	changed, err := mcp.Remove(repoDir)
	if err != nil {
		if errors.Is(err, jsondoc.ErrUnparseable) {
			report.Warn("%v", err)
			return nil
		}
		return fmt.Errorf("remove mcp: %w", err)
	}
	if changed {
		report.Note("unwired MCP servers from %s", filepath.Join(repoDir, ".mcp.json"))
	}
	return nil
}

func goplsInstructions() (string, error) {
	const goplsInstructionsTimeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), goplsInstructionsTimeout)
	defer cancel()
	instructions, err := mcp.GoplsInstructions(ctx)
	if err != nil {
		return "", fmt.Errorf("get gopls instructions: %w", err)
	}
	return instructions, nil
}

func generateGoplsSkill(templates fs.FS, skillsDir, instructions string, report *Report) error {
	backedUp, genErr := mcp.GenerateGoplsSkill(templates, skillsDir, instructions)
	// Backed-up foreign files must reach the report, or a hand-written skill is renamed silently.
	report.Merge(ownedfile.Ledger{BackedUp: backedUp}.Report())
	if genErr != nil {
		return fmt.Errorf("generate gopls skill: %w", genErr)
	}
	return nil
}

func goplsSkill(ctx Context, dest string, report *Report) (generated bool, err error) {
	switch {
	case !ctx.Languages.Has(language.Go):
		// No Go detected: neither the server nor its navigation skill applies.
		return false, nil
	case mcp.GoplsOnPath():
		instructions, instructionsErr := goplsInstructions()
		if instructionsErr != nil {
			return false, instructionsErr
		}
		if err = generateGoplsSkill(ctx.Templates, dest, instructions, report); err != nil {
			return false, err
		}
		return true, nil
	default:
		// A missing tool is a warning, never an install failure.
		report.Warn(noGoplsWarning)
		return false, nil
	}
}
