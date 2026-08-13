package install

import (
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

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

func installPostTargets(
	opts *version.Options,
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
