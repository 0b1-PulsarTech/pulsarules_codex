package cli

import (
	"flag"

	"github.com/wrapped-owls/goremy-di/remy"
)

type (
	runner interface {
		Bind(fs *flag.FlagSet)
		Run(inj remy.Injector, args []string) error
	}

	anchored interface {
		ProjectDir() string
	}
)
