package bootstrap

import (
	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

// why: a named type carries Options.ProjectDir into the constructor below as
// a concrete dependency, instead of the constructor reaching back into
// Options.
type resolvedProjectDir string

// why: vcs.Repository is a factory (not a singleton) because it depends on
// the resolved project directory and can legitimately fail when that
// directory is not a git repository; the command handler that resolves it
// decides whether that failure is fatal.
func registerVCS(inj remy.Injector, opts Options) {
	remy.RegisterInstance(inj, resolvedProjectDir(opts.ProjectDir))
	remy.RegisterConstructorArgs1Err(inj, remy.Factory[vcs.Repository],
		func(dir resolvedProjectDir) (vcs.Repository, error) {
			return vcs.Open(string(dir))
		},
	)
}
