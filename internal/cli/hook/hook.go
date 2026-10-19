package hook

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
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

	disp := hook.NewDispatcher(hook.Deps{
		Templates:  assets,
		Logger:     logger,
		Index:      idx,
		Out:        os.Stdout,
		ProjectDir: os.Getenv("PULSARULES_PROJECT_DIR"),
		SkillsDir:  os.Getenv("PULSARULES_SKILLS_DIR"),
	})

	if dispatchErr := disp.Dispatch(opts.Mode, payload); dispatchErr != nil {
		return fmt.Errorf("dispatch hook %q: %w", opts.Mode, dispatchErr)
	}
	return nil
}

func printHookError(out io.Writer, label string, err error) {
	_, _ = fmt.Fprintln(out, "hook: "+label+":", printable.Line(err.Error()))
}
