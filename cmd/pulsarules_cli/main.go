package main

import (
	"errors"
	"flag"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
)

func main() {
	if tssyntax.RunChild(os.Args, os.Exit) {
		return
	}
	opts, err := version.ParseArgs(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		die(err)
	}

	if opts.Command == "" || version.IsHelp(opts.Command) {
		_ = version.Usage()
		return
	}
	if version.IsVersion(opts.Command) {
		version.PrintVersion()
		return
	}

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if bootErr := bootstrap.DoInjections(inj, cli.BootstrapOptions(opts)); bootErr != nil {
		os.Exit(cli.HandleBootstrapErr(opts.Command, bootErr))
	}

	if err = cli.Run(inj, opts); err != nil {
		die(err)
	}
}

func die(err error) {
	if exitErr, ok := errors.AsType[*cli.ExitError](err); ok {
		os.Exit(exitErr.Code)
	}
	cli.ReportError(os.Stderr, err)
	os.Exit(1)
}
