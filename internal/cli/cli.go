package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

func (call Invocation) Run(svc bootstrap.Services) error {
	return call.runner.run(svc, call.args)
}

func (call Invocation) HandleBootstrapErr(err error) int {
	if call.bootstrapFailed != nil {
		return call.bootstrapFailed(os.Stderr, err)
	}
	ReportError(os.Stderr, err)
	return 1
}

func ReportError(out io.Writer, err error) {
	_, _ = fmt.Fprintln(out, "pulsarules_cli:", printable.Line(err.Error()))
}

func (call Invocation) BootstrapOptions() bootstrap.Options {
	// The bootstrap package never imports cli, so the translation lives here.
	projectDir := "."
	if anchored, ok := call.runner.command.(anchored); ok {
		projectDir = anchored.ProjectDir()
	}
	return bootstrap.Options{
		Root:          call.root,
		ProjectDir:    projectDir,
		LogLevel:      call.logLevel,
		LogPath:       call.env.LogPath,
		UserConfigDir: call.env.UserConfigDir,
	}
}
