package install

import (
	"fmt"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"
	"golang.org/x/term"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/claude"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
)

func Run(inj remy.Injector, opts *version.Options) error {
	cfg, err := resolveInstallConfig(opts)
	if err != nil {
		return err
	}
	deps, err := resolveInstallCollaborators(inj)
	if err != nil {
		return err
	}

	if opts.PrintHooks {
		return printHooksBlock()
	}

	console := newPrompter(os.Stdin, os.Stdout)
	console.isTerminal = term.IsTerminal(int(os.Stdin.Fd()))
	ctx, err := newTargetContext(opts, cfg, deps, console)
	if err != nil {
		return err
	}
	targets := opts.Targets()
	if err = deps.targets.Check(targets); err != nil {
		return err
	}
	for _, name := range targets {
		report, installErr := deps.targets.Install(name, ctx)
		report.Print(os.Stdout, os.Stderr)
		if installErr != nil {
			// The target registry's Install already wraps this as
			// `install target %q: %w`; re-wrapping here would double it.
			return installErr
		}
	}

	return installPostTargets(opts, cfg, deps)
}

func printHooksBlock() error {
	block, err := claude.RenderHooksBlock()
	if err != nil {
		return fmt.Errorf("render hooks block: %w", err)
	}
	_, _ = fmt.Print(string(block))
	return nil
}

func newTargetContext(
	opts *version.Options,
	cfg installConfig,
	deps installCollaborators,
	console *prompter,
) (host.Context, error) {
	if err := applySelectedLayout(opts, deps.idx, console); err != nil {
		return host.Context{}, fmt.Errorf("apply layout: %w", err)
	}
	languages, err := resolveLanguages(cfg.absBase, opts.Languages)
	if err != nil {
		return host.Context{}, err
	}

	selection, err := resolveSelection(opts, deps.idx, console)
	if err != nil {
		return host.Context{}, err
	}
	selection.Languages = languages
	ids, pulled := selection.Resolve(deps.idx)
	printDependencyPulls(os.Stdout, pulled)
	var routerFilter []string
	// Filter the rendered router to what's actually installed, unless it is
	// everything anyway (--all) or router-only (nothing to filter against).
	if !selection.All && !selection.RouterOnly {
		routerFilter = ids
	}

	return host.Context{
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
		Languages:      languages,
	}, nil
}

func resolveLanguages(absBase, override string) (detect.Set, error) {
	languages, err := detect.Resolve(absBase, override)
	if err != nil {
		return detect.Set{}, fmt.Errorf("resolve project languages: %w", err)
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

func resolveInstallConfig(opts *version.Options) (installConfig, error) {
	absBase, err := opts.BaseDir()
	if err != nil {
		return installConfig{}, fmt.Errorf("base dir: %w", err)
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
