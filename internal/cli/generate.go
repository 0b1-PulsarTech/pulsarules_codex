package cli

import (
	"fmt"
	"path/filepath"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func runGenerate(inj remy.Injector, opts *cliopts.Options) error {
	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		return fmt.Errorf("get knowledge index: %w", err)
	}
	rnd, err := remy.Get[*render.Renderer](inj)
	if err != nil {
		return fmt.Errorf("get renderer: %w", err)
	}
	written, err := output.Generate(idx, rnd, opts.Out)
	if err != nil {
		return fmt.Errorf("generate output: %w", err)
	}
	fmt.Printf("generated %d skills into %s\n", len(written), filepath.Join(opts.Out, "skills"))
	return nil
}
