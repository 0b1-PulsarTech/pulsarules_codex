package host

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/mcp"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/mcpserver"
)

func wireClaudeMCP(ctx Context, skillsDir string, report *Report) error {
	wired, skipped, err := writeMCPServers(ctx, report)
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
	if err = goplsSkill(ctx, skillsDir, report); err != nil {
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

func writeMCPServers(ctx Context, report *Report) (wired, skipped []string, err error) {
	if !ctx.Global {
		return mcp.Write(ctx.Base, ctx.Languages)
	}
	// Claude reads user-level servers from ~/.claude.json, which Claude itself rewrites;
	// a file of ours there would race it, so the user adds the server with Claude's own command.
	var available []mcpserver.Server
	available, skipped = mcpserver.Available(ctx.Languages.Sorted())
	for _, srv := range available {
		report.Note(
			"add the %s MCP server for every project with: claude mcp add --scope user %s -- %s",
			srv.Name, srv.Name, strings.Join(append([]string{srv.Command}, srv.Args...), " "),
		)
	}
	return nil, skipped, nil
}

func unwireClaudeMCP(repoDir string, report *Report) error {
	changed, err := mcp.Remove(repoDir)
	if err != nil {
		if warnUnparseable(report, err) {
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

func goplsSkill(ctx Context, dest string, report *Report) error {
	switch {
	case !ctx.Languages.Has(language.Go):
		// No Go detected: neither the server nor its navigation skill applies.
		return nil
	case mcp.GoplsOnPath():
		instructions, err := goplsInstructions()
		if err != nil {
			return err
		}
		return generateGoplsSkill(ctx.Templates, dest, instructions, report)
	default:
		// A missing tool is a warning, never an install failure.
		report.Warn(noGoplsWarning)
		return nil
	}
}
