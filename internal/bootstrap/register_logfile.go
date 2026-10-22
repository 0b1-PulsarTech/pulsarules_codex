package bootstrap

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/logfile"
)

type logging struct {
	logger *slog.Logger
	closer io.Closer
}

func registerLogfile(inj remy.Injector, opts Options, build builders) {
	// Lazy, so the logger runs at most once and only for a command that logs.
	remy.RegisterConstructorErr(inj, remy.LazySingleton[logging], func() (logging, error) {
		logger, closer, err := build.newLogger(
			logfile.Config{Level: opts.LogLevel, Path: opts.LogPath},
		)
		if err != nil {
			return logging{}, fmt.Errorf("build logger: %w", err)
		}
		return logging{logger: logger, closer: closer}, nil
	})
}
