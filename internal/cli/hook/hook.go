package hook

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func RunHook(inj remy.Injector, opts *version.Options) error {
	// simplification: every lookup below degrades to a no-op (log, return nil)
	// so a hook never blocks the caller's turn over a load/logger hiccup.
	// Upgrade path: revisit if a no-op ever hides a failure operators need paged on.
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
