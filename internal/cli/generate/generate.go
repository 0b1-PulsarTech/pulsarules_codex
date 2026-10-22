package generate

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
	fs.StringVar(&cmd.out, "out", "", "output `DIR` (defaults to ./generated)")
}

func (cmd *Command) outDir() string {
	if cmd.out == "" {
		return filepath.Join(".", "generated")
	}
	return cmd.out
}

func (cmd *Command) Run(index *knowledge.Index, renderer *render.Renderer) error {
	out := cmd.outDir()
	written, err := docs.Generate(index, renderer, out)
	if err != nil {
		return fmt.Errorf("generate output: %w", err)
	}
	fmt.Printf("generated %d skills into %s\n", len(written), filepath.Join(out, "skills"))
	return nil
}
