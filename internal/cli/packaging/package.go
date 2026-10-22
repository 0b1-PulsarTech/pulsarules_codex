package packaging

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

func (cmd *Command) Run(svc bootstrap.Services, _ []string) error {
	out := cmd.outPath()
	if err := docs.Package(svc.Index, svc.Renderer, out); err != nil {
		return fmt.Errorf("package: %w", err)
	}
	fmt.Printf("packaged: %s\n", out)
	return nil
}
