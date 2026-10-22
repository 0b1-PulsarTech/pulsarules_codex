package packaging

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/docs"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type Command struct {
	out string
}

func (cmd *Command) Bind(fs *flag.FlagSet) {
	fs.StringVar(
		&cmd.out,
		"out",
		"",
		"output zip `FILE` (defaults to ./build/standards-skills.zip)",
	)
}

func (cmd *Command) outPath() string {
	if cmd.out == "" {
		return filepath.Join(".", "build", "standards-skills.zip")
	}
	return cmd.out
}

func (cmd *Command) Run(inj remy.Injector, _ []string) error {
	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		return fmt.Errorf("get knowledge index: %w", err)
	}
	rnd, err := remy.Get[*render.Renderer](inj)
	if err != nil {
		return fmt.Errorf("get renderer: %w", err)
	}
	out := cmd.outPath()
	if err = docs.Package(idx, rnd, out); err != nil {
		return fmt.Errorf("package: %w", err)
	}
	fmt.Printf("packaged: %s\n", out)
	return nil
}

func (cmd *Command) Terms() []command.Term {
	return []command.Term{command.Opt("root"), command.Opt("out")}
}
