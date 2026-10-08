package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/uninstall"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/escape"
)

type ExitError struct {
	Code int
}

func (exitErr *ExitError) Error() string {
	// Fallback only: the command printed its findings before returning this.
	return fmt.Sprintf("exit code %d", exitErr.Code)
}

type command func(inj remy.Injector, opts *cliopts.Options) error

func commands() map[string]command {
	return map[string]command{
		"generate":   runGenerate,
		"install":    prefixed("install", install.Run),
		"uninstall":  prefixed("uninstall", uninstall.Run),
		"list":       runList,
		"validate":   runValidate,
		"package":    runPackage,
		"hook":       runHook,
		"commitlint": runCommitLint,
		"governance": runGovernance,
		"clean":      runClean,
		"evals":      runEvals,
	}
}

func prefixed(name string, run command) command {
	return func(inj remy.Injector, opts *cliopts.Options) error {
		if runErr := run(inj, opts); runErr != nil {
			return fmt.Errorf("%s: %w", name, runErr)
		}
		return nil
	}
}

func Run(inj remy.Injector, opts *cliopts.Options) error {
	run, ok := commands()[opts.Command]
	if !ok {
		return fmt.Errorf("unknown command %q", opts.Command)
	}
	return run(inj, opts)
}

func HandleBootstrapErr(command string, err error) int {
	return handleBootstrapErr(os.Stderr, command, err)
}

func handleBootstrapErr(out io.Writer, command string, err error) int {
	if command == "hook" {
		_, _ = fmt.Fprintln(out, "pulsarules_cli: hook:", escape.Line(err.Error()))
		// A hook invocation must never block the caller's turn.
		return 0
	}
	ReportError(out, err)
	return 1
}

func ReportError(out io.Writer, err error) {
	_, _ = fmt.Fprintln(out, "pulsarules_cli:", escape.Line(err.Error()))
}

func BootstrapOptions(opts *cliopts.Options) bootstrap.Options {
	// The bootstrap package never imports cli, so the translation lives here.
	return bootstrap.Options{
		Root:       opts.Root,
		ProjectDir: resolveProjectDir(opts),
		LogLevel:   opts.LogLevel,
		LogPath:    resolveLogPath(),
	}
}

func resolveProjectDir(opts *cliopts.Options) string {
	if opts.ProjectDir != "" {
		return opts.ProjectDir
	}
	// Install/uninstall bind --project onto opts.Project, and that checkout is also
	// the analyzed repository.
	if opts.Command == "install" || opts.Command == "uninstall" {
		return opts.Project
	}
	if opts.Command == "governance" {
		return os.Getenv("PULSARULES_PROJECT_DIR")
	}
	return "."
}

func resolveLogPath() string {
	return os.Getenv("PULSARULES_LOG_PATH")
}
