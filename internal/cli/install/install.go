package install

import (
	"fmt"
	"io/fs"
	"os"

	"golang.org/x/term"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/claude"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func (cmd *Command) Run(
	index *knowledge.Index,
	templates fs.FS,
	renderer *render.Renderer,
	targets *host.Registry,
) error {
	shared := host.Context{Templates: templates, Index: index, Renderer: renderer}
	if err := cmd.install(shared, targets); err != nil {
		return fmt.Errorf("install: %w", err)
	}
	return nil
}

func (cmd *Command) ProjectDir() string { return cmd.project }

func (cmd *Command) install(shared host.Context, registry *host.Registry) error {
	cfg, err := resolveInstallConfig(cmd)
	if err != nil {
		return err
	}

	if cmd.printHooks {
		return printHooksBlock(cfg)
	}

	targets := cmd.targets()
	// Refuse before any prompt: a question about an install that cannot happen is noise.
	if err = registry.Check(targets); err != nil {
		return err
	}
	if cfg.global {
		if err = registry.CheckGlobal(targets); err != nil {
			return err
		}
	}

	console := newPrompter(os.Stdin, os.Stdout)
	console.isTerminal = term.IsTerminal(int(os.Stdin.Fd()))
	ctx, err := newTargetContext(cmd, cfg, shared, console)
	if err != nil {
		return err
	}
	for _, name := range targets {
		report, installErr := registry.Install(name, ctx)
		report.Print(os.Stdout, os.Stderr)
		if installErr != nil {
			// The target registry's Install already wraps this as
			// `install target %q: %w`; re-wrapping here would double it.
			return installErr
		}
	}

	return installPostTargets(cmd, cfg)
}

func printHooksBlock(cfg installConfig) error {
	block, err := claude.RenderHooksBlock(claude.Location{GlobalDir: cfg.claudeDir})
	if err != nil {
		return fmt.Errorf("render hooks block: %w", err)
	}
	_, _ = fmt.Print(string(block))
	return nil
}

func newTargetContext(
	cmd *Command,
	cfg installConfig,
	shared host.Context,
	console *prompter,
) (host.Context, error) {
	if err := applySelectedLayout(cmd, shared.Index, console); err != nil {
		return host.Context{}, fmt.Errorf("apply layout: %w", err)
	}
	languages, err := resolveLanguages(cfg, cmd.languages)
	if err != nil {
		return host.Context{}, err
	}

	choice, err := resolveSelection(cmd, shared.Index, console)
	if err != nil {
		return host.Context{}, err
	}
	choice.Languages = languages
	ids, pulled := choice.Resolve(shared.Index)
	printDependencyPulls(os.Stdout, pulled)
	var routerFilter []string
	// Filter the rendered router to what's actually installed, unless it is
	// everything anyway (--all) or router-only (nothing to filter against).
	if !choice.All && !choice.RouterOnly {
		routerFilter = ids
	}

	shared.Base = cfg.absBase
	shared.Global = cfg.global
	shared.ClaudeDir = cfg.claudeDir
	shared.UserClaudeDir = cfg.userClaudeDir
	shared.IDs = ids
	shared.RouterFilter = routerFilter
	shared.NoMCP = cmd.noMCP
	shared.NoHooks = cmd.noHooks
	shared.SettingsFile = cfg.settingsFile
	shared.Languages = languages
	return shared, nil
}

type installConfig struct {
	absBase       string
	claudeDir     string
	userClaudeDir string
	global        bool
	settingsFile  string
	gitHooks      []string
}

func resolveInstallConfig(cmd *Command) (installConfig, error) {
	scope := flags.Scope{Global: cmd.global, Project: cmd.project}.WithEnv(cmd.env)
	absBase, err := flags.BaseDir(scope)
	if err != nil {
		return installConfig{}, fmt.Errorf("base dir: %w", err)
	}
	userClaudeDir, err := flags.UserClaudeDir(scope)
	if err != nil {
		return installConfig{}, fmt.Errorf("claude dir: %w", err)
	}
	var claudeDir string
	if cmd.global {
		claudeDir = userClaudeDir
	}
	hooksScope, err := flags.ParseHooksScope(cmd.hooksScope)
	if err != nil {
		return installConfig{}, fmt.Errorf("settings file: %w", err)
	}
	gitHooks, err := resolveGitHooks(cmd)
	if err != nil {
		return installConfig{}, err
	}
	return installConfig{
		absBase:       absBase,
		claudeDir:     claudeDir,
		userClaudeDir: userClaudeDir,
		global:        cmd.global,
		settingsFile:  hooksScope.File(),
		gitHooks:      gitHooks,
	}, nil
}
