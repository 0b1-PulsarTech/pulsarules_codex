package bootstrap

import (
	"fmt"

	"github.com/wrapped-owls/goremy-di/remy"
)

func New(opts Options) (Services, error) {
	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := doInjections(inj, opts); err != nil {
		return Services{}, err
	}
	return resolve(inj)
}

func doInjections(inj remy.Injector, opts Options) error {
	if err := registerKnowledge(inj, opts); err != nil {
		return fmt.Errorf("register knowledge: %w", err)
	}
	registerVCS(inj, opts)
	registerLogfile(inj, opts)
	registerRegistries(inj)
	registerEmoji(inj)
	return nil
}
