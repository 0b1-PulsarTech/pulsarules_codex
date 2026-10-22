package cli

import (
	"io"
	"slices"

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
	build func(env flags.Environment) runner
	terms []command.Term
	// A nil bootstrapFailed reports the error and exits 1.
	bootstrapFailed func(out io.Writer, err error) int
}

func table() []entry {
	return slices.Concat(renderEntries(), installEntries(), checkEntries())
}

func renderEntries() []entry {
	return []entry{
		{
			name:  "generate",
			build: func(flags.Environment) runner { return &generate.Command{} },
			terms: []command.Term{command.Opt("root"), command.Opt("out")},
		},
		{
			name:  "validate",
			build: func(flags.Environment) runner { return validate.Command{} },
			terms: []command.Term{command.Opt("root")},
		},
		{
			name:  "list",
			build: func(flags.Environment) runner { return list.Command{} },
			terms: []command.Term{
				command.Opt("root"),
				command.Lit("[skills|rules|patterns|workflows]"),
			},
		},
	}
}

func installEntries() []entry {
	return []entry{
		{
			name:  "install",
			build: func(env flags.Environment) runner { return install.New(env) },
			terms: []command.Term{
				command.Opt("root"),
				command.Req("global", "project"),
				command.Opt("target"),
				command.Req("all", "skills", "router-only"),
				command.Opt("no-hooks"),
				command.Opt("no-mcp"),
				command.Opt("print-hooks"),
				command.Opt("hooks-scope"),
				command.Opt("layout"),
				command.Opt("interactive"),
				command.Opt("git-hooks"),
				command.Opt("no-git-hooks"),
				command.Opt("languages"),
				command.Opt("typographic-severity"),
				command.Opt("branch-extra-types"),
			},
		},
		{
			name:  "uninstall",
			build: func(env flags.Environment) runner { return uninstall.New(env) },
			terms: []command.Term{
				command.Req("global", "project"),
				command.Opt("target"),
				command.Opt("hooks-scope"),
				command.Opt("keep-skills"),
			},
		},
		{
			name:  "package",
			build: func(flags.Environment) runner { return &packaging.Command{} },
			terms: []command.Term{command.Opt("root"), command.Opt("out")},
		},
	}
}

func checkEntries() []entry {
	return []entry{
		{
			name:  "commitlint",
			build: func(flags.Environment) runner { return &commitlint.Command{} },
			terms: []command.Term{command.Opt("msg", "file"), command.Opt("project")},
		},
		{
			name:  "governance",
			build: func(env flags.Environment) runner { return governance.New(env) },
			terms: []command.Term{
				command.Opt("project"),
				command.Opt("root"),
				command.Opt("preset"),
				command.Opt("scope"),
				command.Opt("config"),
				command.Opt("golangci-config"),
				command.Opt("all-files"),
				command.Opt("include-generated"),
				command.Opt("typographic-severity"),
				command.Opt("branch-extra-types"),
			},
		},
		{
			name:  "clean",
			build: func(flags.Environment) runner { return &clean.Command{} },
			terms: []command.Term{command.Opt("project"), command.Opt("write")},
		},
		{
			name:  "evals",
			build: func(flags.Environment) runner { return &evals.Command{} },
			terms: []command.Term{command.Req("artifact"), command.Opt("scenario")},
		},
		{
			name:            "hook",
			build:           func(env flags.Environment) runner { return hook.New(env) },
			terms:           []command.Term{command.Lit("<mode>")},
			bootstrapFailed: hook.BootstrapFailed,
		},
	}
}
