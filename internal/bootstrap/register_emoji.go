package bootstrap

import (
	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

func registerEmoji(inj remy.Injector, build builders) {
	// Lazy, so a command that never lints a commit does not parse the catalog.
	remy.RegisterConstructorErr(inj, remy.LazySingleton[*emoji.Catalog], build.newCatalog)
}
