package cli

import (
	"flag"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
)

type (
	runner struct {
		command flagged
		run     func(svc bootstrap.Services, args []string) error
	}

	flagged interface {
		Bind(fs *flag.FlagSet)
	}

	anchored interface {
		ProjectDir() string
	}
)
