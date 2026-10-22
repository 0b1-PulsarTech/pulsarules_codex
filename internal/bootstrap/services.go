package bootstrap

import (
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/loader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
)

type Services struct {
	Index     *knowledge.Index
	Templates templates.Set
	Renderer  *render.Renderer
	Targets   *host.Registry
	// Each of these can fail, so it stays lazy and the command that asks decides if it is fatal.
	OpenRepository func() (*vcs.Repository, error)
	OpenLogger     func() (*slog.Logger, io.Closer, error)
	LoadCatalog    func() (*emoji.Catalog, error)
	// Built on the first call, so only a command that reads project config pays for it.
	Loader func() *loader.Layered
}

func resolve(inj remy.Injector, build builders) (Services, error) {
	var svc Services
	var err error
	if svc.Index, err = lookup[*knowledge.Index](inj, "knowledge index"); err != nil {
		return Services{}, err
	}
	if svc.Templates, err = lookup[templates.Set](inj, "templates"); err != nil {
		return Services{}, err
	}
	if svc.Renderer, err = lookup[*render.Renderer](inj, "renderer"); err != nil {
		return Services{}, err
	}
	if svc.Targets, err = lookup[*host.Registry](inj, "target registry"); err != nil {
		return Services{}, err
	}
	svc.Loader = sync.OnceValue(build.newLoader)
	svc.OpenRepository = deferred[*vcs.Repository](inj)
	svc.OpenLogger = func() (*slog.Logger, io.Closer, error) {
		built, err := remy.Get[logging](inj)
		return built.logger, built.closer, err
	}
	svc.LoadCatalog = deferred[*emoji.Catalog](inj)
	return svc, nil
}

func lookup[Service any](inj remy.Injector, what string) (Service, error) {
	value, err := remy.Get[Service](inj)
	if err != nil {
		return value, fmt.Errorf("get %s: %w", what, err)
	}
	return value, nil
}

func deferred[Service any](inj remy.Injector) func() (Service, error) {
	return func() (Service, error) { return remy.Get[Service](inj) }
}
