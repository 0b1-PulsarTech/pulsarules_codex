package cli

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/clean"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/commitlint"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/evals"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/generate"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/list"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/packaging"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/uninstall"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/validate"
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
		{
			name:  "validate",
			build: func(flags.Environment) command.Runner { return validate.Command{} },
		},
		{
			name:  "list",
			build: func(flags.Environment) command.Runner { return list.Command{} },
		},
		{
			name:  "install",
			build: func(env flags.Environment) command.Runner { return install.New(env) },
		},
		{
			name:  "uninstall",
			build: func(env flags.Environment) command.Runner { return uninstall.New(env) },
		},
		{
			name:  "package",
			build: func(flags.Environment) command.Runner { return &packaging.Command{} },
		},
		{
			name:  "commitlint",
			build: func(flags.Environment) command.Runner { return &commitlint.Command{} },
		},
		{
			name:  "governance",
			build: func(env flags.Environment) command.Runner { return governance.New(env) },
		},
		{
			name:  "clean",
			build: func(flags.Environment) command.Runner { return &clean.Command{} },
		},
		{
			name:  "evals",
			build: func(flags.Environment) command.Runner { return &evals.Command{} },
		},
		{
			name:  "hook",
			build: func(env flags.Environment) command.Runner { return hook.New(env) },
		},
	}
}
