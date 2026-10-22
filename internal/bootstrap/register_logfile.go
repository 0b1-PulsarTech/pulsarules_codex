package bootstrap

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/logfile"
)

func registerLogfile(inj remy.Injector, opts Options) {
	// One lazy bundle, so obs.New runs at most once and only for a command that logs.
	remy.RegisterConstructorErr(inj, remy.LazySingleton[Logging], func() (Logging, error) {
		logger, closer, err := logfile.New(logfile.Config{Level: opts.LogLevel, Path: opts.LogPath})
		if err != nil {
			return Logging{}, fmt.Errorf("build logger: %w", err)
		}
		return Logging{Logger: logger, Closer: closer}, nil
	})
	remy.RegisterConstructorArgs1Err(inj, remy.Factory[*slog.Logger],
		func(bundle Logging) (*slog.Logger, error) { return bundle.Logger, nil },
	)
	remy.RegisterConstructorArgs1Err(inj, remy.Factory[io.Closer],
		func(bundle Logging) (io.Closer, error) { return bundle.Closer, nil },
	)
}
