package uninstall

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/target"
)

// retryHint is printed once, after a partial failure, so the operator knows a
// plain re-run finishes the job instead of needing to diagnose or clean up by
// hand: every step Run takes is idempotent.
const retryHint = "uninstall did not finish cleanly; every step is idempotent, " +
	"so re-running will finish the job"

// Run reverses install for every target: hook wiring, config, git hooks,
// and rendered docs (unless --keep-skills).
// why: unlike install it does not default to "claude" - with no --target it
// probes every target on disk and unwires both settings scopes, since it has
// no record which --hooks-scope install used. Every step is idempotent.
func Run(inj remy.Injector, opts *cliopts.Options) error {
	projectDir, err := opts.BaseDir()
	if err != nil {
		return fmt.Errorf("base dir: %w", err)
	}
	absBase, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("resolve uninstall base: %w", err)
	}
	settingsFiles, err := opts.SettingsFiles()
	if err != nil {
		return fmt.Errorf("settings files: %w", err)
	}
	deps, err := resolveUninstallCollaborators(inj)
	if err != nil {
		return err
	}

	targets, err := resolveTargets(opts, deps.targets, absBase)
	if err != nil {
		return err
	}

	ctx := target.UninstallContext{
		Base:             absBase,
		HookUninstallers: deps.hooks,
		SettingsFiles:    settingsFiles,
		KeepSkills:       opts.KeepSkills,
	}
	var errs []error
	for _, name := range targets {
		report, uninstallErr := deps.targets.Uninstall(name, ctx)
		printReport(report)
		if uninstallErr != nil {
			// why: target.Registry.Uninstall already wraps this as
			// `uninstall target %q: %w`; re-wrapping here would double it.
			errs = append(errs, uninstallErr)
		}
	}

	gitResult, hookErr := deps.hooks.Uninstall("git", install.UninstallContext{Dir: absBase})
	if hookErr != nil {
		errs = append(errs, hookErr)
	}
	if hookErr == nil && len(gitResult.Removed) > 0 {
		_, _ = fmt.Println("removed git hooks")
	}
	for _, msg := range gitResult.Restored {
		_, _ = fmt.Println(msg)
	}

	if len(errs) > 0 {
		_, _ = fmt.Fprintln(os.Stderr, retryHint)
		return errors.Join(errs...)
	}
	return nil
}

// resolveTargets validates an explicit --target list against reg, or - when
// none was passed - detects every layout present on disk under base, so
// uninstall acts on everything it can find rather than install's "claude
// only" default.
func resolveTargets(opts *cliopts.Options, reg *target.Registry, base string) ([]string, error) {
	explicit := opts.ExplicitTargets()
	if len(explicit) == 0 {
		return reg.DetectTargets(base), nil
	}
	for _, name := range explicit {
		if !reg.Has(name) {
			return nil, fmt.Errorf(
				"invalid --target %q (want %s)",
				name,
				strings.Join(reg.Names(), "|"),
			)
		}
	}
	return explicit, nil
}
