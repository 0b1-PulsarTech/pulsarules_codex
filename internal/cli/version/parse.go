package version

import (
	"flag"
	"fmt"
	"maps"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
)

func ParseArgs(args []string, env flags.Environment) (*Options, error) {
	opts := &Options{Env: env}
	if len(args) == 0 {
		return opts, nil
	}
	opts.Command = args[0]
	if IsHelp(opts.Command) || IsVersion(opts.Command) {
		return opts, nil
	}

	fs := flag.NewFlagSet(opts.Command, flag.ContinueOnError)
	fs.Usage = func() { _ = Usage() }
	flags.BindCommon(fs, &opts.Root, &opts.LogLevel, env)
	if err := bindCommandFlags(opts.Command, fs, opts); err != nil {
		return nil, err
	}
	if err := fs.Parse(args[1:]); err != nil {
		// %w keeps flag.ErrHelp, so main exits 0 after fs.Usage rather than reporting an error.
		return nil, fmt.Errorf("parse %s flags: %w", opts.Command, err)
	}

	parsePositionalArgs(opts, fs)
	return opts, nil
}

type flagBinder func(fs *flag.FlagSet, opts *Options)

func commandBinders() map[string]flagBinder {
	return map[string]flagBinder{
		"install":    bindInstallFlags,
		"uninstall":  bindUninstallFlags,
		"hook":       bindNoFlags,
		"governance": bindGovernanceFlags,
	}
}

func bindCommandFlags(command string, fs *flag.FlagSet, opts *Options) error {
	bind, ok := commandBinders()[command]
	if !ok {
		return fmt.Errorf("unknown command %q", command)
	}
	bind(fs, opts)
	return nil
}

func bindNoFlags(*flag.FlagSet, *Options) {}

func parsePositionalArgs(opts *Options, fs *flag.FlagSet) {
	if opts.Command == "hook" && fs.NArg() > 0 {
		opts.Mode = fs.Arg(0)
	}
}

func bindGovernanceFlags(fs *flag.FlagSet, opts *Options) {
	fs.StringVar(&opts.ProjectDir, "project", "", "project dir to analyze")
	fs.StringVar(
		&opts.Preset,
		"preset",
		"",
		"analyzer preset (recommended|strict|minimal); wins over the project config file",
	)
	fs.StringVar(
		&opts.Config,
		"config",
		"",
		"path to the project config file (default: <project>/.pulsarules.yaml if present)",
	)
	fs.StringVar(&opts.Scope, "scope", "full", "analysis scope (full|commit|changed)")
	fs.StringVar(
		&opts.GolangciConfig,
		"golangci-config",
		"",
		"path to .golangci.yml (overrides auto-discovery)",
	)
	opts.Tuning.Bind(fs, flags.Analyze)
	fs.BoolVar(
		&opts.AllFiles,
		"all-files",
		false,
		"analyze every source file in the tree instead of only changed ones",
	)
	fs.BoolVar(
		&opts.IncludeGenerated,
		"include-generated",
		false,
		"report findings in generated files too (suppressed by default)",
	)
}

func Commands() []string {
	return slices.Sorted(maps.Keys(commandBinders()))
}
