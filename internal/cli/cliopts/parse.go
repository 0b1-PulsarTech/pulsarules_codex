package cliopts

import (
	"flag"
	"fmt"
	"os"
)

// ParseArgs binds the CLI flags for the chosen subcommand onto opts and applies
// defaults. A parse failure wraps flag.ErrHelp, which main detects with
// errors.Is so it can print usage cleanly instead of an error.
func ParseArgs(args []string) (*Options, error) {
	opts := &Options{}
	if len(args) == 0 {
		return opts, nil
	}
	opts.Command = args[0]
	if IsHelp(opts.Command) || IsVersion(opts.Command) {
		return opts, nil
	}

	fs := flag.NewFlagSet(opts.Command, flag.ContinueOnError)
	fs.Usage = func() { _ = Usage() }
	fs.StringVar(&opts.Root, "root", "", "repository root for dev mode (defaults to embedded)")
	fs.StringVar(
		&opts.LogLevel,
		"log-level",
		os.Getenv("PULSARULES_LOG_LEVEL"),
		"hook execution log level: debug|info|warn|error (empty disables logging)",
	)
	if err := bindCommandFlags(opts.Command, fs, opts); err != nil {
		return nil, err
	}
	if err := fs.Parse(args[1:]); err != nil {
		return nil, fmt.Errorf("parse %s flags: %w", opts.Command, err)
	}

	parsePositionalArgs(opts, fs)
	opts.applyDefaults()
	return opts, nil
}

// bindCommandFlags registers the command-specific flags onto the flag set.
func bindCommandFlags(command string, fs *flag.FlagSet, opts *Options) error {
	switch command {
	case "generate":
		fs.StringVar(&opts.Out, "out", "", "output dir (defaults to ./generated)")
	case "install":
		bindInstallFlags(fs, opts)
	case "uninstall":
		bindUninstallFlags(fs, opts)
	case "package":
		fs.StringVar(
			&opts.Out,
			"out",
			"",
			"output zip path (defaults to ./build/standards-skills.zip)",
		)
	case "list", "validate", "hook":
	case "commitlint":
		fs.StringVar(&opts.CommitMsg, "msg", "", "commit message to validate")
		fs.StringVar(&opts.CommitFile, "file", "", "path to a COMMIT_EDITMSG file")
		fs.StringVar(&opts.ProjectDir, "project", "", "project dir for git history lookup")
	case "governance":
		fs.StringVar(&opts.ProjectDir, "project", "", "project dir to analyze")
		fs.StringVar(
			&opts.Preset,
			"preset",
			"recommended",
			"analyzer preset (recommended|strict|minimal)",
		)
		fs.StringVar(
			&opts.Scope,
			"scope",
			"full",
			"analysis scope (full|commit)",
		)
		fs.StringVar(
			&opts.GolangciConfig,
			"golangci-config",
			"",
			"path to .golangci.yml (overrides auto-discovery)",
		)
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
	default:
		return fmt.Errorf("unknown command %q", command)
	}
	return nil
}

// parsePositionalArgs handles positional argument parsing for list and hook.
func parsePositionalArgs(opts *Options, fs *flag.FlagSet) {
	if opts.Command == "list" {
		opts.Kind = "skills"
		if fs.NArg() > 0 {
			opts.Kind = fs.Arg(0)
		}
	}
	if opts.Command == "hook" && fs.NArg() > 0 {
		opts.Mode = fs.Arg(0)
	}
}
