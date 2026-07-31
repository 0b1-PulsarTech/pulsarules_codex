package bootstrap

import (
	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/target"
)

func registerRegistries(inj remy.Injector) {
	remy.RegisterConstructor(inj, remy.Singleton[*install.Registry], install.NewRegistry)
	remy.RegisterConstructor(inj, remy.Singleton[*target.Registry], target.NewRegistry)
}
