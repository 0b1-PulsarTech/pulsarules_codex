package uninstall

import (
	"flag"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
)

type Command struct {
	env        flags.Environment
	global     bool
	project    string
	target     []string
	hooksScope string
	keepSkills bool
}

func New(env flags.Environment) *Command { return &Command{env: env} }

func (cmd *Command) Bind(fs *flag.FlagSet) {
	fs.BoolVar(
		&cmd.global,
		"global",
		false,
		"uninstall for the user: ~/.claude, or $CLAUDE_CONFIG_DIR when set",
	)
	fs.StringVar(&cmd.project, "project", "", "uninstall from `PATH`/.claude/skills")
	fs.Var(
		flags.NewRepeatable(&cmd.target),
		"target",
		"uninstall target (repeatable): `claude|opencode|agents|cursor ...` "+
			"(default: every target detected on disk)",
	)
	fs.StringVar(
		&cmd.hooksScope,
		"hooks-scope",
		"",
		"hook settings file to narrow to: `project|local` (settings.json | "+
			"settings.local.json) (default: unwire both, since uninstall cannot "+
			"recover which scope install used)",
	)
	fs.BoolVar(
		&cmd.keepSkills,
		"keep-skills",
		false,
		"keep the rendered skill/workflow docs; the hook wiring is still removed",
	)
}
