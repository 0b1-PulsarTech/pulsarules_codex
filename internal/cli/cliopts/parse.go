package cliopts

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
)

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
		// %w keeps flag.ErrHelp, so main exits 0 after fs.Usage rather than reporting an error.
		return nil, fmt.Errorf("parse %s flags: %w", opts.Command, err)
	}

	parsePositionalArgs(opts, fs)
	opts.applyDefaults()
	return opts, nil
}

type flagBinder func(fs *flag.FlagSet, opts *Options)

func commandBinders() map[string]flagBinder {
	return map[string]flagBinder{
		"generate":   bindGenerateFlags,
		"install":    bindInstallFlags,
		"uninstall":  bindUninstallFlags,
		"package":    bindPackageFlags,
		"list":       bindNoFlags,
		"validate":   bindNoFlags,
		"hook":       bindNoFlags,
		"commitlint": bindCommitLintFlags,
		"clean":      bindCleanFlags,
		"evals":      bindEvalsFlags,
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

func bindGenerateFlags(fs *flag.FlagSet, opts *Options) {
	fs.StringVar(&opts.Out, "out", "", "output dir (defaults to ./generated)")
}

func bindPackageFlags(fs *flag.FlagSet, opts *Options) {
	fs.StringVar(
		&opts.Out,
		"out",
		"",
		"output zip path (defaults to ./build/standards-skills.zip)",
	)
}

func bindCommitLintFlags(fs *flag.FlagSet, opts *Options) {
	fs.StringVar(&opts.CommitMsg, "msg", "", "commit message to validate")
	fs.StringVar(&opts.CommitFile, "file", "", "path to a COMMIT_EDITMSG file")
	fs.StringVar(&opts.ProjectDir, "project", "", "project dir for git history lookup")
}

func bindCleanFlags(fs *flag.FlagSet, opts *Options) {
	fs.StringVar(&opts.ProjectDir, "project", "", "project dir to clean")
	fs.BoolVar(&opts.Write, "write", false, "rewrite files instead of only reporting")
}

func bindEvalsFlags(fs *flag.FlagSet, opts *Options) {
	fs.StringVar(&opts.Artifact, "artifact", "", "path to the produced artifact to grade")
	fs.StringVar(&opts.Scenario, "scenario", "", "grade only this scenario id")
}

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

func bindGovernanceFlags(fs *flag.FlagSet, opts *Options) {
	fs.StringVar(&opts.ProjectDir, "project", "", "project dir to analyze")
	fs.StringVar(
		&opts.Preset,
		"preset",
		"recommended",
		"analyzer preset (recommended|strict|minimal)",
	)
	fs.StringVar(&opts.Scope, "scope", "full", "analysis scope (full|commit|changed)")
	fs.StringVar(
		&opts.GolangciConfig,
		"golangci-config",
		"",
		"path to .golangci.yml (overrides auto-discovery)",
	)
	fs.StringVar(
		&opts.TypographicSeverity,
		"typographic-severity",
		"",
		"how hard a typographic-marker finding lands (error|warning|info)",
	)
	fs.StringVar(
		&opts.BranchExtraTypes,
		"branch-extra-types",
		"",
		"comma-separated branch types allowed beyond the Conventional Commit and gitflow types",
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
}

func Commands() []string {
	return slices.Sorted(maps.Keys(commandBinders()))
}
