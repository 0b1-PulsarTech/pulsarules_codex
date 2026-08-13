package bootstrap

import (
	"fmt"

	"github.com/wrapped-owls/goremy-di/remy"
)

// DoInjections is the ONE composition-root switchboard for every CLI
// collaborator (knowledge, vcs, obs, install/target registries); nothing
// outside it, or a command handler's first lines, reads inj.
func DoInjections(inj remy.Injector, opts Options) error {
	if err := registerKnowledge(inj, opts); err != nil {
		return fmt.Errorf("register knowledge: %w", err)
	}
	registerVCS(inj, opts)
	registerObs(inj, opts)
	registerRegistries(inj)
	return nil
}
