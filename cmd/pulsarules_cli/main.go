package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
)

func main() {
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
		die(bootErr)
	}

	if err = cli.Run(inj, opts); err != nil {
		die(err)
	}
}

func die(err error) {
	if exitErr, ok := errors.AsType[*cli.ExitError](err); ok {
		os.Exit(exitErr.Code)
	}
	_, _ = fmt.Fprintln(os.Stderr, "pulsarules_cli:", err)
	os.Exit(1)
}
