package host

import (
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/docs"
)

const noGoplsWarning = "gopls not on PATH; skipping gopls MCP + gopls-navigation skill " +
	"(install: go install golang.org/x/tools/gopls@latest)"

func installSkills(ctx Context, dest string, report *Report) error {
	installed, skipped, err := docs.Install(
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
	return nil
}
