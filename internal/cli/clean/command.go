package clean

import (
	"flag"
	"os"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
)

type Command struct {
	project string
	write   bool
}

func (cmd *Command) Bind(fs *flag.FlagSet) {
	fs.StringVar(&cmd.project, "project", "", "project dir to clean")
	fs.BoolVar(&cmd.write, "write", false, "rewrite files instead of only reporting")
}

func (cmd *Command) ProjectDir() string {
	if cmd.project == "" {
		return "."
	}
	return cmd.project
}

func (cmd *Command) Run(inj remy.Injector, _ []string) error {
	// Only --write mutates; the dangerous path is the one a caller has to ask for.
	how := inspecting()
	if cmd.write {
		how = rewriting()
	}
	repo, err := command.OptionalRepository(inj)
	if err != nil {
		return err
	}
	return sweep(os.Stdout, cmd.ProjectDir(), how, repo)
}
