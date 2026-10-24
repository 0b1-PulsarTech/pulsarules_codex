package install

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/claude"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/codex"
)

func printHooksBlock(cfg installConfig, targets []string) error {
	wantsCodex := slices.Contains(targets, "codex")
	// Any target set without codex keeps printing the Claude block, as it always did.
	if !wantsCodex || slices.Contains(targets, "claude") {
		block, err := claude.RenderHooksBlock(claude.Location{GlobalDir: cfg.claudeDir})
		if err != nil {
			return fmt.Errorf("render hooks block: %w", err)
		}
		_, _ = fmt.Print(string(block))
	}
	if !wantsCodex {
		return nil
	}
	loc := codex.Location{}
	if cfg.global {
		loc = codex.Location{
			GlobalDir:  cfg.codexDir,
			UserSkills: filepath.Join(cfg.absBase, codex.SkillsDir),
		}
	}
	block, err := codex.RenderHooksBlock(loc)
	if err != nil {
		return fmt.Errorf("render codex hooks block: %w", err)
	}
	_, _ = fmt.Print(string(block))
	return nil
}
