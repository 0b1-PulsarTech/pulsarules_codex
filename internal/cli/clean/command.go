package clean

import (
	"flag"
	"os"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

type Command struct {
	command.Project
	write bool
}

func (cmd *Command) Bind(fs *flag.FlagSet) {
	cmd.BindProject(fs, "project `DIR` to clean")
	fs.BoolVar(&cmd.write, "write", false, "rewrite files instead of only reporting")
}

func (cmd *Command) Run(openRepository func() (*vcs.Repository, error)) error {
	// Only --write mutates; the dangerous path is the one a caller has to ask for.
	how := inspecting()
	if cmd.write {
		how = rewriting()
	}
	repo, err := command.OptionalRepository(openRepository)
	if err != nil {
		return err
	}
	return sweep(os.Stdout, cmd.ProjectDir(), how, repo)
}
