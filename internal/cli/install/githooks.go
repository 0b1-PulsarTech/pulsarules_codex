package install

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/branch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/githook"
)

func resolveGitHooks(opts *version.Options) ([]string, error) {
	hooks := opts.GitHookNames()
	if opts.NoGitHooks {
		// Moot with --no-git-hooks: nothing is installed, so a bad name is harmless.
		return hooks, nil
	}
	if err := validateGitHooks(hooks); err != nil {
		return nil, err
	}
	// Reject before the value is baked into a script: a typo that installs cleanly
	// would fail every commit from inside the hook, where the flag is not visible.
	if err := finding.ValidateSeverityName(opts.TypographicSeverity); err != nil {
		return nil, fmt.Errorf("invalid --typographic-severity: %w", err)
	}
	// The value is written into a generated shell script, so an entry
	// outside the allowed alphabet is refused here rather than quoted and hoped
	// for at the point it runs.
	if err := branch.ValidateExtraTypes(opts.BranchExtraTypes); err != nil {
		return nil, fmt.Errorf("invalid --branch-extra-types: %w", err)
	}
	return hooks, nil
}

func validateGitHooks(hooks []string) error {
	valid := githook.HookNames()
	for _, name := range hooks {
		if !slices.Contains(valid, name) {
			return fmt.Errorf("invalid --git-hooks %q (want %s)", name, strings.Join(valid, "|"))
		}
	}
	return nil
}

func installPostTargets(opts *version.Options, cfg installConfig) error {
	if !opts.NoGitHooks {
		rpt, hookErr := install.Git(cfg.absBase, cfg.gitHooks, githook.Options{
			TypographicSeverity: opts.TypographicSeverity,
			BranchExtraTypes:    opts.BranchExtraTypes,
		})
		rpt.Print(os.Stdout, os.Stderr)
		if hookErr != nil {
			printWarning(os.Stderr, "git hooks", hookErr)
		}
	}

	// The binary is copied even with --no-git-hooks.
	binaryErr := githook.InstallBinary(cfg.absBase)
	if binaryErr != nil {
		printWarning(os.Stderr, "binary", binaryErr)
	} else {
		_, _ = fmt.Println("installed governance binary")
	}

	return nil
}
