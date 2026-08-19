package cli

import (
	"fmt"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/clean"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/generate"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/list"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/packaging"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/uninstall"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/validate"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
)

type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	// Fallback only: the command printed its findings before returning this.
	return fmt.Sprintf("exit code %d", e.Code)
}

func Run(inj remy.Injector, opts *version.Options) error {
	switch opts.Command {
	case "generate":
		return generate.RunGenerate(inj, opts)
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
		return list.RunList(inj, opts)
	case "validate":
		return validate.RunValidate(inj, opts)
	case "package":
		return packaging.RunPackage(inj, opts)
	case "hook":
		return hook.RunHook(inj, opts)
	case "commitlint":
		return runCommitLint(inj, opts)
	case "governance":
		return runGovernance(inj, opts)
	case "clean":
		return clean.RunClean(inj, opts)
	case "evals":
		return runEvals(inj, opts)
	default:
		return fmt.Errorf("unknown command %q", opts.Command)
	}
}

func HandleBootstrapErr(command string, err error) int {
	if command == "hook" {
		_, _ = fmt.Fprintln(os.Stderr, "pulsarules_cli: hook:", err)
		// A hook invocation must never block the caller's turn.
		return 0
	}
	_, _ = fmt.Fprintln(os.Stderr, "pulsarules_cli:", err)
	return 1
}

func BootstrapOptions(opts *version.Options) bootstrap.Options {
	// The bootstrap package never imports cli, so the translation lives here.
	return bootstrap.Options{
		Root:       opts.Root,
		ProjectDir: resolveProjectDir(opts),
		LogLevel:   opts.LogLevel,
		LogPath:    resolveLogPath(),
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
		return os.Getenv("PULSARULES_PROJECT_DIR")
	}
	return "."
}

func resolveLogPath() string {
	return os.Getenv("PULSARULES_LOG_PATH")
}
