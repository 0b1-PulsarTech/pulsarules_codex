package install

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/githook"
)

func resolveGitHooks(opts *cliopts.Options) ([]string, error) {
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
