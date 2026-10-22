package governance

import (
	"flag"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
)

func (cmd *Command) Bind(fs *flag.FlagSet) {
	fs.StringVar(&cmd.project, "project", "", "project `DIR` to analyze")
	fs.StringVar(
		&cmd.preset,
		"preset",
		"",
		"analyzer preset (`recommended|strict|minimal`); wins over the project config file",
	)
	fs.StringVar(
		&cmd.config,
		"config",
		"",
		"project config file `PATH` (default: <project>/.pulsarules.yaml if present)",
	)
	fs.StringVar(&cmd.scope, "scope", "full", "analysis scope (`full|commit|changed`)")
	fs.StringVar(
		&cmd.golangciConfig,
		"golangci-config",
		"",
		".golangci.yml `PATH` (overrides auto-discovery)",
	)
	cmd.tuning.Bind(fs, flags.Analyze)
	fs.BoolVar(
		&cmd.allFiles,
		"all-files",
		false,
		"analyze every source file in the tree instead of only changed ones",
	)
	fs.BoolVar(
		&cmd.includeGenerated,
		"include-generated",
		false,
		"report findings in generated files too (suppressed by default)",
	)
}

func (cmd *Command) Terms() []command.Term {
	return []command.Term{
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
	}
}
