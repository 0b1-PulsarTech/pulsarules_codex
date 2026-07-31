package bootstrap

import (
	"fmt"

	"github.com/wrapped-owls/goremy-di/remy"
)

// DoInjections registers every collaborator the CLI commands resolve:
// knowledge (index, templates, renderer) and vcs (the repository factory) as
// one layer, obs (the hook logger and its closer) as another, and the
// install/target registries as a third. It is the ONE composition-root
// switchboard; nothing outside it, or a command handler's first lines, reads
// inj.
func DoInjections(inj remy.Injector, opts Options) error {
	if err := registerKnowledge(inj, opts); err != nil {
		return fmt.Errorf("register knowledge: %w", err)
	}
	registerVCS(inj, opts)
	registerObs(inj, opts)
	registerRegistries(inj)
	return nil
}
