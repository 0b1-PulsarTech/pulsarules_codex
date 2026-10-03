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

func (e *ExitError) Error() string {
	// Fallback only: the command printed its findings before returning this.
	return fmt.Sprintf("exit code %d", e.Code)
}

func Run(inj remy.Injector, opts *cliopts.Options) error {
	switch opts.Command {
	case "generate":
		return runGenerate(inj, opts)
	case "install":
		if installErr := install.Run(inj, opts); installErr != nil {
			return fmt.Errorf("install: %w", installErr)
		}
		return nil
	case "uninstall":
		if uninstallErr := uninstall.Run(inj, opts); uninstallErr != nil {
			return fmt.Errorf("uninstall: %w", uninstallErr)
		}
		return nil
	case "list":
		return runList(inj, opts)
	case "validate":
		return runValidate(inj, opts)
	case "package":
		return runPackage(inj, opts)
	case "hook":
		return runHook(inj, opts)
	case "commitlint":
		return runCommitLint(inj, opts)
	case "governance":
		return runGovernance(inj, opts)
	case "clean":
		return runClean(inj, opts)
	case "evals":
		return runEvals(inj, opts)
	default:
		return fmt.Errorf("unknown command %q", opts.Command)
	}
}

func HandleBootstrapErr(command string, err error) int {
	return handleBootstrapErr(os.Stderr, command, err)
}

func handleBootstrapErr(w io.Writer, command string, err error) int {
	if command == "hook" {
		_, _ = fmt.Fprintln(w, "pulsarules_cli: hook:", escape.Line(err.Error()))
		// A hook invocation must never block the caller's turn.
		return 0
	}
	ReportError(w, err)
	return 1
}

func ReportError(w io.Writer, err error) {
	_, _ = fmt.Fprintln(w, "pulsarules_cli:", escape.Line(err.Error()))
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
