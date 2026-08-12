package version

import "flag"

func bindInstallFlags(fs *flag.FlagSet, opts *Options) {
	bindInstallSelectionFlags(fs, opts)
	bindInstallWiringFlags(fs, opts)
}

func bindInstallSelectionFlags(fs *flag.FlagSet, opts *Options) {
	fs.BoolVar(&opts.Global, "global", false, "install to ~/.claude/skills")
	fs.StringVar(&opts.Project, "project", "", "install to PATH/.claude/skills")
	fs.BoolVar(&opts.All, "all", false, "install every skill")
	fs.StringVar(
		&opts.Skills,
		"skills",
		"",
		"comma-separated skill ids to install (plus mandatory + their deps)",
	)
	fs.Var(
		(*stringSliceFlag)(&opts.Target),
		"target",
		"install target (repeatable): claude | opencode | agents | cursor (default claude)",
	)
	fs.BoolVar(
		&opts.RouterOnly,
		"router-only",
		false,
		"install only project-router, its workflows, and gopls-navigation unless --no-mcp",
	)
	fs.BoolVar(
		&opts.NoHooks,
		"no-hooks",
		false,
		"skip the Claude hook script and settings wiring; git hooks use --no-git-hooks",
	)
	fs.BoolVar(
		&opts.PrintHooks,
		"print-hooks",
		false,
		"print the resolved hooks block for the target and exit",
	)
}

func bindInstallWiringFlags(fs *flag.FlagSet, opts *Options) {
	fs.BoolVar(
		&opts.NoMCP,
		"no-mcp",
		false,
		"skip writing .mcp.json + generating the gopls-navigation skill",
	)
	fs.StringVar(
		&opts.HooksScope,
		"hooks-scope",
		"project",
		"hook settings file: project (settings.json) | local (settings.local.json)",
	)
	fs.StringVar(
		&opts.Layout,
		"layout",
		"",
		"customization profile id to apply (e.g. monorepo | inner-modules)",
	)
	fs.BoolVar(
		&opts.Interactive,
		"interactive",
		false,
		"prompt for unset customizations (e.g. the layout) before installing",
	)
	fs.StringVar(
		&opts.GitHooks,
		"git-hooks",
		"commit-msg,pre-commit",
		"comma-separated git hooks to install (commit-msg,pre-commit,pre-push)",
	)
	fs.BoolVar(
		&opts.NoGitHooks,
		"no-git-hooks",
		false,
		"skip git hooks installation entirely",
	)
}
