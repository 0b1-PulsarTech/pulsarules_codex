package bootstrap

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/loader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/logfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

type builders struct {
	openRepository func(dir string) (*vcs.Repository, error)
	newCatalog     func() (*emoji.Catalog, error)
	newLoader      func(userConfigDir string) *loader.Layered
	newLogger      func(cfg logfile.Config) (*slog.Logger, io.Closer, error)
}

func New(opts Options) (Services, error) {
	return newServices(opts, defaultBuilders())
}

func defaultBuilders() builders {
	return builders{
		openRepository: vcs.Open,
		newCatalog:     emoji.NewCatalog,
		newLoader: func(userConfigDir string) *loader.Layered {
			return loader.New(registry.ConfigRules(), userConfigDir)
		},
		newLogger: logfile.New,
	}
}

func newServices(opts Options, build builders) (Services, error) {
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := doInjections(inj, opts, build); err != nil {
		return Services{}, err
	}
	return resolve(inj, build, opts.UserConfigDir)
}

func doInjections(inj remy.Injector, opts Options, build builders) error {
	if err := registerKnowledge(inj, opts); err != nil {
		return fmt.Errorf("register knowledge: %w", err)
	}
	registerVCS(inj, opts, build)
	registerLogfile(inj, opts, build)
	registerRegistries(inj)
	registerEmoji(inj, build)
	return nil
}
