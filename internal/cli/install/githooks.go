package install

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ordered"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/githook"
)

func resolveGitHooks(cmd *Command) ([]string, error) {
	hooks := ordered.Dedupe(cmd.gitHooks)
	if cmd.noGitHooks {
		// Moot with --no-git-hooks: nothing is installed, so a bad name is harmless.
		return hooks, nil
	}
	if err := validateGitHooks(hooks); err != nil {
		return nil, err
	}
	if err := cmd.tuning.Validate(); err != nil {
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

func installPostTargets(cmd *Command, cfg installConfig) error {
	if cfg.global {
		// Git hooks and the hook binary live in a repository's .git, not in a home directory.
		if !cmd.noGitHooks {
			_, _ = fmt.Println("git hooks are per repository; skipped under --global")
		}
		return nil
	}
	if !cmd.noGitHooks {
		rpt, hookErr := install.Git(cfg.absBase, cfg.gitHooks, githook.Options{
			TypographicSeverity: cmd.tuning.TypographicSeverity,
			BranchExtraTypes:    cmd.tuning.BranchExtraTypes,
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
