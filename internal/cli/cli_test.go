package cli

import (
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
)

func TestRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	err := Run(inj, &version.Options{Command: "frobnicate"})
	if err == nil {
		t.Fatal("expected an error for an unknown command")
	}
}
