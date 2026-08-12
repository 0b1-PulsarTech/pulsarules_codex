package uninstall

import (
	"fmt"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/target"
)

type uninstallCollaborators struct {
	targets *target.Registry
	hooks   *install.Registry
}

func resolveUninstallCollaborators(inj remy.Injector) (uninstallCollaborators, error) {
	targets, err := remy.Get[*target.Registry](inj)
	if err != nil {
		return uninstallCollaborators{}, fmt.Errorf("get target registry: %w", err)
	}
	hooks, err := remy.Get[*install.Registry](inj)
	if err != nil {
		return uninstallCollaborators{}, fmt.Errorf("get install registry: %w", err)
	}
	return uninstallCollaborators{targets: targets, hooks: hooks}, nil
}
