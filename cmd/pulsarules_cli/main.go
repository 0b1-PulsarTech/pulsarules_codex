package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
)

func main() {
	opts, err := cliopts.ParseArgs(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		die(err)
	}

	if opts.Command == "" || cliopts.IsHelp(opts.Command) {
		_ = cliopts.Usage()
		return
	}
	if cliopts.IsVersion(opts.Command) {
		cliopts.PrintVersion()
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
	_, _ = fmt.Fprintln(os.Stderr, "pulsarules_cli:", err)
	os.Exit(1)
}
