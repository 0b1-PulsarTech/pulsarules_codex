package main

import (
	"errors"
	"flag"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
)

func main() {
	if tssyntax.RunChild(os.Args, os.Exit) {
		return
	}
	call, err := cli.Parse(os.Args[1:], flags.ReadEnvironment())
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		die(err)
	}

	if call.Name == "" || version.IsHelp(call.Name) {
		_ = version.Usage()
		return
	}
	if version.IsVersion(call.Name) {
		version.PrintVersion()
		return
	}

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if bootErr := bootstrap.DoInjections(inj, call.BootstrapOptions()); bootErr != nil {
		os.Exit(call.HandleBootstrapErr(bootErr))
	}

	if err = call.Run(inj); err != nil {
		die(err)
	}
}

func die(err error) {
	if errors.Is(err, command.ErrFindings) {
		os.Exit(1)
	}
	cli.ReportError(os.Stderr, err)
	os.Exit(1)
}
