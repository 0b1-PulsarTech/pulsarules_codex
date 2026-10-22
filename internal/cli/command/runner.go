package command

import (
	"flag"
	"io"

	"github.com/wrapped-owls/goremy-di/remy"
)

type (
	Runner interface {
		Bind(fs *flag.FlagSet)
		Terms() []Term
		Run(inj remy.Injector, args []string) error
	}

	Anchored interface {
		ProjectDir() string
	}

	Forgiving interface {
		BootstrapFailed(out io.Writer, err error) int
	}
)
