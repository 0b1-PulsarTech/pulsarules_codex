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
	targets := opts.Targets()
	cfg, err := resolveInstallConfig(opts)
	if err != nil {
		return err
	}
	absBase, settingsFile, gitHooks := cfg.absBase, cfg.settingsFile, cfg.gitHooks
	deps, err := resolveInstallCollaborators(inj)
	if err != nil {
		return err
	}
	idx, rnd, templates, reg, hookReg := deps.idx, deps.rnd, deps.templates, deps.targets, deps.hooks

	if opts.PrintHooks {
		block, blockErr := hookwire.RenderHooksBlock(templates)
		if blockErr != nil {
			return fmt.Errorf("render hooks block: %w", blockErr)
		}
		_, _ = fmt.Print(string(block))
		return nil
	}

	if err = applySelectedLayout(opts, idx); err != nil {
		return fmt.Errorf("apply layout: %w", err)
	}

	selection, err := resolveSelection(opts, idx)
	if err != nil {
		return err
	}
	ids, pulled := selection.Resolve(idx)
	printDependencyPulls(pulled)
	// Filter the rendered router to what's actually installed, unless it is
	// everything anyway (--all) or router-only (nothing to filter against).
	var routerFilter []string
	if !selection.All && !selection.RouterOnly {
		routerFilter = ids
	}

	for _, name := range targets {
		if !reg.Has(name) {
			return fmt.Errorf("invalid --target %q (want %s)", name, strings.Join(reg.Names(), "|"))
		}
	}
	ctx := target.Context{
		Templates:      templates,
		Index:          idx,
		Renderer:       rnd,
		HookInstallers: hookReg,
		Base:           absBase,
		IDs:            ids,
		RouterFilter:   routerFilter,
		NoMCP:          opts.NoMCP,
		NoHooks:        opts.NoHooks,
		SettingsFile:   settingsFile,
	}
	for _, name := range targets {
		report, installErr := reg.Install(name, ctx)
		printReport(report)
		if installErr != nil {
			return fmt.Errorf("install target %q: %w", name, installErr)
		}
	}

	return installPostTargets(opts, absBase, templates, hookReg, gitHooks)
}

// installConfig groups the small derived values Run resolves before touching
// any collaborator, so a single err check replaces four.
type installConfig struct {
	absBase      string
	settingsFile string
	gitHooks     []string
}

// resolveInstallConfig resolves the install base dir, the hook settings file,
// and the validated --git-hooks list from opts.
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

// installPostTargets runs the post-target wiring: git hooks, governance
// config, and binary copy. The binary is always copied so it is available
// even when --no-git-hooks is set. hookReg is resolved once by Run and
// threaded in rather than constructed here.
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
