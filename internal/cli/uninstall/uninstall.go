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
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/langdetect"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/target"
)

func Run(inj remy.Injector, opts *cliopts.Options) error {
	absBase, settingsFiles, err := uninstallBase(opts)
	if err != nil {
		return err
	}
	deps, err := resolveUninstallCollaborators(inj)
	if err != nil {
		return err
	}

	// No default target: with none given, every layout found on disk is unwired.
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
	errs := uninstallTargets(deps, targets, ctx)

	gitReport, hookErr := deps.hooks.Uninstall("git", install.UninstallContext{Dir: absBase})
	printReport(os.Stdout, os.Stderr, gitReport)
	if hookErr != nil {
		errs = append(errs, hookErr)
	}

	if langErr := removeLanguageConfig(absBase); langErr != nil {
		errs = append(errs, langErr)
	}

	if len(errs) > 0 {
		const retryHint = "uninstall did not finish cleanly; every step is idempotent, " +
			"so re-running will finish the job"
		_, _ = fmt.Fprintln(os.Stderr, retryHint)
		return errors.Join(errs...)
	}
	return nil
}

func uninstallBase(opts *cliopts.Options) (absBase string, settingsFiles []string, err error) {
	projectDir, dirErr := opts.BaseDir()
	if dirErr != nil {
		return "", nil, fmt.Errorf("base dir: %w", dirErr)
	}
	absBase, err = filepath.Abs(projectDir)
	if err != nil {
		return "", nil, fmt.Errorf("resolve uninstall base: %w", err)
	}
	settingsFiles, err = opts.SettingsFiles()
	if err != nil {
		return "", nil, fmt.Errorf("settings files: %w", err)
	}
	return absBase, settingsFiles, nil
}

func uninstallTargets(
	deps uninstallCollaborators,
	targets []string,
	ctx target.UninstallContext,
) []error {
	var errs []error
	for _, name := range targets {
		report, uninstallErr := deps.targets.Uninstall(name, ctx)
		printReport(os.Stdout, os.Stderr, report)
		if uninstallErr != nil {
			// The target registry's Uninstall already wraps this as
			// `uninstall target %q: %w`; re-wrapping here would double it.
			errs = append(errs, uninstallErr)
		}
	}
	return errs
}

func removeLanguageConfig(absBase string) error {
	removed, err := langdetect.RemoveConfig(absBase)
	if err != nil {
		return err
	}
	// Project-wide, not per-target, so it is removed once here rather
	// than per target; kept, it would let a reinstall trust a stale set.
	if removed {
		_, _ = fmt.Println("removed", langdetect.ConfigFileName)
	}
	return nil
}

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
