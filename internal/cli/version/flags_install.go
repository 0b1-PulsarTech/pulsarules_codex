package version

import (
	"flag"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
)

func bindInstallFlags(fs *flag.FlagSet, opts *Options) {
	bindInstallSelectionFlags(fs, opts)
	bindInstallWiringFlags(fs, opts)
}

func bindInstallSelectionFlags(fs *flag.FlagSet, opts *Options) {
	fs.BoolVar(&opts.Global, "global", false, "install to ~/.claude/skills")
	fs.StringVar(&opts.Project, "project", "", "install to PATH/.claude/skills")
	fs.BoolVar(&opts.All, "all", false, "install every skill")
	fs.Var(
		flags.NewCommaList(&opts.Skills),
		"skills",
		"comma-separated skill ids to install (plus mandatory + their deps)",
	)
	fs.Var(
		flags.NewRepeatable(&opts.Target),
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
	fs.Var(
		flags.NewCommaList(&opts.Languages),
		"languages",
		"comma-separated override of the project's language set (e.g. go,dart); "+
			"persisted, taking precedence over both a prior config and autodetection",
	)
	// Git-hook flags are install-time policy baked into the scripts, not render choices.
	bindGitHookFlags(fs, opts)
}

func bindGitHookFlags(fs *flag.FlagSet, opts *Options) {
	opts.GitHooks = []string{"commit-msg", "pre-commit"}
	fs.Var(
		flags.NewCommaList(&opts.GitHooks),
		"git-hooks",
		"comma-separated git hooks to install (commit-msg,pre-commit,pre-push)",
	)
	fs.BoolVar(
		&opts.NoGitHooks,
		"no-git-hooks",
		false,
		"skip git hooks installation entirely",
	)
	opts.Tuning.Bind(fs, flags.Bake)
}

func bindUninstallFlags(fs *flag.FlagSet, opts *Options) {
	fs.BoolVar(&opts.Global, "global", false, "uninstall from ~/.claude/skills")
	fs.StringVar(&opts.Project, "project", "", "uninstall from PATH/.claude/skills")
	fs.Var(
		flags.NewRepeatable(&opts.Target),
		"target",
		"uninstall target (repeatable): claude | opencode | agents | cursor "+
			"(default: every target detected on disk)",
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
