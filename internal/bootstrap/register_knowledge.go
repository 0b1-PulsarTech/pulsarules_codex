package bootstrap

import (
	"fmt"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func registerKnowledge(inj remy.Injector, opts Options) error {
	idx, templates, err := knowledge.Load(opts.Root)
	if err != nil {
		return fmt.Errorf("load knowledge: %w", err)
	}
	rnd, err := render.NewRenderer(templates)
	if err != nil {
		return fmt.Errorf("new renderer: %w", err)
	}

	// One Load feeds the index, templates and renderer, all registered as singletons.
	remy.RegisterInstance(inj, idx)
	remy.RegisterInstance(inj, templates)
	remy.RegisterInstance(inj, rnd)
	return nil
}
