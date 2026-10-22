package cli

import (
	"io"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/hook"
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
			build: generateRunner,
			terms: []command.Term{command.Opt("root"), command.Opt("out")},
		},
		{
			name:  "validate",
			build: validateRunner,
			terms: []command.Term{command.Opt("root")},
		},
		{
			name:  "list",
			build: listRunner,
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
			build: installRunner,
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
			build: uninstallRunner,
			terms: []command.Term{
				command.Req("global", "project"),
				command.Opt("target"),
				command.Opt("hooks-scope"),
				command.Opt("keep-skills"),
			},
		},
		{
			name:  "package",
			build: packageRunner,
			terms: []command.Term{command.Opt("root"), command.Opt("out")},
		},
	}
}

func checkEntries() []entry {
	return []entry{
		{
			name:  "commitlint",
			build: commitlintRunner,
			terms: []command.Term{command.Opt("msg", "file"), command.Opt("project")},
		},
		{
			name:  "governance",
			build: governanceRunner,
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
			build: cleanRunner,
			terms: []command.Term{command.Opt("project"), command.Opt("write")},
		},
		{
			name:  "evals",
			build: evalsRunner,
			terms: []command.Term{command.Req("artifact"), command.Opt("scenario")},
		},
		{
			name:            "hook",
			build:           hookRunner,
			terms:           []command.Term{command.Lit("<mode>")},
			bootstrapFailed: hook.BootstrapFailed,
		},
	}
}
