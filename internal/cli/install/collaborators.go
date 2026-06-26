package install

import (
	"fmt"
	"io/fs"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/target"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// why: groups the bindings Run resolves from the injector at its
// first lines, so that resolution is one call instead of five branches
// inline in Run.
type installCollaborators struct {
	idx       *knowledge.Index
	rnd       *render.Renderer
	templates fs.FS
	targets   *target.Registry
	hooks     *install.Registry
}

func resolveInstallCollaborators(inj remy.Injector) (installCollaborators, error) {
	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		return installCollaborators{}, fmt.Errorf("get knowledge index: %w", err)
	}
	rnd, err := remy.Get[*render.Renderer](inj)
	if err != nil {
		return installCollaborators{}, fmt.Errorf("get renderer: %w", err)
	}
	templates, err := remy.Get[fs.FS](inj)
	if err != nil {
		return installCollaborators{}, fmt.Errorf("get templates: %w", err)
	}
	targets, err := remy.Get[*target.Registry](inj)
	if err != nil {
		return installCollaborators{}, fmt.Errorf("get target registry: %w", err)
	}
	hooks, err := remy.Get[*install.Registry](inj)
	if err != nil {
		return installCollaborators{}, fmt.Errorf("get install registry: %w", err)
	}
	return installCollaborators{
		idx:       idx,
		rnd:       rnd,
		templates: templates,
		targets:   targets,
		hooks:     hooks,
	}, nil
}
