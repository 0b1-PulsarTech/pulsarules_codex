package install

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ordered"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/githook"
)

func resolveGitHooks(opts *version.Options) ([]string, error) {
	hooks := ordered.Dedupe(opts.GitHooks)
	if opts.NoGitHooks {
		// Moot with --no-git-hooks: nothing is installed, so a bad name is harmless.
		return hooks, nil
	}
	if err := validateGitHooks(hooks); err != nil {
		return nil, err
	}
	if err := opts.Tuning.Validate(); err != nil {
		return nil, err
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
			TypographicSeverity: opts.Tuning.TypographicSeverity,
			BranchExtraTypes:    opts.Tuning.BranchExtraTypes,
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
