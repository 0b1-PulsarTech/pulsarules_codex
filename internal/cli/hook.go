package cli

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func runHook(inj remy.Injector, opts *cliopts.Options) error {
	// simplification: every lookup below degrades to a no-op (log, then
	// return nil) rather than a returned error, because a hook must never
	// block the caller's turn over a governance-loading or logger-setup
	// hiccup. The ceiling: a corrupt knowledge base, a bad --log-level, or
	// unreadable stdin degrades to a no-op hook instead of surfacing the
	// failure.
	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "hook: get knowledge index:", err)
		return nil
	}
	templates, err := remy.Get[fs.FS](inj)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "hook: get templates:", err)
		return nil
	}
	logger, err := remy.Get[*slog.Logger](inj)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "hook: build logger:", err)
		return nil
	}
	if closer, closerErr := remy.Get[io.Closer](inj); closerErr == nil {
		defer func() { _ = closer.Close() }()
	}

	payload, _ := io.ReadAll(os.Stdin)

	disp := hook.NewDispatcher(hook.Deps{
		Templates:  templates,
		Logger:     logger,
		Index:      idx,
		Out:        os.Stdout,
		ProjectDir: os.Getenv("CLAUDE_PROJECT_DIR"),
	})

	if dispatchErr := disp.Dispatch(opts.Mode, payload); dispatchErr != nil {
		return fmt.Errorf("dispatch hook %q: %w", opts.Mode, dispatchErr)
	}
	return nil
}
