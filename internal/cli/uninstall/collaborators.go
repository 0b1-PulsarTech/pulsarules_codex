package uninstall

import (
	"fmt"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host"
)

type uninstallCollaborators struct {
	targets *host.Registry
}

func resolveUninstallCollaborators(inj remy.Injector) (uninstallCollaborators, error) {
	targets, err := remy.Get[*host.Registry](inj)
	if err != nil {
		return uninstallCollaborators{}, fmt.Errorf("get target registry: %w", err)
	}
	return uninstallCollaborators{targets: targets}, nil
}
