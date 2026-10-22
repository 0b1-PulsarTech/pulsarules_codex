package generate

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/docs"
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

func (cmd *Command) Run(svc bootstrap.Services, _ []string) error {
	out := cmd.outDir()
	written, err := docs.Generate(svc.Index, svc.Renderer, out)
	if err != nil {
		return fmt.Errorf("generate output: %w", err)
	}
	fmt.Printf("generated %d skills into %s\n", len(written), filepath.Join(out, "skills"))
	return nil
}
