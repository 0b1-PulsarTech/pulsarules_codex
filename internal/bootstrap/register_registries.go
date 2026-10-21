package bootstrap

import (
	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
)

func registerRegistries(inj remy.Injector) {
	remy.RegisterConstructor(inj, remy.Singleton[*host.Registry], host.NewRegistry)
}
