package dartsyntax

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

var errExeGone = errors.New("dart helper exe vanished")

func summarizeWith(
	ctx context.Context,
	requests []request,
	prepare func(context.Context) (string, error),
) (map[string]*syntax.File, error) {
	files, err := attempt(ctx, requests, prepare)
	// A rival process may prune the exe before or while it runs.
	if errors.Is(err, errExeGone) {
		return attempt(ctx, requests, prepare)
	}
	return files, err
}

func attempt(
	ctx context.Context,
	requests []request,
	prepare func(context.Context) (string, error),
) (map[string]*syntax.File, error) {
	const runTimeout = 30 * time.Second
	exe, err := prepare(ctx)
	if err != nil {
		return nil, err
	}
	runCtx, cancelRun := context.WithTimeout(ctx, runTimeout)
	defer cancelRun()
	files, err := runHelper(runCtx, exe, "", requests)
	if err != nil && runCtx.Err() == nil && !fileExists(exe) {
		return nil, fmt.Errorf("%w: %w", errExeGone, err)
	}
	return files, err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
