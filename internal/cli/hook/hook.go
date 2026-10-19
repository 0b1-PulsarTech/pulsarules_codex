package hook

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/loader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/findings"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

func RunHook(inj remy.Injector, opts *version.Options) error {
	// simplification: every lookup below degrades to a no-op (log, return nil)
	// so a hook never blocks the caller's turn over a load/logger hiccup.
	// Upgrade path: revisit if a no-op ever hides a failure operators need paged on.
	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		printHookError(os.Stderr, "get knowledge index", err)
		return nil
	}
	assets, err := remy.Get[templates.Set](inj)
	if err != nil {
		printHookError(os.Stderr, "get templates", err)
		return nil
	}
	logger, err := remy.Get[*slog.Logger](inj)
	if err != nil {
		printHookError(os.Stderr, "build logger", err)
		return nil
	}
	if closer, closerErr := remy.Get[io.Closer](inj); closerErr == nil {
		defer func() { _ = closer.Close() }()
	}

	payload, _ := io.ReadAll(os.Stdin)

	disp := newHookDispatcher(idx, assets, logger, os.Stdout)

	if dispatchErr := disp.Dispatch(opts.Mode, payload); dispatchErr != nil {
		return fmt.Errorf("dispatch hook %q: %w", opts.Mode, dispatchErr)
	}
	return nil
}

func newHookDispatcher(
	idx *knowledge.Index,
	assets templates.Set,
	logger *slog.Logger,
	out io.Writer,
) *hook.Dispatcher {
	return hook.NewDispatcher(hook.Deps{
		Templates:  assets,
		Logger:     logger,
		Index:      idx,
		Out:        out,
		ProjectDir: os.Getenv("PULSARULES_PROJECT_DIR"),
		SkillsDir:  os.Getenv("PULSARULES_SKILLS_DIR"),
		Governance: func(repo vcs.Repository, status vcs.Status) (string, int) {
			return governChanges(repo, status, idx)
		},
	})
}

func printHookError(out io.Writer, label string, err error) {
	_, _ = fmt.Fprintln(out, "hook: "+label+":", printable.Line(err.Error()))
}

func governChanges(
	repo vcs.Repository,
	status vcs.Status,
	index *knowledge.Index,
) (string, int) {
	// The empty commit message is deliberate: only the git commit-msg hook validates one.
	cfg, configFindings := loader.ForRepo(repo)
	sess := governance.NewSession(repo, "", index, cfg)
	result := sess.Analyze(governance.ScopeChanged, &status, governance.FileSetChanged)
	findingsItem := slices.Concat(configFindings, result.Findings)

	block := findings.FormatFindings(findingsItem, findings.StyleHook, "Governance checks:")
	if block == "" {
		return "", 0
	}
	if result.SuppressedGenerated > 0 {
		// The hook has no footer of its own, so the suppression is stated here
		// rather than dropped: a filtered block must say it was filtered.
		block += fmt.Sprintf(
			"  %d finding(s) suppressed in generated files\n",
			result.SuppressedGenerated,
		)
	}
	return "\n" + block, len(findingsItem)
}
