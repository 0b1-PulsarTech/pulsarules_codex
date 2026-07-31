package bootstrap

import (
	"fmt"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// why: one Load call registers the index, templates, and renderer as
// singletons, replacing the separate knowledge.Load every run* command used
// to make on its own.
func registerKnowledge(inj remy.Injector, opts Options) error {
	idx, templates, err := knowledge.Load(opts.Root)
	if err != nil {
		return fmt.Errorf("load knowledge: %w", err)
	}
	rnd, err := render.NewRenderer(templates)
	if err != nil {
		return fmt.Errorf("new renderer: %w", err)
	}

	remy.RegisterInstance(inj, idx)
	remy.RegisterInstance(inj, templates)
	remy.RegisterInstance(inj, rnd)
	return nil
}
