package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/uninstall"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

type handler func(inj remy.Injector, opts *version.Options) error

func commands() map[string]handler {
	return map[string]handler{
		"install":    prefixed("install", install.Run),
		"uninstall":  prefixed("uninstall", uninstall.Run),
		"hook":       hook.RunHook,
		"governance": governance.RunGovernance,
	}
}

func prefixed(name string, run handler) handler {
	return func(inj remy.Injector, opts *version.Options) error {
		if runErr := run(inj, opts); runErr != nil {
			return fmt.Errorf("%s: %w", name, runErr)
		}
		return nil
	}
}

func (call Invocation) Run(inj remy.Injector) error {
	return call.Runner.Run(inj, call.Args)
}

func (call Invocation) HandleBootstrapErr(err error) int {
	if forgiving, ok := call.Runner.(command.Forgiving); ok {
		return forgiving.BootstrapFailed(os.Stderr, err)
	}
	ReportError(os.Stderr, err)
	return 1
}

func handleBootstrapErr(out io.Writer, command string, err error) int {
	if command == "hook" {
		_, _ = fmt.Fprintln(out, "pulsarules_cli: hook:", printable.Line(err.Error()))
		// A hook invocation must never block the caller's turn.
		return 0
	}
	ReportError(out, err)
	return 1
}

func ReportError(out io.Writer, err error) {
	_, _ = fmt.Fprintln(out, "pulsarules_cli:", printable.Line(err.Error()))
}

func (call Invocation) BootstrapOptions() bootstrap.Options {
	// The bootstrap package never imports cli, so the translation lives here.
	projectDir := "."
	if anchored, ok := call.Runner.(command.Anchored); ok {
		projectDir = anchored.ProjectDir()
	}
	return bootstrap.Options{
		Root:       call.Root,
		ProjectDir: projectDir,
		LogLevel:   call.LogLevel,
		LogPath:    call.Env.LogPath,
	}
}

func resolveProjectDir(opts *version.Options) string {
	if opts.ProjectDir != "" {
		return opts.ProjectDir
	}
	// Install/uninstall bind --project onto opts.Project, and that checkout is also
	// the analyzed repository.
	if opts.Command == "install" || opts.Command == "uninstall" {
		return opts.Project
	}
	if opts.Command == "governance" {
		return opts.Env.ProjectDir
	}
	return "."
}
