package cli

import (
	"fmt"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func runPackage(inj remy.Injector, opts *cliopts.Options) error {
	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		return fmt.Errorf("get knowledge index: %w", err)
	}
	rnd, err := remy.Get[*render.Renderer](inj)
	if err != nil {
		return fmt.Errorf("get renderer: %w", err)
	}
	if err = output.Package(idx, rnd, opts.Out); err != nil {
		return fmt.Errorf("package: %w", err)
	}
	fmt.Printf("packaged: %s\n", opts.Out)
	return nil
}
