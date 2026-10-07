package dartsyntax

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/execx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

var errBuildBound = errors.New("dart helper build exceeded its time bound")

func exeName() string {
	const exeStem = "summary"
	if runtime.GOOS == "windows" {
		return exeStem + ".exe"
	}
	return exeStem
}

func ensureHelper(ctx context.Context, base, dart string) (string, error) {
	version, err := sdkVersion(ctx, dart)
	if err != nil {
		return "", err
	}
	hash, err := helperHash()
	if err != nil {
		return "", err
	}
	dir := cacheDir(base, version, hash)
	exe := filepath.Join(dir, exeName())
	if exeExists(dir) {
		return exe, nil
	}
	if recent := recentFailure(dir); recent != nil {
		return "", recent
	}
	if buildErr := buildHelper(ctx, dart, dir); buildErr != nil {
		return "", rememberFailure(ctx, dir, buildErr)
	}
	// Tidying is hygiene; a leftover file must not fail a good build.
	_ = clearFailure(dir)
	return exe, nil
}

func rememberFailure(ctx context.Context, dir string, buildErr error) error {
	if ctx.Err() != nil && !errors.Is(context.Cause(ctx), errBuildBound) {
		return buildErr
	}
	if markErr := recordFailure(dir, buildErr); markErr != nil {
		return errors.Join(buildErr, markErr)
	}
	return buildErr
}

func buildHelper(ctx context.Context, dart, dir string) error {
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, fsperm.DirPrivate); err != nil {
		return fmt.Errorf("create helper cache: %w", err)
	}
	work, err := os.MkdirTemp(parent, "build-*")
	if err != nil {
		return fmt.Errorf("create helper build dir: %w", err)
	}
	buildErr := compileAndPublish(ctx, dart, work, dir)
	if cleanErr := os.RemoveAll(work); cleanErr != nil {
		return errors.Join(buildErr, fmt.Errorf("remove helper build dir: %w", cleanErr))
	}
	return buildErr
}

func compileAndPublish(ctx context.Context, dart, work, dir string) error {
	if err := materialise(work); err != nil {
		return err
	}
	tool := toolchain{dir: work, dart: dart}
	if err := pubGet(ctx, tool); err != nil {
		return err
	}
	if err := runStep(ctx, tool, "compile exe", "compile", "exe",
		filepath.Join("bin", "summary.dart"), "-o", exeName()); err != nil {
		return err
	}
	// A concurrent build may have won the rename; its exe is equivalent.
	return publish(work, dir)
}

func pubGet(ctx context.Context, tool toolchain) error {
	offlineErr := runStep(ctx, tool, "pub get offline", "pub", "get", "--offline")
	if offlineErr == nil {
		return nil
	}
	if ctx.Err() != nil {
		return offlineErr
	}
	onlineErr := runStep(ctx, tool, "pub get", "pub", "get")
	if onlineErr == nil {
		return nil
	}
	return errors.Join(offlineErr, onlineErr)
}

type toolchain struct {
	dir  string
	dart string
}

func runStep(ctx context.Context, tool toolchain, step string, args ...string) error {
	command := execx.Command{Name: tool.dart, Args: dartArgs(args), Dir: tool.dir}
	result, err := execx.Run(ctx, command)
	if err != nil {
		return stepError(ctx, step, result, err)
	}
	return nil
}
