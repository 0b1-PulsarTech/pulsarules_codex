package packaging

import (
	"flag"
	"fmt"
	"path/filepath"

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

func (cmd *Command) Run(index *knowledge.Index, renderer *render.Renderer) error {
	out := cmd.outPath()
	if err := docs.Package(index, renderer, out); err != nil {
		return fmt.Errorf("package: %w", err)
	}
	fmt.Printf("packaged: %s\n", out)
	return nil
}
