package cli

import (
	"flag"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
)

type (
	runner interface {
		Bind(fs *flag.FlagSet)
		Run(svc bootstrap.Services, args []string) error
	}

	anchored interface {
		ProjectDir() string
	}
)
