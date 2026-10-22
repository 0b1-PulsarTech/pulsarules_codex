package bootstrap

import (
	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

type resolvedProjectDir string

func registerVCS(inj remy.Injector, opts Options, build builders) {
	remy.RegisterInstance(inj, resolvedProjectDir(opts.ProjectDir))
	// A factory, not a singleton: it fails when the directory is not a git repo, and the
	// caller decides whether that is fatal.
	remy.RegisterConstructorArgs1Err(inj, remy.Factory[*vcs.Repository],
		func(dir resolvedProjectDir) (*vcs.Repository, error) {
			return build.openRepository(string(dir))
		},
	)
}
