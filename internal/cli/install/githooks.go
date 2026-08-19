package install

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/branchname"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
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
	// why: rejected HERE, before the value is baked into a script. A typo that
	// installs cleanly would instead fail every commit from inside the hook,
	// where the person committing cannot see which flag was wrong.
	if err := core.ValidateSeverityName(opts.TypographicSeverity); err != nil {
		return nil, err
	}
	// why: the value is written INTO a generated shell script, so an entry
	// outside the allowed alphabet is refused here rather than quoted and hoped
	// for at the point it runs.
	if err := branchname.ValidateExtraTypes(opts.BranchExtraTypes); err != nil {
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
