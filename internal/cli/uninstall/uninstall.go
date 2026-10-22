package uninstall

import (
	"errors"
	"fmt"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
)

func (cmd *Command) Run(inj remy.Injector, _ []string) error {
	if err := cmd.uninstall(inj); err != nil {
		return fmt.Errorf("uninstall: %w", err)
	}
	return nil
}

func (cmd *Command) ProjectDir() string { return cmd.project }

func (cmd *Command) uninstall(inj remy.Injector) error {
	ctx, err := uninstallContext(cmd)
	if err != nil {
		return err
	}
	deps, err := resolveUninstallCollaborators(inj)
	if err != nil {
		return err
	}

	// No default target: with none given, every layout found on disk is unwired.
	targets, err := resolveTargets(cmd, deps.targets, ctx.Base)
	if err != nil {
		return err
	}

	errs := uninstallTargets(deps, targets, ctx)
	if !cmd.global {
		errs = append(errs, uninstallProjectWide(ctx.Base)...)
	}

	if len(errs) > 0 {
		const retryHint = "uninstall did not finish cleanly; every step is idempotent, " +
			"so re-running will finish the job"
		_, _ = fmt.Fprintln(os.Stderr, retryHint)
		return errors.Join(errs...)
	}
	return nil
}

func uninstallProjectWide(absBase string) []error {
	// Git hooks and the language config belong to one repository, never a home directory.
	var errs []error
	gitReport, hookErr := install.UninstallGit(absBase)
	gitReport.Print(os.Stdout, os.Stderr)
	if hookErr != nil {
		errs = append(errs, hookErr)
	}
	if langErr := removeLanguageConfig(absBase); langErr != nil {
		errs = append(errs, langErr)
	}
	return errs
}

func uninstallContext(cmd *Command) (host.UninstallContext, error) {
	scope := flags.Scope{Global: cmd.global, Project: cmd.project}.WithEnv(cmd.env)
	absBase, err := flags.BaseDir(scope)
	if err != nil {
		return host.UninstallContext{}, fmt.Errorf("base dir: %w", err)
	}
	var claudeDir string
	if cmd.global {
		claudeDir, err = flags.UserClaudeDir(scope)
	}
	if err != nil {
		return host.UninstallContext{}, fmt.Errorf("claude dir: %w", err)
	}
	hooksScope, err := flags.ParseHooksScope(cmd.hooksScope)
	if err != nil {
		return host.UninstallContext{}, fmt.Errorf("settings files: %w", err)
	}
	return host.UninstallContext{
		Base:          absBase,
		Global:        cmd.global,
		ClaudeDir:     claudeDir,
		SettingsFiles: hooksScope.Files(),
		KeepSkills:    cmd.keepSkills,
	}, nil
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

func resolveTargets(cmd *Command, reg *host.Registry, base string) ([]string, error) {
	explicit := cmd.target
	if len(explicit) == 0 && cmd.global {
		// Only claude has a global layout; detection under the home directory
		// would also match a user's own AGENTS.md or .opencode.
		return []string{"claude"}, nil
	}
	if len(explicit) == 0 {
		return reg.DetectTargets(base), nil
	}
	if err := reg.Check(explicit); err != nil {
		return nil, err
	}
	return explicit, nil
}
