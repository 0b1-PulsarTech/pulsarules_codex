package bootstrap

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/obs"
)

// why: the *slog.Logger and its io.Closer share one lazy construction so
// obs.New runs at most once, and only when the hook command actually
// resolves one of them - a malformed --log-level then never fails a command
// that never logs.
type obsBundle struct {
	logger *slog.Logger
	closer io.Closer
}

func registerObs(inj remy.Injector, opts Options) {
	remy.RegisterConstructorErr(inj, remy.LazySingleton[obsBundle], func() (obsBundle, error) {
		logger, closer, err := obs.New(obs.Config{Level: opts.LogLevel, Path: opts.LogPath})
		if err != nil {
			return obsBundle{}, fmt.Errorf("build logger: %w", err)
		}
		return obsBundle{logger: logger, closer: closer}, nil
	})
	remy.RegisterConstructorArgs1Err(inj, remy.Factory[*slog.Logger],
		func(bundle obsBundle) (*slog.Logger, error) { return bundle.logger, nil },
	)
	remy.RegisterConstructorArgs1Err(inj, remy.Factory[io.Closer],
		func(bundle obsBundle) (io.Closer, error) { return bundle.closer, nil },
	)
}
