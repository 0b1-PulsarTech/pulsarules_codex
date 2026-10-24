package host

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/codex"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/mcpserver"
)

type codexTarget struct{}

var _ Target = codexTarget{}

func (codexTarget) Name() string { return "codex" }

func (codexTarget) Present(base string) bool {
	_, err := os.Stat(filepath.Join(base, codex.RootDir))
	return err == nil
}

type codexScope struct {
	Base, CodexDir string
	Global         bool
}

func codexPaths(scope codexScope) (dir, skills, agentsDir string) {
	base := scope.Base
	dir = cmp.Or(scope.CodexDir, filepath.Join(base, codex.RootDir))
	// Codex reads user skills from $HOME/.agents/skills, never from under CODEX_HOME, and a
	// user-level AGENTS.md from CODEX_HOME.
	skills = filepath.Join(base, codex.SkillsDir)
	agentsDir = base
	if scope.Global {
		agentsDir = dir
	}
	return dir, skills, agentsDir
}

func (codexTarget) Install(ctx Context) (Report, error) {
	var report Report
	codexDir, skills, agentsDir := codexPaths(codexScope{ctx.Base, ctx.CodexDir, ctx.Global})
	if err := installSkills(ctx, skills, &report); err != nil {
		return report, err
	}
	if !ctx.NoMCP {
		if err := wireCodexMCP(ctx, codexDir, skills, &report); err != nil {
			return report, err
		}
	}
	if err := writeAgentsIn(ctx, agentsDir, &report); err != nil {
		return report, err
	}
	if err := retireLegacyCodex(codexDir, &report); err != nil {
		return report, err
	}
	if ctx.NoHooks {
		return report, nil
	}
	loc := codex.Location{}
	if ctx.Global {
		loc = codex.Location{GlobalDir: codexDir, UserSkills: skills}
	}
	sub, err := install.Codex(codexDir, ctx.Templates, loc)
	report.Merge(sub)
	if err != nil {
		return report, fmt.Errorf("install hooks: %w", err)
	}
	report.Note("trust the project in Codex and run /hooks to approve the hooks")
	return report, nil
}

func wireCodexMCP(ctx Context, codexDir, skills string, report *Report) error {
	servers, skipped := mcpserver.Available(ctx.Languages.Sorted())
	for _, name := range skipped {
		if name != "gopls" {
			// The gopls warning in goplsSkill carries richer install guidance.
			report.Warn("%s not on PATH; skipping its MCP server", name)
		}
	}
	if err := goplsSkill(ctx, skills, report); err != nil {
		return fmt.Errorf("generate gopls skill: %w", err)
	}
	wired, clashed, err := codex.WriteMCP(codexDir, servers)
	config := filepath.Join(codexDir, codex.ConfigFile)
	switch {
	case errors.Is(err, codex.ErrUnterminatedBlock):
		report.Warn("%v; left %s alone", err, config)
		return nil
	case errors.Is(err, codex.ErrSymlink):
		warnBlock(report, fmt.Sprintf("%v; left it alone. Add this to the file it points to:", err),
			codex.RenderMCP(servers))
		return nil
	case err != nil:
		return fmt.Errorf("write mcp: %w", err)
	}
	for _, name := range clashed {
		found, findErr := codex.ClashFound(codexDir, name)
		if findErr != nil {
			return fmt.Errorf("describe mcp clash: %w", findErr)
		}
		report.Warn("%s already defines %s; left that server alone", config, found)
	}
	for _, name := range wired {
		report.Note("wired MCP server %s into %s", name, config)
	}
	return nil
}

func retireLegacyCodex(codexDir string, report *Report) error {
	// Skills once lived under .codex/skills; only our own copies are removed.
	if err := removeSkills(filepath.Join(codexDir, codex.LegacySkillsSubdir), report); err != nil {
		return err
	}
	wired, err := codex.SettingsWiresHook(codexDir)
	if err != nil {
		return fmt.Errorf("check codex settings: %w", err)
	}
	if wired {
		report.Warn(
			"%s wires our hook, but Codex does not read settings.json; remove that wiring",
			filepath.Join(codexDir, "settings.json"),
		)
	}
	return nil
}

func (codexTarget) Uninstall(ctx UninstallContext) (Report, error) {
	var report Report
	codexDir, skills, agentsDir := codexPaths(codexScope{ctx.Base, ctx.CodexDir, ctx.Global})
	if !ctx.KeepSkills {
		if err := removeSkills(skills, &report); err != nil {
			return report, err
		}
		if err := removeAgents(agentsDir, &report); err != nil {
			return report, err
		}
	}
	if err := unwireCodexMCP(codexDir, &report); err != nil {
		return report, err
	}
	if err := retireLegacyCodex(codexDir, &report); err != nil {
		return report, err
	}
	sub, err := install.UninstallCodex(codexDir)
	report.Merge(sub)
	if err != nil {
		if warnUnparseable(&report, err) {
			return report, nil
		}
		return report, fmt.Errorf("uninstall hooks: %w", err)
	}
	for _, dir := range []string{codexDir, filepath.Join(ctx.Base, ".agents")} {
		if err = disk.RemoveEmptyDir(dir); err != nil {
			return report, fmt.Errorf("remove empty codex dir: %w", err)
		}
	}
	return report, nil
}

func unwireCodexMCP(codexDir string, report *Report) error {
	changed, err := codex.RemoveMCP(codexDir)
	config := filepath.Join(codexDir, codex.ConfigFile)
	switch {
	case errors.Is(err, codex.ErrUnterminatedBlock):
		report.Warn("%v; left %s alone", err, config)
	case errors.Is(err, codex.ErrSymlink):
		report.Warn("%v; left it alone. Remove our managed block from the file it points to", err)
	case err != nil:
		return fmt.Errorf("remove mcp: %w", err)
	case changed:
		report.Note("unwired MCP servers from %s", config)
	}
	return nil
}
