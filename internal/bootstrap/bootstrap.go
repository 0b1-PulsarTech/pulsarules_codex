package bootstrap

import (
	"fmt"

	"github.com/wrapped-owls/goremy-di/remy"
)

func DoInjections(inj remy.Injector, opts Options) error {
	if err := registerKnowledge(inj, opts); err != nil {
		return fmt.Errorf("register knowledge: %w", err)
	}
	registerVCS(inj, opts)
	registerObs(inj, opts)
	registerRegistries(inj)
	return nil
}
