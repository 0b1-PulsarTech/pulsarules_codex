package hook

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/loader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

type Collaborators struct {
	Index       *knowledge.Index
	Templates   templates.Set
	OpenLogger  func() (*slog.Logger, io.Closer, error)
	LoadCatalog func() (*emoji.Catalog, error)
	Loader      func() *loader.Layered
}

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

func (cmd *Command) Run(needs Collaborators, args []string) error {
	// simplification: the logger lookup below degrades to a no-op (log, return nil)
	// so a hook never blocks the caller's turn over a logger hiccup.
	// Upgrade path: revisit if a no-op ever hides a failure operators need paged on.
	logger, closer, err := needs.OpenLogger()
	if err != nil {
		printHookError(os.Stderr, "build logger", err)
		return nil
	}
	defer func() { _ = closer.Close() }()

	payload, _ := io.ReadAll(os.Stdin)

	// Only the stop checklist lints with it, so no other hook mode needs the catalog.
	disp := newHookDispatcher(hook.Deps{
		Templates:  needs.Templates,
		Logger:     logger,
		Index:      needs.Index,
		Out:        os.Stdout,
		ProjectDir: cmd.env.ProjectDir,
		SkillsDir:  cmd.env.SkillsDir,
	}, needs.LoadCatalog, needs.Loader)

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
	newLayered func() *loader.Layered,
) *hook.Dispatcher {
	deps.Governance = governor{index: deps.Index, loader: newLayered}.check(catalog)
	return hook.NewDispatcher(deps)
}

func printHookError(out io.Writer, label string, err error) {
	_, _ = fmt.Fprintln(out, "hook: "+label+":", printable.Line(err.Error()))
}
