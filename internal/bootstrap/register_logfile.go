package bootstrap

import (
	"fmt"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/logfile"
)

func registerLogfile(inj remy.Injector, opts Options) {
	// Lazy, so obs.New runs at most once and only for a command that logs.
	remy.RegisterConstructorErr(inj, remy.LazySingleton[Logging], func() (Logging, error) {
		logger, closer, err := logfile.New(logfile.Config{Level: opts.LogLevel, Path: opts.LogPath})
		if err != nil {
			return Logging{}, fmt.Errorf("build logger: %w", err)
		}
		return Logging{Logger: logger, Closer: closer}, nil
	})
}
