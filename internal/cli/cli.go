package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/uninstall"
)

// ExitError signals that main must exit the process with Code instead of the
// generic exit-1 path taken for every other returned error. Commands that
// used to call os.Exit directly (runGovernance, runCommitLint on an
// error-severity finding) return an *ExitError instead, keeping os.Exit the
// sole province of main.
type ExitError struct {
	Code int
}

// Error implements the error interface. The command has already printed its
// findings to stderr by the time it returns an ExitError, so the message is
// only a fallback for a caller that logs the error value directly.
func (e *ExitError) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}

// Run dispatches opts.Command to its handler. It is the ONE switchboard
// cmd/pulsarules_cli/main.go calls after the injector is wired.
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
	default:
		return fmt.Errorf("unknown command %q", opts.Command)
	}
}

// HandleBootstrapErr reports a composition-root wiring failure and returns the
// process exit code. A hook invocation must never block the caller's turn
// over a governance-loading hiccup, so it degrades to a no-op (exit 0)
// instead of failing like every other command.
func HandleBootstrapErr(command string, err error) int {
	if command == "hook" {
		_, _ = fmt.Fprintln(os.Stderr, "pulsarules_cli: hook:", err)
		return 0
	}
	_, _ = fmt.Fprintln(os.Stderr, "pulsarules_cli:", err)
	return 1
}

// BootstrapOptions mirrors the parsed CLI options into bootstrap.Options.
// bootstrap never imports this package, so cli is the one place that
// translates between them.
func BootstrapOptions(opts *cliopts.Options) bootstrap.Options {
	return bootstrap.Options{
		Root:       opts.Root,
		ProjectDir: resolveProjectDir(opts),
		LogLevel:   opts.LogLevel,
		LogPath:    resolveLogPath(),
	}
}

// resolveProjectDir mirrors each command's pre-DI project-directory fallback,
// so moving vcs.Repository behind the injector does not change behavior:
// governance requires an explicit project dir (--project or
// PULSARULES_PROJECT_DIR) and reports that itself; every other command
// tolerates a missing repository and defaults to ".".
func resolveProjectDir(opts *cliopts.Options) string {
	if opts.ProjectDir != "" {
		return opts.ProjectDir
	}
	if opts.Command == "governance" {
		return os.Getenv("PULSARULES_PROJECT_DIR")
	}
	return "."
}

// resolveLogPath mirrors the hook command's prior PULSARULES_PROJECT_DIR-based
// log path, leaving obs to pick its own default when the env var is unset.
func resolveLogPath() string {
	dir := os.Getenv("PULSARULES_PROJECT_DIR")
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, ".claude", "hook-execution.log")
}
