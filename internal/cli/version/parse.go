package version

import (
	"flag"
	"fmt"
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

func bindCommandFlags(command string, fs *flag.FlagSet, opts *Options) error {
	switch command {
	case "generate":
		fs.StringVar(&opts.Out, "out", "", "output dir (defaults to ./generated)")
	case "package":
		fs.StringVar(
			&opts.Out,
			"out",
			"",
			"output zip path (defaults to ./build/standards-skills.zip)",
		)
	case "list", "validate":
	case "commitlint":
		fs.StringVar(&opts.CommitMsg, "msg", "", "commit message to validate")
		fs.StringVar(&opts.CommitFile, "file", "", "path to a COMMIT_EDITMSG file")
		fs.StringVar(&opts.ProjectDir, "project", "", "project dir for git history lookup")
	default:
		return fmt.Errorf("unknown command %q", command)
	}
	return nil
}

func parsePositionalArgs(opts *Options, fs *flag.FlagSet) {
	if opts.Command == "list" {
		opts.Kind = "skills"
		if fs.NArg() > 0 {
			opts.Kind = fs.Arg(0)
		}
	}
}
