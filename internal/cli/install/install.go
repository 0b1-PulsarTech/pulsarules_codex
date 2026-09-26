package install

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/langdetect"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/hookwire"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/target"
)

func Run(inj remy.Injector, opts *cliopts.Options) error {
	cfg, err := resolveInstallConfig(opts)
	if err != nil {
		return err
	}
	deps, err := resolveInstallCollaborators(inj)
	if err != nil {
		return err
	}

	if opts.PrintHooks {
		return printHooksBlock(deps.templates)
	}

	if err = applySelectedLayout(opts, deps.idx); err != nil {
		return fmt.Errorf("apply layout: %w", err)
	}

	ctx, err := newTargetContext(opts, cfg, deps)
	if err != nil {
		return err
	}
	targets := opts.Targets()
	if err = checkTargets(deps.targets, targets); err != nil {
		return err
	}
	for _, name := range targets {
		report, installErr := deps.targets.Install(name, ctx)
		printReport(report)
		if installErr != nil {
			// The target registry's Install already wraps this as
			// `install target %q: %w`; re-wrapping here would double it.
			return installErr
		}
	}

	return installPostTargets(opts, cfg.absBase, deps.templates, deps.hooks, cfg.gitHooks)
}

func printHooksBlock(templates fs.FS) error {
	block, err := hookwire.RenderHooksBlock(templates)
	if err != nil {
		return fmt.Errorf("render hooks block: %w", err)
	}
	_, _ = fmt.Print(string(block))
	return nil
}

func checkTargets(reg *target.Registry, targets []string) error {
	for _, name := range targets {
		if !reg.Has(name) {
			return fmt.Errorf(
				"invalid --target %q (want %s)", name, strings.Join(reg.Names(), "|"),
			)
		}
	}
	return nil
}

func newTargetContext(
	opts *cliopts.Options,
	cfg installConfig,
	deps installCollaborators,
) (target.Context, error) {
	languages, err := resolveLanguages(cfg.absBase, opts.Languages)
	if err != nil {
		return target.Context{}, err
	}

	selection, err := resolveSelection(opts, deps.idx)
	if err != nil {
		return target.Context{}, err
	}
	selection.Languages = languages
	ids, pulled := selection.Resolve(deps.idx)
	printDependencyPulls(pulled)
	var routerFilter []string
	// Filter the rendered router to what's actually installed, unless it is
	// everything anyway (--all) or router-only (nothing to filter against).
	if !selection.All && !selection.RouterOnly {
		routerFilter = ids
	}

	return target.Context{
		Templates:      deps.templates,
		Index:          deps.idx,
		Renderer:       deps.rnd,
		HookInstallers: deps.hooks,
		Base:           cfg.absBase,
		IDs:            ids,
		RouterFilter:   routerFilter,
		NoMCP:          opts.NoMCP,
		NoHooks:        opts.NoHooks,
		SettingsFile:   cfg.settingsFile,
	}, nil
}

func resolveLanguages(absBase, override string) (langdetect.Set, error) {
	languages, err := langdetect.Resolve(absBase, override)
	if err != nil {
		return langdetect.Set{}, fmt.Errorf("resolve project languages: %w", err)
	}
	if languages.Empty() {
		// An empty set installs every skill, not none.
		_, _ = fmt.Fprintf(os.Stderr,
			"warning: no project language detected in %q; installing every skill "+
				"regardless of language. Pass --languages to say which apply.\n", absBase)
	}
	return languages, nil
}

type installConfig struct {
	absBase      string
	settingsFile string
	gitHooks     []string
}

func resolveInstallConfig(opts *cliopts.Options) (installConfig, error) {
	projectDir, err := opts.BaseDir()
	if err != nil {
		return installConfig{}, fmt.Errorf("base dir: %w", err)
	}
	absBase, err := filepath.Abs(projectDir)
	if err != nil {
		return installConfig{}, fmt.Errorf("resolve install base: %w", err)
	}
	settingsFile, err := opts.SettingsFileName()
	if err != nil {
		return installConfig{}, fmt.Errorf("settings file: %w", err)
	}
	gitHooks, err := resolveGitHooks(opts)
	if err != nil {
		return installConfig{}, err
	}
	return installConfig{absBase: absBase, settingsFile: settingsFile, gitHooks: gitHooks}, nil
}
