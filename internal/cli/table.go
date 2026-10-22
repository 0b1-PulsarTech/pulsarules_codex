package cli

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/generate"
)

type entry struct {
	name  string
	build func(env flags.Environment) command.Runner
}

func table() []entry {
	return []entry{
		{
			name:  "generate",
			build: func(flags.Environment) command.Runner { return &generate.Command{} },
		},
	}
}
