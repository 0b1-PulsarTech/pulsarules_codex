package install

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/githook"
)

// resolveGitHooks parses --git-hooks, validating it against the known set
// unless --no-git-hooks makes the value moot (the hooks are never installed,
// so an unrecognized name in that case is not the caller's problem).
func resolveGitHooks(opts *cliopts.Options) ([]string, error) {
	hooks := opts.GitHookNames()
	if opts.NoGitHooks {
		return hooks, nil
	}
	if err := validateGitHooks(hooks); err != nil {
		return nil, err
	}
	return hooks, nil
}

// validateGitHooks rejects any --git-hooks name githook does not recognize,
// naming the valid set, before installPostTargets writes anything - the same
// discipline the --target loop in Run applies to target names.
func validateGitHooks(hooks []string) error {
	valid := githook.HookNames()
	for _, name := range hooks {
		if !slices.Contains(valid, name) {
			return fmt.Errorf("invalid --git-hooks %q (want %s)", name, strings.Join(valid, "|"))
		}
	}
	return nil
}
