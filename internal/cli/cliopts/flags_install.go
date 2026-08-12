package cliopts

import "flag"

// bindInstallFlags registers the install subcommand's flags onto the flag set.
func bindInstallFlags(fs *flag.FlagSet, opts *Options) {
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

// bindUninstallFlags registers the uninstall subcommand's flags onto the flag
// set - a subset of install's, since uninstall never renders or selects
// skills to add, only removes what a prior install wrote.
func bindUninstallFlags(fs *flag.FlagSet, opts *Options) {
	fs.BoolVar(&opts.Global, "global", false, "uninstall from ~/.claude/skills")
	fs.StringVar(&opts.Project, "project", "", "uninstall from PATH/.claude/skills")
	fs.Var(
		(*stringSliceFlag)(&opts.Target),
		"target",
		"uninstall target (repeatable): claude | opencode | agents | cursor (default: every target detected on disk)",
	)
	fs.StringVar(
		&opts.HooksScope,
		"hooks-scope",
		"",
		"hook settings file to narrow to: project (settings.json) | local "+
			"(settings.local.json) (default: unwire both, since uninstall cannot "+
			"recover which scope install used)",
	)
	fs.BoolVar(
		&opts.KeepSkills,
		"keep-skills",
		false,
		"keep the rendered skill/workflow docs; the hook wiring is still removed",
	)
}
