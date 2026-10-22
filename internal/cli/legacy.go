package cli

import (
	"flag"
	"io"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
)

type legacyRunner struct {
	name string
	opts *version.Options
	run  handler
}

func (legacy legacyRunner) Bind(*flag.FlagSet) {}

func (legacy legacyRunner) Run(inj remy.Injector, _ []string) error {
	return legacy.run(inj, legacy.opts)
}

func (legacy legacyRunner) ProjectDir() string { return resolveProjectDir(legacy.opts) }

func (legacy legacyRunner) BootstrapFailed(out io.Writer, err error) int {
	return handleBootstrapErr(out, legacy.name, err)
}
