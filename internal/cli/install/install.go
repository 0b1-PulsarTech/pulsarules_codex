package install

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/githook"
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
			return fmt.Errorf("install target %q: %w", name, installErr)
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
	selection, err := resolveSelection(opts, deps.idx)
	if err != nil {
		return target.Context{}, err
	}
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

func installPostTargets(
	opts *cliopts.Options,
	projectDir string,
	templates fs.FS,
	hookReg *install.Registry,
	gitHooks []string,
) error {
	if !opts.NoGitHooks {
		hookErr := hookReg.Install("git", install.Context{
			Dir:       projectDir,
			Templates: templates,
			GitHooks:  gitHooks,
			Warn: func(format string, args ...any) {
				_, _ = fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
			},
		})
		if hookErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: git hooks: %v\n", hookErr)
		} else {
			_, _ = fmt.Printf("installed git hooks: %s\n", strings.Join(gitHooks, ","))
		}
	}

	// The binary is copied even with --no-git-hooks.
	binaryErr := githook.InstallBinary(projectDir)
	if binaryErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: binary: %v\n", binaryErr)
	} else {
		_, _ = fmt.Println("installed governance binary")
	}

	return nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		if id := strings.TrimSpace(part); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
