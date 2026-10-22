package hook

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

type Command struct {
	env flags.Environment
}

func New(env flags.Environment) *Command {
	return &Command{env: env}
}

func (*Command) Bind(*flag.FlagSet) {}

func BootstrapFailed(out io.Writer, err error) int {
	_, _ = fmt.Fprintln(out, "pulsarules_cli: hook:", printable.Line(err.Error()))
	// A hook invocation must never block the caller's turn.
	return 0
}

func (cmd *Command) Run(inj remy.Injector, args []string) error {
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

	// Only the stop checklist lints with it, so no other hook mode needs the catalog.
	catalog := func() (*emoji.Catalog, error) { return remy.Get[*emoji.Catalog](inj) }
	disp := newHookDispatcher(hook.Deps{
		Templates:  assets,
		Logger:     logger,
		Index:      idx,
		Out:        os.Stdout,
		ProjectDir: cmd.env.ProjectDir,
		SkillsDir:  cmd.env.SkillsDir,
	}, catalog)

	var mode string
	if len(args) > 0 {
		mode = args[0]
	}
	if dispatchErr := disp.Dispatch(mode, payload); dispatchErr != nil {
		return fmt.Errorf("dispatch hook %q: %w", mode, dispatchErr)
	}
	return nil
}

func newHookDispatcher(
	deps hook.Deps,
	catalog func() (*emoji.Catalog, error),
) *hook.Dispatcher {
	deps.Governance = governanceCheck(deps.Index, catalog)
	return hook.NewDispatcher(deps)
}

func printHookError(out io.Writer, label string, err error) {
	_, _ = fmt.Fprintln(out, "hook: "+label+":", printable.Line(err.Error()))
}
