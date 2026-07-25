package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
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
	return "."
}

func resolveLogPath() string {
	dir := os.Getenv("CLAUDE_PROJECT_DIR")
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, ".claude", "hook-execution.log")
}
