package uninstall

import (
	"errors"
	"fmt"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
)

func Run(inj remy.Injector, opts *version.Options) error {
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

	ctx := host.UninstallContext{
		Base:          absBase,
		SettingsFiles: settingsFiles,
		KeepSkills:    opts.KeepSkills,
	}
	errs := uninstallTargets(deps, targets, ctx)

	gitReport, hookErr := install.UninstallGit(absBase)
	gitReport.Print(os.Stdout, os.Stderr)
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

func uninstallBase(opts *version.Options) (absBase string, settingsFiles []string, err error) {
	absBase, err = opts.BaseDir()
	if err != nil {
		return "", nil, fmt.Errorf("base dir: %w", err)
	}
	var scope flags.HooksScope
	scope, err = flags.ParseHooksScope(opts.HooksScope)
	if err != nil {
		return "", nil, fmt.Errorf("settings files: %w", err)
	}
	return absBase, scope.Files(), nil
}

func uninstallTargets(
	deps uninstallCollaborators,
	targets []string,
	ctx host.UninstallContext,
) []error {
	var errs []error
	for _, name := range targets {
		report, uninstallErr := deps.targets.Uninstall(name, ctx)
		report.Print(os.Stdout, os.Stderr)
		if uninstallErr != nil {
			// The target registry's Uninstall already wraps this as
			// `uninstall target %q: %w`; re-wrapping here would double it.
			errs = append(errs, uninstallErr)
		}
	}
	return errs
}

func removeLanguageConfig(absBase string) error {
	removed, err := detect.RemoveConfig(absBase)
	if err != nil {
		return err
	}
	// Project-wide, not per-target, so it is removed once here rather
	// than per target; kept, it would let a reinstall trust a stale set.
	if removed {
		_, _ = fmt.Println("removed", detect.ConfigFileName)
	}
	return nil
}

func resolveTargets(opts *version.Options, reg *host.Registry, base string) ([]string, error) {
	explicit := opts.ExplicitTargets()
	if len(explicit) == 0 {
		return reg.DetectTargets(base), nil
	}
	if err := reg.Check(explicit); err != nil {
		return nil, err
	}
	return explicit, nil
}
