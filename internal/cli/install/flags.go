package install

import (
	"flag"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
)

type Command struct {
	env         flags.Environment
	global      bool
	project     string
	all         bool
	skills      []string
	target      []string
	routerOnly  bool
	noHooks     bool
	noMCP       bool
	printHooks  bool
	hooksScope  string
	layout      string
	interactive bool
	gitHooks    []string
	noGitHooks  bool
	languages   []string
	tuning      flags.Tuning
}

func New(env flags.Environment) *Command { return &Command{env: env} }

func (cmd *Command) targets() []string {
	if len(cmd.target) == 0 {
		const defaultTarget = "claude"
		return []string{defaultTarget}
	}
	return cmd.target
}

func (cmd *Command) Bind(fs *flag.FlagSet) {
	cmd.bindSelection(fs)
	cmd.bindWiring(fs)
}

func (cmd *Command) bindSelection(fs *flag.FlagSet) {
	fs.BoolVar(
		&cmd.global,
		"global",
		false,
		"install for the user: ~/.claude, or $CLAUDE_CONFIG_DIR when set",
	)
	fs.StringVar(&cmd.project, "project", "", "install to `PATH`/.claude/skills")
	fs.BoolVar(&cmd.all, "all", false, "install every skill")
	fs.Var(
		flags.NewCommaList(&cmd.skills),
		"skills",
		"comma-separated skill ids to install, as `a,b,c` (plus mandatory + their deps)",
	)
	fs.Var(
		flags.NewRepeatable(&cmd.target),
		"target",
		"install target (repeatable): `claude|opencode|agents|cursor ...` (default claude); "+
			"only claude supports --global",
	)
	fs.BoolVar(
		&cmd.routerOnly,
		"router-only",
		false,
		"install only project-router, its workflows, and gopls-navigation unless --no-mcp",
	)
	fs.BoolVar(
		&cmd.noHooks,
		"no-hooks",
		false,
		"skip the Claude hook script and settings wiring; git hooks use --no-git-hooks",
	)
	fs.BoolVar(
		&cmd.printHooks,
		"print-hooks",
		false,
		"print the resolved hooks block for the target and exit",
	)
}

func (cmd *Command) bindWiring(fs *flag.FlagSet) {
	fs.BoolVar(
		&cmd.noMCP,
		"no-mcp",
		false,
		"skip writing .mcp.json + generating the gopls-navigation skill",
	)
	fs.StringVar(
		&cmd.hooksScope,
		"hooks-scope",
		"project",
		"hook settings file: `project|local` (settings.json | settings.local.json)",
	)
	fs.StringVar(
		&cmd.layout,
		"layout",
		"",
		"customization `PROFILE` id to apply (e.g. monorepo | inner-modules)",
	)
	fs.BoolVar(
		&cmd.interactive,
		"interactive",
		false,
		"prompt for unset customizations (e.g. the layout) before installing",
	)
	fs.Var(
		flags.NewCommaList(&cmd.languages),
		"languages",
		"comma-separated override of the project's language set (e.g. `a,b`); "+
			"persisted, taking precedence over both a prior config and autodetection",
	)
	// Git-hook flags are install-time policy baked into the scripts, not render choices.
	cmd.bindGitHooks(fs)
}

func (cmd *Command) bindGitHooks(fs *flag.FlagSet) {
	cmd.gitHooks = []string{"commit-msg", "pre-commit"}
	fs.Var(
		flags.NewCommaList(&cmd.gitHooks),
		"git-hooks",
		"comma-separated git hooks to install: `commit-msg,pre-commit` (or pre-push)",
	)
	fs.BoolVar(
		&cmd.noGitHooks,
		"no-git-hooks",
		false,
		"skip git hooks installation entirely",
	)
	cmd.tuning.Bind(fs, flags.Bake)
}

func (cmd *Command) Terms() []command.Term {
	return []command.Term{
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
	}
}
